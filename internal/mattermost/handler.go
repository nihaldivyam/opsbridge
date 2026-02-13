package mattermost

import (
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/nihaldivyam/opsbridge/internal/config"
	"github.com/nihaldivyam/opsbridge/internal/gitea"
)

func extractLabel(text, labelKey string) string {
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

		alertname := extractLabel(parentPost.Message, "alertname")
		certname := extractLabel(parentPost.Message, "certname")

		if alertname == "" || certname == "" {
			continue
		}

		log.Printf("Extracted labels - Alert: %s, Cert: %s", alertname, certname)

		issueNumber, err := gc.FindIssueByLabels(alertname, certname)
		if err != nil {
			log.Printf("Fuzzy search failed: %v", err)
			continue
		}

		issueURL := fmt.Sprintf("%s/%s/%s/issues/%d", cfg.GiteaURL, cfg.GiteaOwner, cfg.GiteaRepo, issueNumber)
		log.Printf("Found match! Processing action on Gitea issue: %s", issueURL)

		processAction(gc, issueNumber, post.Message)
	}
}

func processAction(gc *gitea.Client, issueNumber int, text string) {
	// Trim spaces from the raw message
	replyText := strings.TrimSpace(text)

	// Convert to lowercase just for the prefix check
	lowerText := strings.ToLower(replyText)

	// If the user typed exactly "/ticket" (or with trailing spaces)
	if lowerText == "/ticket" {
		log.Printf("Utility command detected: Ticket link displayed above. No comment added to Gitea.")
		return // Exit early without calling gc.AddComment
	}

	// If they typed "/ticket something..." we treat "something..." as the comment
	if strings.HasPrefix(lowerText, "/ticket ") {
		replyText = strings.TrimSpace(replyText[7:]) // Strip "/ticket " from the start
	}

	// If the resulting text is empty after stripping /ticket, do nothing
	if replyText == "" {
		return
	}

	// Post the remaining text as a comment
	if err := gc.AddComment(issueNumber, replyText); err != nil {
		log.Printf("Failed to add comment: %v", err)
	} else {
		log.Printf("Successfully added comment to issue #%d", issueNumber)
	}
}
