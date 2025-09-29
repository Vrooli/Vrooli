package main

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"
)

type config struct {
	addr                string
	cliPath             string
	sessionTTL          time.Duration
	idleTimeout         time.Duration
	storagePath         string
	enableProxyGuard    bool
	maxConcurrent       int
	panicKillGrace      time.Duration
	readBufferSizeBytes int
}

func loadConfig() (config, error) {
	port := os.Getenv("API_PORT")
	if port == "" {
		return config{}, errors.New("API_PORT must be provided (no default)")
	}

	host := os.Getenv("API_HOST")
	if host == "" {
		host = "0.0.0.0"
	}

	c := config{
		addr:                fmt.Sprintf("%s:%s", host, port),
		cliPath:             firstNonEmpty(os.Getenv("CODEX_CONSOLE_CLI_PATH"), "codex"),
		sessionTTL:          parseDurationOrDefault(os.Getenv("CODEX_CONSOLE_SESSION_TTL"), 30*time.Minute),
		idleTimeout:         parseDurationOrDefault(os.Getenv("CODEX_CONSOLE_IDLE_TIMEOUT"), 5*time.Minute),
		storagePath:         firstNonEmpty(os.Getenv("CODEX_CONSOLE_STORAGE_PATH"), "data/sessions"),
		enableProxyGuard:    parseBoolOrDefault(os.Getenv("CODEX_CONSOLE_EXPECT_PROXY"), true),
		maxConcurrent:       parseIntOrDefault(os.Getenv("CODEX_CONSOLE_MAX_CONCURRENT"), 4),
		panicKillGrace:      parseDurationOrDefault(os.Getenv("CODEX_CONSOLE_PANIC_GRACE"), 3*time.Second),
		readBufferSizeBytes: parseIntOrDefault(os.Getenv("CODEX_CONSOLE_READ_BUFFER"), 4096),
	}

	if c.sessionTTL <= 0 {
		return config{}, errors.New("CODEX_CONSOLE_SESSION_TTL must be > 0")
	}
	if c.idleTimeout <= 0 {
		return config{}, errors.New("CODEX_CONSOLE_IDLE_TIMEOUT must be > 0")
	}
	if c.maxConcurrent <= 0 {
		return config{}, errors.New("CODEX_CONSOLE_MAX_CONCURRENT must be > 0")
	}
	if c.panicKillGrace < 500*time.Millisecond {
		return config{}, errors.New("CODEX_CONSOLE_PANIC_GRACE must be >= 500ms")
	}
	if c.readBufferSizeBytes < 512 {
		return config{}, errors.New("CODEX_CONSOLE_READ_BUFFER must be >= 512 bytes")
	}

	return c, nil
}

func parseDurationOrDefault(raw string, fallback time.Duration) time.Duration {
	if raw == "" {
		return fallback
	}
	d, err := time.ParseDuration(raw)
	if err != nil {
		return fallback
	}
	return d
}

func parseIntOrDefault(raw string, fallback int) int {
	if raw == "" {
		return fallback
	}
	i, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return i
}

func parseBoolOrDefault(raw string, fallback bool) bool {
	switch raw {
	case "", "default":
		return fallback
	case "1", "true", "TRUE", "True", "yes", "YES", "Yes":
		return true
	case "0", "false", "FALSE", "False", "no", "NO", "No":
		return false
	default:
		return fallback
	}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
