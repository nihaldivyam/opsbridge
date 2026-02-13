package config

import (
	"log"
	"os"
)

type Config struct {
	GiteaURL           string
	GiteaToken         string
	GiteaOwner         string
	GiteaRepo          string
	GiteaBasicUser     string // NEW
	GiteaBasicPass     string // NEW
	MattermostURL      string
	MattermostBotToken string
}

func LoadConfig() *Config {
	cfg := &Config{
		GiteaURL:           os.Getenv("GITEA_URL"),
		GiteaToken:         os.Getenv("GITEA_TOKEN"),
		GiteaOwner:         os.Getenv("GITEA_OWNER"),
		GiteaRepo:          os.Getenv("GITEA_REPO"),
		GiteaBasicUser:     os.Getenv("GITEA_BASIC_USER"), // NEW
		GiteaBasicPass:     os.Getenv("GITEA_BASIC_PASS"), // NEW
		MattermostURL:      os.Getenv("MATTERMOST_URL"),
		MattermostBotToken: os.Getenv("MATTERMOST_BOT_TOKEN"),
	}

	if cfg.GiteaURL == "" || cfg.GiteaToken == "" || cfg.MattermostURL == "" || cfg.MattermostBotToken == "" {
		log.Fatal("Missing required environment variables. Please check GITEA_URL, GITEA_TOKEN, MATTERMOST_URL, and MATTERMOST_BOT_TOKEN.")
	}

	return cfg
}
