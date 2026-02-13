package mattermost

import (
	"encoding/json"
	"log"
	"regexp"
	"strings"

	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/nihaldivyam/opsbridge/internal/config"
	"github.com/nihaldivyam/opsbridge/internal/gitea"
)

// extractLabel dynamically finds a label key and returns its value, ignoring markdown.
// It handles formats like: "- **alertname:** `ArgoCdAppOutOfSync`" or "certname: qa-az1..."
func extractLabel(text, labelKey string) string {
	// Regex looks for the label, optional colons/asterisks/spaces, and captures the value without backticks
	re := regexp.MustCompile(`(?i)` + labelKey + `:\**\s*\x60?([a-zA-Z0-9_.-]+)\x60?`)
	matches := re.FindStringSubmatch(text)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

func StartWebSocketListener(cfg *config.Config, gc *gitea.Client) {
	mmClient := model.NewAPIv4Client(cfg.MattermostURL)
	mmClient.SetToken(cfg.MattermostBotToken)

	botUser, _, err := mmClient.GetMe("")
	if err != nil {
		log.Fatalf("Failed to fetch bot user from Mattermost: %v", err)
	}
	log.Printf("Successfully authenticated to Mattermost as Bot User: %s", botUser.Username)

	wsURL := strings.Replace(cfg.MattermostURL, "http", "ws", 1)
	wsClient, err := model.NewWebSocketClient4(wsURL, cfg.MattermostBotToken)
	if err != nil {
		log.Fatalf("Failed to connect to Mattermost WebSocket: %v", err)
	}

	wsClient.Listen()
	log.Println("Listening for Mattermost events...")

	for event := range wsClient.EventChannel {
		if event.EventType() != model.WebsocketEventPosted {
			continue
		}

		postData, ok := event.GetData()["post"].(string)
		if !ok {
			continue
		}

		var post model.Post
		if err := json.Unmarshal([]byte(postData), &post); err != nil {
			continue
		}

		senderName, _ := event.GetData()["sender_name"].(string)

		if post.UserId == botUser.Id || senderName == "bot_opsmondo" {
			continue
		}

		if post.RootId == "" {
			continue
		}

		parentPost, _, err := mmClient.GetPost(post.RootId, "")
		if err != nil {
			log.Printf("Failed to get parent post: %v", err)
			continue
		}

		// --- NEW FUZZY MATCHING LOGIC ---
		alertname := extractLabel(parentPost.Message, "alertname")
		certname := extractLabel(parentPost.Message, "certname")

		if alertname == "" || certname == "" {
			// This thread doesn't look like a standard alert, ignore it
			continue
		}

		log.Printf("Extracted labels - Alert: %s, Cert: %s", alertname, certname)

		issueNumber, err := gc.FindIssueByLabels(alertname, certname)
		if err != nil {
			log.Printf("Fuzzy search failed: %v", err)
			continue
		}

		log.Printf("Found match! Processing action on Gitea issue #%d", issueNumber)
		processAction(gc, issueNumber, post.Message)
	}
}

func processAction(gc *gitea.Client, issueNumber int, text string) {
	replyText := strings.TrimSpace(text)

	if strings.HasPrefix(replyText, "/ticket") {
		parts := strings.SplitN(replyText, " ", 3)
		if len(parts) >= 2 {
			if err := gc.AddTimeReg(issueNumber, parts[1]); err != nil {
				log.Printf("Failed to register time: %v", err)
			} else {
				log.Printf("Successfully added time %s to issue #%d", parts[1], issueNumber)
			}

			if len(parts) == 3 {
				if err := gc.AddComment(issueNumber, parts[2]); err != nil {
					log.Printf("Failed to add time comment: %v", err)
				}
			}
		}
	} else {
		if err := gc.AddComment(issueNumber, replyText); err != nil {
			log.Printf("Failed to add comment: %v", err)
		} else {
			log.Printf("Successfully added comment to issue #%d", issueNumber)
		}
	}
}
