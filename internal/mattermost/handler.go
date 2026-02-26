package mattermost

import (
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/nihaldivyam/opsbridge/internal/config"
	"github.com/nihaldivyam/opsbridge/internal/gitea"
)

// extractLabel parses Alertmanager labels from raw markdown text
func extractLabel(text, key string) string {
	re := regexp.MustCompile(`(?i)` + key + `:\**\s*\x60?([a-zA-Z0-9_.-]+)\x60?`)
	matches := re.FindStringSubmatch(text)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// buildStatusMessage creates the formatted Mattermost summary from a Gitea issue
func buildStatusMessage(cfg *config.Config, issue gitea.Issue) string {
	var labelNames []string
	for _, lbl := range issue.Labels {
		name := lbl.Name
		if strings.EqualFold(name, "Incident") || strings.Contains(strings.ToLower(name), "time to handle") {
			continue
		}
		labelNames = append(labelNames, "`"+name+"`")
	}
	currentLabels := "None"
	if len(labelNames) > 0 {
		currentLabels = strings.Join(labelNames, ", ")
	}

	assigneeNames := "Unassigned"
	if len(issue.Assignees) > 0 {
		var names []string
		for _, user := range issue.Assignees {
			names = append(names, user.Username)
		}
		assigneeNames = strings.Join(names, ", ")
	}

	issueURL := fmt.Sprintf("%s/%s/%s/issues/%d", cfg.GiteaURL, cfg.GiteaOwner, cfg.GiteaRepo, issue.Number)
	statusEmoji := "🟢"
	if issue.State == "closed" {
		statusEmoji = "🔴"
	}

	return fmt.Sprintf("🔗 **Matched Gitea Ticket:** %s\n**Status:** %s %s\n**Alert Labels:** %s\n**Assigned to:** %s",
		issueURL, statusEmoji, strings.Title(issue.State), currentLabels, assigneeNames)
}

func StartWebSocketListener(cfg *config.Config, gc *gitea.Client) {
	mmClient := model.NewAPIv4Client(cfg.MattermostURL)
	mmClient.SetToken(cfg.MattermostBotToken)

	// Explicitly check if the REST API connection succeeds
	bot, _, err := mmClient.GetMe("")
	if err != nil {
		log.Fatalf("❌ Failed to authenticate with Mattermost REST API: %v", err)
	}

	wsURL := strings.Replace(cfg.MattermostURL, "http", "ws", 1)

	// Check if WebSocket creation succeeds
	wsClient, err := model.NewWebSocketClient4(wsURL, cfg.MattermostBotToken)
	if err != nil {
		log.Fatalf("❌ Failed to create WebSocket client: %v", err)
	}

	// Explicitly connect and catch network errors
	if appErr := wsClient.Connect(); appErr != nil {
		log.Fatalf("❌ Failed to connect to Mattermost WebSocket: %v", appErr)
	}

	wsClient.Listen()
	log.Println("✅ Connected and listening for Mattermost events...")

	for event := range wsClient.EventChannel {
		// Prevent nil pointer panics if the connection drops
		if event == nil {
			log.Println("⚠️ Received nil event, WebSocket channel might be closed.")
			continue
		}

		if event.EventType() != model.WebsocketEventPosted {
			continue
		}
		var post model.Post
		json.Unmarshal([]byte(event.GetData()["post"].(string)), &post)

		// Ignore bot's own messages to prevent infinite loops
		if post.UserId == bot.Id {
			continue
		}

		msg := strings.TrimSpace(strings.ToLower(post.Message))
		isCommand := strings.HasPrefix(msg, "/ticket") || strings.HasPrefix(msg, "/assign") || strings.HasPrefix(msg, "/addign")

		// ==========================================
		// TRACK 1: Auto-Detect New Alerts
		// ==========================================
		if !isCommand {
			a, c := extractLabel(post.Message, "alertname"), extractLabel(post.Message, "certname")
			if a != "" && c != "" {
				log.Printf("New alert detected! Alert: %s, Cert: %s", a, c)

				issue, err := gc.FindIssueByLabels(a, c)
				if err != nil {
					log.Printf("Fuzzy search failed for new alert: %v", err)
					continue
				}

				// Build the status message
				statusMsg := buildStatusMessage(cfg, issue)

				// Determine who to ping based on the channel name
				pingTags := cfg.DefaultAlertTags // Default to the whole operations team
				channel, _, err := mmClient.GetChannel(post.ChannelId, "")
				if err == nil && channel.Name == cfg.QAChannelName {
					pingTags = cfg.QAChannelTags
				}

				// Format it as a human-readable sentence
				finalMsg := fmt.Sprintf("🚨 **Operations Team** (%s) - A new alert has been detected, please look into it!\n\n%s", pingTags, statusMsg)

				// Post the status summary to start the thread
				reply(mmClient, &post, finalMsg)
				log.Printf("Auto-replied with ticket status for issue #%d and pinged %s", issue.Number, pingTags)
			}
			continue
		}

		// ==========================================
		// TRACK 2: Process User Commands in Threads
		// ==========================================
		if isCommand && post.RootId != "" {
			parent, _, _ := mmClient.GetPost(post.RootId, "")
			a, c := extractLabel(parent.Message, "alertname"), extractLabel(parent.Message, "certname")
			if a == "" || c == "" {
				continue
			}

			issue, err := gc.FindIssueByLabels(a, c)
			if err != nil {
				continue
			}

			if msg == "/ticket" {
				// User manually requested a status refresh
				statusMsg := buildStatusMessage(cfg, issue)
				reply(mmClient, &post, statusMsg)
			} else {
				// Process assignments or comments
				processAction(gc, mmClient, &post, issue.Number)
			}
		}
	}
}

func processAction(gc *gitea.Client, mm *model.Client4, post *model.Post, num int) {
	raw := strings.TrimSpace(post.Message)
	low := strings.ToLower(raw)

	if low == "/assignme" {
		user, _, _ := mm.GetUser(post.UserId, "")
		if err := gc.AssignIssue(num, user.Username); err == nil {
			reply(mm, post, "👤 Ticket successfully assigned to **"+user.Username+"**")
		}
		return
	}

	if strings.HasPrefix(low, "/assign @") || strings.HasPrefix(low, "/addign @") {
		parts := strings.Split(raw, " ")
		if len(parts) >= 2 {
			target := strings.TrimPrefix(parts[1], "@")
			if err := gc.AssignIssue(num, target); err == nil {
				reply(mm, post, "👤 Ticket successfully assigned to **@"+target+"**")
			}
		}
		return
	}

	// Feature: /ticket [comment] with user attribution
	if strings.HasPrefix(low, "/ticket ") {
		commentText := strings.TrimSpace(raw[8:])
		if commentText != "" {
			user, _, err := mm.GetUser(post.UserId, "")
			username := "Unknown User"
			if err == nil {
				username = user.Username
			}

			postTime := time.UnixMilli(post.CreateAt).Format("2006-01-02 15:04:05")
			attributedComment := fmt.Sprintf("🗣 **@%s** commented via Mattermost at `%s`:\n\n> %s",
				username, postTime, commentText)

			if err := gc.AddComment(num, attributedComment); err == nil {
				successMsg := fmt.Sprintf("✅ *Synced to Gitea:*\n> %s", commentText)
				reply(mm, post, successMsg)
			}
		}
	}
}

// reply automatically handles threading. If the trigger post is a root post, it uses its Id.
func reply(mm *model.Client4, post *model.Post, msg string) {
	rootId := post.RootId
	if rootId == "" {
		rootId = post.Id // Start a new thread if replying to a top-level alert
	}

	mm.CreatePost(&model.Post{
		ChannelId: post.ChannelId,
		RootId:    rootId,
		Message:   msg,
	})
}
