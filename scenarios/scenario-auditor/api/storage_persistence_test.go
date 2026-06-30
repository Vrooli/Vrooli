package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRuleStateStorePersistsToConfigClass(t *testing.T) {
	root := t.TempDir()
	t.Setenv("VROOLI_CONFIG_ROOT", filepath.Join(root, "config-root"))

	store := &RuleStateStore{states: make(map[string]bool)}
	store.enablePersistence()

	if store.filePath == "" {
		t.Fatal("expected filePath to be set")
	}
	if filepath.Base(store.filePath) != "rule-preferences.json" {
		t.Fatalf("unexpected file path %q", store.filePath)
	}
	if err := store.SetState("health_check", false); err != nil {
		t.Fatalf("SetState: %v", err)
	}
	if _, err := os.Stat(store.filePath); err != nil {
		t.Fatalf("expected persisted file: %v", err)
	}
}

func TestProtectedScenariosStorePersistsToConfigClass(t *testing.T) {
	root := t.TempDir()
	t.Setenv("VROOLI_CONFIG_ROOT", filepath.Join(root, "config-root"))

	store := &ProtectedScenariosStore{protectedSet: make(map[string]bool)}
	store.enablePersistence()

	if err := store.AddProtectedScenario("ecosystem-manager"); err != nil {
		t.Fatalf("AddProtectedScenario: %v", err)
	}
	if filepath.Base(store.filePath) != "protected-scenarios.json" {
		t.Fatalf("unexpected file path %q", store.filePath)
	}
	if _, err := os.Stat(store.filePath); err != nil {
		t.Fatalf("expected persisted file: %v", err)
	}
}

func TestStandardsAndVulnerabilitiesPersistToCacheClass(t *testing.T) {
	root := t.TempDir()
	cacheRoot := filepath.Join(root, "cache-root")
	t.Setenv("VROOLI_CACHE_ROOT", cacheRoot)

	stdStore := &StandardsStore{
		violations: make(map[string][]StandardsViolation),
		lastCheck:  make(map[string]time.Time),
	}
	stdStore.enablePersistence()
	stdStore.StoreViolations("demo", []StandardsViolation{{ID: "S-1", ScenarioName: "demo", Severity: "high"}})

	vStore := &VulnerabilityStore{
		vulnerabilities: make(map[string][]StoredVulnerability),
		lastScan:        make(map[string]time.Time),
	}
	vStore.enablePersistence()
	vStore.vulnerabilities["demo"] = []StoredVulnerability{{ID: "V-1", ScenarioName: "demo", Severity: "critical"}}
	vStore.lastScan["demo"] = time.Now()
	if err := vStore.saveToFile(); err != nil {
		t.Fatalf("saveToFile: %v", err)
	}

	for _, path := range []string{stdStore.filePath, vStore.filePath} {
		if !filepath.HasPrefix(path, filepath.Join(cacheRoot, "vrooli", scenarioAuditorScenarioID)) {
			t.Fatalf("expected cache path, got %q", path)
		}
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected persisted file %q: %v", path, err)
		}
	}
}

func TestAutomatedFixStoreSplitsConfigAndHistory(t *testing.T) {
	root := t.TempDir()
	configRoot := filepath.Join(root, "config-root")
	dataRoot := filepath.Join(root, "data-root")
	t.Setenv("VROOLI_CONFIG_ROOT", configRoot)
	t.Setenv("VROOLI_DATA_ROOT", dataRoot)

	store := &AutomatedFixStore{
		config: AutomatedFixConfig{
			ViolationTypes: []string{"security"},
			Severities:     []string{"critical"},
			Strategy:       defaultAutomatedFixStrategy,
			LoopDelay:      defaultLoopDelaySeconds,
			TimeoutSeconds: defaultTimeoutSeconds,
			Model:          openRouterModel(),
		},
		maxHistory: 10,
	}
	store.enablePersistence()
	store.Enable(AutomatedFixConfigInput{})
	store.Append(AutomatedFixRecord{ID: "fix-1", ScenarioName: "demo", Status: "applied"})

	if filepath.Base(store.configPath) != "automated-fix-config.json" {
		t.Fatalf("unexpected config path %q", store.configPath)
	}
	if filepath.Base(store.historyPath) != "automated-fix-history.json" {
		t.Fatalf("unexpected history path %q", store.historyPath)
	}
	if _, err := os.Stat(store.configPath); err != nil {
		t.Fatalf("expected config file: %v", err)
	}
	if _, err := os.Stat(store.historyPath); err != nil {
		t.Fatalf("expected history file: %v", err)
	}

	configBytes, err := os.ReadFile(store.configPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	historyBytes, err := os.ReadFile(store.historyPath)
	if err != nil {
		t.Fatalf("read history: %v", err)
	}

	var cfgPayload automatedFixStoreData
	if err := json.Unmarshal(configBytes, &cfgPayload); err != nil {
		t.Fatalf("unmarshal config: %v", err)
	}
	var historyPayload automatedFixHistoryData
	if err := json.Unmarshal(historyBytes, &historyPayload); err != nil {
		t.Fatalf("unmarshal history: %v", err)
	}
	if len(historyPayload.History) != 1 {
		t.Fatalf("expected one history record, got %d", len(historyPayload.History))
	}
	if cfgPayload.Config.Model == "" {
		t.Fatal("expected config payload to contain model")
	}
}
