package mattermost

import (
	"encoding/json"
	"log"
	"net/http"
	"regexp"
	"strings"

	"github.com/nihaldivyam/opsbridge/internal/config"
	"github.com/nihaldivyam/opsbridge/internal/gitea"
)

type WebhookPayload struct {
	Token    string `json:"token"`
	Text     string `json:"text"`
	UserName string `json:"user_name"`
}

func extractHash(text string) string {
	re := regexp.MustCompile(`\[([a-f0-9]{32})\]`)
	matches := re.FindStringSubmatch(text)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

func HandleWebhook(cfg *config.Config, gc *gitea.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var payload WebhookPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		if payload.Token != cfg.MattermostToken {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Ignore messages from the bot itself to prevent infinite loops
		if payload.UserName == "bot_opsmondo" {
			w.WriteHeader(http.StatusOK)
			return
		}

		hash := extractHash(payload.Text)
		if hash == "" {
			w.WriteHeader(http.StatusOK)
			return
		}

		issueNumber, err := gc.FindIssueByHash(hash)
		if err != nil {
			log.Printf("Issue not found for hash %s: %v", hash, err)
			http.Error(w, "Issue not found", http.StatusNotFound)
			return
		}

		processAction(gc, issueNumber, payload.Text)
		w.WriteHeader(http.StatusOK)
	}
}

func processAction(gc *gitea.Client, issueNumber int, text string) {
	replyText := strings.TrimSpace(text)

	if strings.HasPrefix(replyText, "/spend") {
		parts := strings.SplitN(replyText, " ", 3)
		if len(parts) >= 2 {
			if err := gc.AddTimeReg(issueNumber, parts[1]); err != nil {
				log.Printf("Failed to register time: %v", err)
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
		}
	}
}
