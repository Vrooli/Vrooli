package config

import (
	"testing"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/testutil/mocks"
)

type mapEnvironment map[string]string

func (e mapEnvironment) LookupEnv(key string) (string, bool) {
	value, ok := e[key]
	return value, ok
}

var _ Environment = mapEnvironment{}

func TestLoadWithEnvironmentCapturesRuntimeOverrides(t *testing.T) {
	environment := mapEnvironment{
		"API_PORT":                                    "8080",
		"VROOLI_BRIDGE_API_TOKEN":                     "bridge-token",
		"VROOLI_LIFECYCLE_MANAGED":                    "true",
		"VROOLI_BUILD_IDENTITY":                       "build-identity",
		"OLLAMA_BASE_URL":                             "http://ollama.local:11434",
		"SYSTEM_MONITOR_STORAGE_MODE":                 "memory",
		"SYSTEM_MONITOR_COMPACTION_WINDOW":            "02:00-04:00",
		"SYSTEM_MONITOR_INVESTIGATION_ROOT":           "/tmp/investigations",
		"SYSTEM_MONITOR_SOCKET_ATTRIBUTION_THRESHOLD": "900",
		"VROOLI_CLEANUP_MANAGER_API_URL":              "http://storage.local",
		"VROOLI_AUTOHEAL_API_URL":                     "http://autoheal.local",
		"GOCACHE":                                     "/tmp/go-cache",
		"PLAYWRIGHT_BROWSERS_PATH":                    "/tmp/playwright",
		"VROOLI_HOME":                                 "/tmp/vrooli",
	}
	cfg, err := LoadWithEnvironment(environment)
	if err != nil {
		t.Fatalf("LoadWithEnvironment: %v", err)
	}
	if cfg.Getenv("VROOLI_BRIDGE_API_TOKEN") != "bridge-token" {
		t.Fatal("runtime environment lookup did not use injected reader")
	}
	if !cfg.Server.ManagedLifecycle || cfg.Server.BridgeToken != "bridge-token" || cfg.Health.BuildIdentity != "build-identity" || cfg.Health.OllamaBaseURL != "http://ollama.local:11434" {
		t.Fatalf("server/health runtime config mismatch: %+v %+v", cfg.Server, cfg.Health)
	}
	if cfg.Monitoring.StorageMode != "memory" || cfg.Monitoring.CompactionWindow != "02:00-04:00" || cfg.Monitoring.InvestigationRoot != "/tmp/investigations" || cfg.Monitoring.SocketAttributionThreshold != 900 {
		t.Fatalf("monitoring runtime config mismatch: %+v", cfg.Monitoring)
	}
	if cfg.Resources.StorageManagerURL != "http://storage.local" || cfg.Resources.AutohealURL != "http://autoheal.local" {
		t.Fatalf("integration URLs mismatch: %+v", cfg.Resources)
	}
	if cfg.Monitoring.GoCachePath != "/tmp/go-cache" || cfg.Monitoring.PlaywrightCachePath != "/tmp/playwright" || cfg.Monitoring.RuntimeHome != "/tmp/vrooli" {
		t.Fatalf("host path config mismatch: %+v", cfg.Monitoring)
	}
}

func TestLoadWithEnvironmentNormalizesInvalidSocketThreshold(t *testing.T) {
	cfg, err := LoadWithEnvironment(mapEnvironment{"API_PORT": "8080", "SYSTEM_MONITOR_SOCKET_ATTRIBUTION_THRESHOLD": "-1"})
	if err != nil {
		t.Fatalf("LoadWithEnvironment: %v", err)
	}
	if cfg.Monitoring.SocketAttributionThreshold != 5000 {
		t.Fatalf("socket attribution threshold = %d, want default 5000", cfg.Monitoring.SocketAttributionThreshold)
	}
}

func TestLoadWithEnvironmentUsesInjectedLoggerForDiagnostics(t *testing.T) {
	logger := &mocks.Logger{}
	_, err := LoadWithEnvironmentAndLogger(mapEnvironment{
		"API_PORT":                 "8080",
		"METRICS_INTERVAL_SECONDS": "invalid",
	}, logger)
	if err != nil {
		t.Fatalf("LoadWithEnvironmentAndLogger: %v", err)
	}
	if !logger.Contains("warn", "invalid integer configuration; using default") {
		t.Fatalf("configuration warning did not use injected logger: %+v", logger.Entries)
	}
}
