package config

import "os"

type Config struct {
	ResourceName, Package, MCPServer, MCPEndpoint string
	MaxTurns, TimeoutSeconds                      int
}

func Defaults() Config {
	return Config{ResourceName: "claude-code", Package: "@anthropic-ai/claude-code", MCPServer: "vrooli-local", MCPEndpoint: "/mcp/sse", MaxTurns: 5, TimeoutSeconds: 600}
}

func (c Config) WithEnvironment() Config {
	if v := os.Getenv("CLAUDE_MAX_TURNS"); v != "" {
		c.MaxTurns = atoi(v, c.MaxTurns)
	}
	if v := os.Getenv("CLAUDE_TIMEOUT"); v != "" {
		c.TimeoutSeconds = atoi(v, c.TimeoutSeconds)
	}
	return c
}

func atoi(value string, fallback int) int {
	n := 0
	for _, r := range value {
		if r < '0' || r > '9' {
			return fallback
		}
		n = n*10 + int(r-'0')
	}
	if n == 0 {
		return fallback
	}
	return n
}
