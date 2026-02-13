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
	GiteaBasicUser     string
	GiteaBasicPass     string
	MattermostURL      string
	MattermostBotToken string
}

func LoadConfig() *Config {
	cfg := &Config{
		GiteaURL:           os.Getenv("GITEA_URL"),
		GiteaToken:         os.Getenv("GITEA_TOKEN"),
		GiteaOwner:         os.Getenv("GITEA_OWNER"),
		GiteaRepo:          os.Getenv("GITEA_REPO"),
		GiteaBasicUser:     os.Getenv("GITEA_BASIC_USER"),
		GiteaBasicPass:     os.Getenv("GITEA_BASIC_PASS"),
		MattermostURL:      os.Getenv("MATTERMOST_URL"),
		MattermostBotToken: os.Getenv("MATTERMOST_BOT_TOKEN"),
	}

	if cfg.GiteaURL == "" || cfg.GiteaToken == "" || cfg.MattermostURL == "" || cfg.MattermostBotToken == "" {
		log.Fatal("Missing required environment variables.")
	}

	return cfg
}
