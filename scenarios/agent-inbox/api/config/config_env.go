// Package config provides centralized configuration for the Agent Inbox scenario.
// This file implements environment variable loading helpers.
package config

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/vrooli/api-core/discovery"
)

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getOllamaBaseURL() string {
	if url := os.Getenv("OLLAMA_BASE_URL"); url != "" {
		return url
	}
	port := getEnvOrDefault("OLLAMA_PORT", "11434")
	return fmt.Sprintf("http://localhost:%s", port)
}

func getPromptManagerURL() string {
	// Check for explicit override first (useful for testing)
	if url := os.Getenv("PROMPT_MANAGER_URL"); url != "" {
		return url
	}

	// Use api-core discovery to resolve prompt-manager URL
	resolver := discovery.NewResolver(discovery.ResolverConfig{})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	url, err := resolver.ResolveScenarioURLDefault(ctx, "prompt-manager")
	if err != nil {
		// Discovery failed - return empty, will be handled by PromptSyncService
		log.Printf("Prompt sync: prompt-manager discovery failed: %v", err)
		return ""
	}
	return url
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}
	return defaultValue
}

func _getEnvDuration(key string, defaultValue time.Duration) time.Duration { //nolint:unused // Reserved for future use
	if value := os.Getenv(key); value != "" {
		if seconds, err := strconv.Atoi(value); err == nil {
			return time.Duration(seconds) * time.Second
		}
	}
	return defaultValue
}

func getEnvInt64(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		if i, err := strconv.ParseInt(value, 10, 64); err == nil {
			return i
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		lower := strings.ToLower(value)
		if lower == "true" || lower == "1" || lower == "yes" {
			return true
		}
		if lower == "false" || lower == "0" || lower == "no" {
			return false
		}
	}
	return defaultValue
}

func getToolScenarios() []string {
	if value := os.Getenv("TOOL_SCENARIOS"); value != "" {
		var scenarios []string
		for _, s := range strings.Split(value, ",") {
			if trimmed := strings.TrimSpace(s); trimmed != "" {
				scenarios = append(scenarios, trimmed)
			}
		}
		if len(scenarios) > 0 {
			return scenarios
		}
	}
	// Default to agent-manager and scenario-to-cloud
	return []string{"agent-manager", "scenario-to-cloud"}
}
