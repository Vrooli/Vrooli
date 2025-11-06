package main

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"app-issue-tracker-api/internal/logging"
	serverpkg "app-issue-tracker-api/internal/server"
)

func loadConfig() *serverpkg.Config {
	vrooliRoot := getVrooliRoot()
	scenarioRoot := filepath.Join(vrooliRoot, "scenarios/app-issue-tracker")

	defaultIssuesDir := filepath.Join(scenarioRoot, "data/issues")
	if _, err := os.Stat("./data/issues"); err == nil {
		defaultIssuesDir = "./data/issues"
	}

	qdrantURL := strings.TrimSpace(os.Getenv("QDRANT_URL"))
	if qdrantURL == "" {
		logging.LogInfo("Qdrant URL not provided; semantic search disabled")
	}

	return &serverpkg.Config{
		Port:                    getEnv("API_PORT", getEnv("PORT", "")),
		QdrantURL:               qdrantURL,
		IssuesDir:               getEnv("ISSUES_DIR", defaultIssuesDir),
		ScenarioRoot:            scenarioRoot,
		VrooliRoot:              vrooliRoot,
		WebsocketAllowedOrigins: parseAllowedOrigins(os.Getenv("API_WS_ALLOWED_ORIGINS")),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func parseAllowedOrigins(raw string) []string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil
	}
	parts := strings.Split(trimmed, ",")
	origins := make([]string, 0, len(parts))
	for _, part := range parts {
		candidate := strings.TrimSpace(part)
		if candidate == "" {
			continue
		}
		origins = append(origins, candidate)
	}
	if len(origins) == 0 {
		return nil
	}
	return origins
}

func getVrooliRoot() string {
	if root, ok := os.LookupEnv("APP_ISSUE_TRACKER_ROOT"); ok {
		trimmed := strings.TrimSpace(root)
		if trimmed == "" {
			logging.LogError("APP_ISSUE_TRACKER_ROOT environment variable is set but empty")
			os.Exit(1)
		}
		return trimmed
	}

	if root, ok := os.LookupEnv("VROOLI_ROOT"); ok {
		trimmed := strings.TrimSpace(root)
		if trimmed == "" {
			logging.LogError("VROOLI_ROOT environment variable is set but empty")
			os.Exit(1)
		}
		return trimmed
	}

	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	if out, err := cmd.Output(); err == nil {
		return strings.TrimSpace(string(out))
	}

	ex, _ := os.Executable()
	return filepath.Dir(filepath.Dir(ex))
}

func main() {
	lifecycleManaged, ok := os.LookupEnv("VROOLI_LIFECYCLE_MANAGED")
	if !ok || lifecycleManaged != "true" {
		fmt.Fprintf(os.Stderr, `❌ This binary must be run through the Vrooli lifecycle system.

🚀 Instead, use:
   vrooli scenario start app-issue-tracker

💡 The lifecycle system provides environment variables, port allocation,
   and dependency management automatically. Direct execution is not supported.
`)
		os.Exit(1)
	}

	config := loadConfig()
	if strings.TrimSpace(config.Port) == "" {
		logging.LogError("API_PORT (or PORT) must be configured before starting the server")
		os.Exit(1)
	}

	srv, handler, err := serverpkg.NewServer(config)
	if err != nil {
		logging.LogErrorErr("Failed to initialize server", err)
		os.Exit(1)
	}

	srv.Start()
	defer srv.Stop()

	logging.LogInfo("Starting App Issue Tracker API", "port", config.Port)
	logging.LogInfo("API health endpoint configured", "url", fmt.Sprintf("http://localhost:%s/health", config.Port))
	logging.LogInfo("API base URL", "url", fmt.Sprintf("http://localhost:%s/api/v1", config.Port))
	logging.LogInfo("Scenario configuration", "issues_dir", config.IssuesDir, "scenario_root", config.ScenarioRoot)
	logging.LogInfo("Processor loop state", "active", false)

	if err := http.ListenAndServe(":"+config.Port, handler); err != nil {
		logging.LogErrorErr("Server failed to start", err, "port", config.Port)
		os.Exit(1)
	}
}
