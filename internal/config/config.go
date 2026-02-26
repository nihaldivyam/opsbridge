package config

import "os"

type Config struct {
	GiteaURL           string
	GiteaToken         string
	GiteaOwner         string
	GiteaRepo          string
	MattermostURL      string
	MattermostBotToken string

	// Optional Basic Auth for Gitea Proxy
	GiteaBasicUser string
	GiteaBasicPass string

	// Dynamic routing configuration
	QAChannelName    string
	QAChannelTags    string
	DefaultAlertTags string
}

func Load() *Config {
	return &Config{
		GiteaURL:           os.Getenv("GITEA_URL"),
		GiteaToken:         os.Getenv("GITEA_TOKEN"),
		GiteaOwner:         os.Getenv("GITEA_OWNER"),
		GiteaRepo:          os.Getenv("GITEA_REPO"),
		MattermostURL:      os.Getenv("MATTERMOST_URL"),
		MattermostBotToken: os.Getenv("MATTERMOST_BOT_TOKEN"),

		// Proxy Auth
		GiteaBasicUser: os.Getenv("GITEA_BASIC_USER"),
		GiteaBasicPass: os.Getenv("GITEA_BASIC_PASS"),

		// Load from env, but provide safe fallbacks
		QAChannelName:    getEnv("QA_CHANNEL_NAME", "kilroy-alerts-qa"),
		QAChannelTags:    getEnv("QA_CHANNEL_TAGS", "@hritik @sidharth @smahar @franco-david @yulian @nitish0 @olayinka @kanha @onkar @gaurav"),
		DefaultAlertTags: getEnv("DEFAULT_ALERT_TAGS", "@operation"),
	}
}

// Helper function to provide default values if the env var is missing
func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
