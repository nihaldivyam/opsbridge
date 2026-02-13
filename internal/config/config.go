package config

import (
	"log"
	"os"
)

type Config struct {
	GiteaURL        string
	GiteaToken      string
	GiteaOwner      string
	GiteaRepo       string
	MattermostToken string
	Port            string
}

func LoadConfig() *Config {
	cfg := &Config{
		GiteaURL:        os.Getenv("GITEA_URL"),
		GiteaToken:      os.Getenv("GITEA_TOKEN"),
		GiteaOwner:      os.Getenv("GITEA_OWNER"),
		GiteaRepo:       os.Getenv("GITEA_REPO"),
		MattermostToken: os.Getenv("MATTERMOST_TOKEN"),
		Port:            os.Getenv("PORT"),
	}

	if cfg.Port == "" {
		cfg.Port = "8080"
	}

	if cfg.GiteaURL == "" || cfg.GiteaToken == "" || cfg.MattermostToken == "" {
		log.Fatal("Missing required environment variables (GITEA_URL, GITEA_TOKEN, MATTERMOST_TOKEN)")
	}

	return cfg
}
