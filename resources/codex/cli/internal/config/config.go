package config

import "os"

type Config struct {
	Name, Category, Description, Endpoint, Model string
	TimeoutSeconds, RetryCount                   int
}

func Defaults() Config {
	return Config{Name: "codex", Category: "ai", Description: "AI-powered code completion and generation via OpenAI Codex", Endpoint: "https://api.openai.com/v1", Model: "gpt-5-nano", TimeoutSeconds: 30, RetryCount: 3}
}

func (c Config) WithEnvironment() Config {
	if v := os.Getenv("CODEX_API_ENDPOINT"); v != "" {
		c.Endpoint = v
	}
	if v := os.Getenv("CODEX_DEFAULT_MODEL"); v != "" {
		c.Model = v
	}
	return c
}
