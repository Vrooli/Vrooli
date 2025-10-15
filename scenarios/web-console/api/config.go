package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type config struct {
	addr                string
	defaultCommand      string
	defaultArgs         []string
	sessionTTL          time.Duration
	idleTimeout         time.Duration
	storagePath         string
	enableProxyGuard    bool
	maxConcurrent       int
	panicKillGrace      time.Duration
	readBufferSizeBytes int
	defaultTTYRows      int
	defaultTTYCols      int
	defaultWorkingDir   string
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

	defaultCommand := firstNonEmpty(
		os.Getenv("WEB_CONSOLE_DEFAULT_COMMAND"),
		os.Getenv("SHELL"),
		"/bin/bash",
	)
	defaultCommand = strings.TrimSpace(defaultCommand)
	defaultArgs := parseArgs(os.Getenv("WEB_CONSOLE_DEFAULT_ARGS"))

	workingDir, err := resolveWorkingDir()
	if err != nil {
		return config{}, err
	}

	c := config{
		addr:                fmt.Sprintf("%s:%s", host, port),
		defaultCommand:      defaultCommand,
		defaultArgs:         defaultArgs,
		sessionTTL:          parseDurationOrDefault(os.Getenv("WEB_CONSOLE_SESSION_TTL"), 30*time.Minute),
		idleTimeout:         parseDurationOrDefault(os.Getenv("WEB_CONSOLE_IDLE_TIMEOUT"), 5*time.Minute),
		storagePath:         firstNonEmpty(os.Getenv("WEB_CONSOLE_STORAGE_PATH"), "data/sessions"),
		enableProxyGuard:    parseBoolOrDefault(os.Getenv("WEB_CONSOLE_EXPECT_PROXY"), true),
		maxConcurrent:       parseIntOrDefault(os.Getenv("WEB_CONSOLE_MAX_CONCURRENT"), 20),
		panicKillGrace:      parseDurationOrDefault(os.Getenv("WEB_CONSOLE_PANIC_GRACE"), 3*time.Second),
		readBufferSizeBytes: parseIntOrDefault(os.Getenv("WEB_CONSOLE_READ_BUFFER"), 4096),
		defaultTTYRows:      parseIntOrDefault(os.Getenv("WEB_CONSOLE_TTY_ROWS"), 32),
		defaultTTYCols:      parseIntOrDefault(os.Getenv("WEB_CONSOLE_TTY_COLS"), 120),
		defaultWorkingDir:   workingDir,
	}

	if c.defaultCommand == "" {
		return config{}, errors.New("WEB_CONSOLE_DEFAULT_COMMAND (or equivalent) must be provided")
	}

	if c.sessionTTL <= 0 {
		return config{}, errors.New("WEB_CONSOLE_SESSION_TTL must be > 0")
	}
	if c.idleTimeout <= 0 {
		return config{}, errors.New("WEB_CONSOLE_IDLE_TIMEOUT must be > 0")
	}
	if c.maxConcurrent <= 0 {
		return config{}, errors.New("WEB_CONSOLE_MAX_CONCURRENT must be > 0")
	}
	if c.panicKillGrace < 500*time.Millisecond {
		return config{}, errors.New("WEB_CONSOLE_PANIC_GRACE must be >= 500ms")
	}
	if c.readBufferSizeBytes < 512 {
		return config{}, errors.New("WEB_CONSOLE_READ_BUFFER must be >= 512 bytes")
	}
	if c.defaultTTYRows <= 0 {
		return config{}, errors.New("WEB_CONSOLE_TTY_ROWS must be > 0")
	}
	if c.defaultTTYCols <= 0 {
		return config{}, errors.New("WEB_CONSOLE_TTY_COLS must be > 0")
	}

	return c, nil
}

func resolveWorkingDir() (string, error) {
	base := upDirectory(3)
	override := strings.TrimSpace(os.Getenv("WEB_CONSOLE_WORKING_DIR"))

	var candidate string
	if override != "" {
		if filepath.IsAbs(override) {
			candidate = filepath.Clean(override)
		} else {
			candidate = filepath.Clean(filepath.Join(base, override))
		}
	} else {
		candidate = base
	}

	if candidate == "" {
		return "", errors.New("unable to determine working directory")
	}

	info, err := os.Stat(candidate)
	if err != nil {
		return "", fmt.Errorf("resolve working dir: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("resolve working dir: %s is not a directory", candidate)
	}

	return candidate, nil
}

func upDirectory(levels int) string {
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}
	dir := filepath.Clean(cwd)
	for i := 0; i < levels; i++ {
		next := filepath.Dir(dir)
		if next == dir {
			break
		}
		dir = next
	}
	return dir
}

func parseArgs(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	fields := strings.Fields(raw)
	if len(fields) == 0 {
		return nil
	}
	return fields
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
