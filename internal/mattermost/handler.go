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

func StartWebSocketListener(cfg *config.Config, gc *gitea.Client) {
	mmClient := model.NewAPIv4Client(cfg.MattermostURL)
	mmClient.SetToken(cfg.MattermostBotToken)
	bot, _, _ := mmClient.GetMe("")

	wsURL := strings.Replace(cfg.MattermostURL, "http", "ws", 1)
	wsClient, _ := model.NewWebSocketClient4(wsURL, cfg.MattermostBotToken)
	wsClient.Listen()

	log.Println("Listening for Mattermost events...")

	for event := range wsClient.EventChannel {
		if event.EventType() != model.WebsocketEventPosted {
			continue
		}
		var post model.Post
		json.Unmarshal([]byte(event.GetData()["post"].(string)), &post)

		// Input Validation: Only trigger on explicit commands
		msg := strings.TrimSpace(strings.ToLower(post.Message))
		if !strings.HasPrefix(msg, "/ticket") && !strings.HasPrefix(msg, "/assign") && !strings.HasPrefix(msg, "/addign") {
			continue
		}

		// Avoid self-replies and ensure we are in a thread
		if post.UserId == bot.Id || post.RootId == "" {
			continue
		}

		parent, _, _ := mmClient.GetPost(post.RootId, "")
		a, c := extractLabel(parent.Message, "alertname"), extractLabel(parent.Message, "certname")
		if a == "" || c == "" {
			continue
		}

		log.Printf("Event caught. Extracted Labels -> Alert: %s, Cert: %s", a, c)

		// Fetch the full issue object from Gitea including Labels and Assignees
		issue, err := gc.FindIssueByLabels(a, c)
		if err != nil {
			log.Printf("Fuzzy search failed: %v", err)
			continue
		}

		// Filter Gitea Labels to remove "Incident" and "Time To Handle"
		var labelNames []string
		for _, lbl := range issue.Labels {
			name := lbl.Name
			// Check if the label should be excluded
			if strings.EqualFold(name, "Incident") || strings.Contains(strings.ToLower(name), "time to handle") {
				continue
			}
			labelNames = append(labelNames, "`"+name+"`")
		}

		currentLabels := "None"
		if len(labelNames) > 0 {
			currentLabels = strings.Join(labelNames, ", ")
		}

		// Parse Assignees
		assigneeNames := "Unassigned"
		if len(issue.Assignees) > 0 {
			var names []string
			for _, user := range issue.Assignees {
				names = append(names, user.Username)
			}
			assigneeNames = strings.Join(names, ", ")
		}

		// Construct the detailed Mattermost status report
		issueURL := fmt.Sprintf("%s/%s/%s/issues/%d", cfg.GiteaURL, cfg.GiteaOwner, cfg.GiteaRepo, issue.Number)
		statusEmoji := "🟢"
		if issue.State == "closed" {
			statusEmoji = "🔴"
		}

		statusMsg := fmt.Sprintf(":round_pushpin: **Matched Gitea Ticket:** %s\n**Status:** %s %s\n**Alert Labels:** %s\n**Assigned to:** %s",
			issueURL, statusEmoji, strings.Title(issue.State), currentLabels, assigneeNames)

		log.Printf("Found match! #%d | Status: %s | Labels: %s | Assignees: %s", issue.Number, issue.State, currentLabels, assigneeNames)

		// Post the detailed ticket summary to the thread
		reply(mmClient, &post, statusMsg)

		processAction(gc, mmClient, &post, issue.Number, issueURL)
	}
}

func processAction(gc *gitea.Client, mm *model.Client4, post *model.Post, num int, url string) {
	raw := strings.TrimSpace(post.Message)
	low := strings.ToLower(raw)

	if low == "/ticket" {
		return
	}

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

	// NEW: Feature: /ticket [comment] with user attribution
	// Feature: /ticket [comment] with user attribution
	if strings.HasPrefix(low, "/ticket ") {
		commentText := strings.TrimSpace(raw[8:])
		if commentText != "" {
			// 1. Fetch the Mattermost user who typed the command
			user, _, err := mm.GetUser(post.UserId, "")
			username := "Unknown User"
			if err == nil {
				username = user.Username
			}

			// 2. Format the timestamp
			postTime := time.UnixMilli(post.CreateAt).Format("2006-01-02 15:04:05")

			// 3. Construct the attributed comment string for Gitea
			attributedComment := fmt.Sprintf("🗣 **@%s** commented via Mattermost at `%s`:\n\n> %s",
				username, postTime, commentText)

			// 4. Send the formatted comment to Gitea
			if err := gc.AddComment(num, attributedComment); err == nil {
				log.Printf("Successfully added attributed comment to issue #%d on behalf of %s", num, username)

				// UPDATE: Reply with the actual comment text in Mattermost
				successMsg := fmt.Sprintf("✅ *Synced to Gitea:*\n> %s", commentText)
				reply(mm, post, successMsg)
			} else {
				log.Printf("Failed to add comment to #%d: %v", num, err)
			}
		}
	}
}

func reply(mm *model.Client4, post *model.Post, msg string) {
	mm.CreatePost(&model.Post{
		ChannelId: post.ChannelId,
		RootId:    post.RootId,
		Message:   msg,
	})
}
