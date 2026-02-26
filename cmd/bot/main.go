package main

import (
	"log"

	"github.com/nihaldivyam/opsbridge/internal/config"
	"github.com/nihaldivyam/opsbridge/internal/gitea"
	"github.com/nihaldivyam/opsbridge/internal/mattermost"
)

func main() {
	cfg := config.Load() // Ensure your config struct now has MattermostURL and MattermostBotToken
	giteaClient := gitea.NewClient(cfg)

	log.Println("Starting opsbridge bot via Mattermost WebSocket...")

	// Start listening to Mattermost. This will block and run continuously.
	mattermost.StartWebSocketListener(cfg, giteaClient)
}
