package main

import (
	"log"
	"net/http"

	"github.com/nihaldivyam/opsbridge/internal/config"
	"github.com/nihaldivyam/opsbridge/internal/gitea"
	"github.com/nihaldivyam/opsbridge/internal/mattermost"
)

func main() {
	cfg := config.LoadConfig()
	giteaClient := gitea.NewClient(cfg)

	http.HandleFunc("/webhook", mattermost.HandleWebhook(cfg, giteaClient))

	log.Printf("Starting opsbridge bot on port %s...", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
