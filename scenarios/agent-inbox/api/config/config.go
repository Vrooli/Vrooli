// Package config provides centralized configuration for the Agent Inbox scenario.
// This is the control surface for tunable levers - parameters that operators and
// agents can adjust to steer behavior without touching implementation code.
//
// Design principles:
//   - Sane defaults that work for common usage
//   - Clear, intention-revealing names describing tradeoffs
//   - Bounded values with validation
//   - Monotonic behavior (e.g., "higher = more thorough but slower")
package config

import "time"

// Default returns the default configuration with all sane defaults.
// This configuration works well for local development and typical deployments.
func Default() *Config {
	return &Config{
		Server: ServerConfig{
			Port:            getEnvOrDefault("API_PORT", "8080"),
			ReadTimeout:     30 * time.Second,
			WriteTimeout:    30 * time.Second,
			IdleTimeout:     120 * time.Second,
			ShutdownTimeout: 10 * time.Second,
		},
		AI: AIConfig{
			DefaultModel:             getEnvOrDefault("DEFAULT_AI_MODEL", "anthropic/claude-3.5-sonnet"),
			CompletionTimeout:        120 * time.Second,
			StreamBufferSize:         4096,
			CompletionReservePercent: 25,
			DefaultContextLength:     8192,
		},
		Integration: IntegrationConfig{
			OllamaBaseURL:       getOllamaBaseURL(),
			OllamaTimeout:       30 * time.Second,
			AgentManagerTimeout: 60 * time.Second,
			OpenRouterTimeout:   120 * time.Second,
			ModelCacheTTL:       5 * time.Minute,
			Naming: NamingConfig{
				Model:               getEnvOrDefault("OLLAMA_NAMING_MODEL", "llama3.1:8b"),
				Temperature:         0.3,
				MaxTokens:           20,
				SummaryMessageLimit: 10,
				SummaryContentLimit: 200,
				Timeout:             30 * time.Second,
				FallbackName:        "New Conversation",
			},
			ToolDiscovery: ToolDiscoveryConfig{
				AutoDiscovery:    getEnvBool("TOOL_AUTO_DISCOVERY", true),
				Scenarios:        getToolScenarios(),
				DiscoveryTimeout: 10 * time.Second,
				CacheTTL:         60 * time.Second,
				RefreshOnStartup: true,
			},
		},
		Resilience: ResilienceConfig{
			RetryAttempts:           3,
			RetryBaseDelay:          1 * time.Second,
			RetryMaxDelay:           10 * time.Second,
			CircuitBreakerThreshold: 5,
			CircuitBreakerCooldown:  30 * time.Second,
			EnableNamingFallback:    true,
		},
		Storage: StorageConfig{
			BasePath:             getEnvOrDefault("UPLOAD_BASE_PATH", "./uploads"),
			BaseURL:              getEnvOrDefault("UPLOAD_BASE_URL", "/api/v1/uploads"),
			MaxFileSize:          getEnvInt64("UPLOAD_MAX_SIZE_MB", 20) * 1024 * 1024,
			AllowedImageTypes:    []string{"image/jpeg", "image/png", "image/gif", "image/webp"},
			AllowedDocumentTypes: []string{"application/pdf"},
		},
		Templates: TemplatesConfig{
			BasePath:    getEnvOrDefault("TEMPLATES_BASE_PATH", "../templates"),
			DefaultsDir: "defaults",
			UserDir:     "user",
		},
		Skills: SkillsConfig{
			BasePath:    getEnvOrDefault("SKILLS_BASE_PATH", "../skills"),
			DefaultsDir: "defaults",
			UserDir:     "user",
		},
		PromptSync: PromptSyncConfig{
			Enabled:             getEnvBool("PROMPT_SYNC_ENABLED", true),
			PromptManagerURL:    getPromptManagerURL(),
			SyncIntervalSeconds: getEnvInt("PROMPT_SYNC_INTERVAL", 60),
			SyncTimeout:         30 * time.Second,
			SkillOverridesPath:  getEnvOrDefault("SKILL_OVERRIDES_PATH", "../config/skills.json"),
		},
		SkillSuggest: SkillSuggestConfig{
			Enabled:         getEnvBool("SKILL_SUGGEST_ENABLED", true),
			Model:           getEnvOrDefault("SKILL_SUGGEST_MODEL", "llama3.1:8b"),
			MaxMessages:     getEnvInt("SKILL_SUGGEST_MAX_MESSAGES", 10),
			MaxContentLen:   getEnvInt("SKILL_SUGGEST_MAX_CONTENT_LEN", 300),
			CacheTTLSeconds: getEnvInt("SKILL_SUGGEST_CACHE_TTL", 60),
			MaxSuggestions:  getEnvInt("SKILL_SUGGEST_MAX_SUGGESTIONS", 5),
			QueryCount:      getEnvInt("SKILL_SUGGEST_QUERY_COUNT", 3),
			Timeout:         15 * time.Second,
		},
	}
}

// GetStorageConfig returns the storage configuration with defaults.
func GetStorageConfig() *StorageConfig {
	return &StorageConfig{
		BasePath:             getEnvOrDefault("UPLOAD_BASE_PATH", "./uploads"),
		BaseURL:              getEnvOrDefault("UPLOAD_BASE_URL", "/api/v1/uploads"),
		MaxFileSize:          getEnvInt64("UPLOAD_MAX_SIZE_MB", 20) * 1024 * 1024,
		AllowedImageTypes:    []string{"image/jpeg", "image/png", "image/gif", "image/webp"},
		AllowedDocumentTypes: []string{"application/pdf"},
	}
}
