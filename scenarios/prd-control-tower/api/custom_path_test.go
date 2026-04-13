package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// ──────────────────────────────────────────────────────────────────────────────
// resolveEntityBaseDir
// ──────────────────────────────────────────────────────────────────────────────

func TestResolveEntityBaseDir(t *testing.T) {
	tests := []struct {
		name       string
		entityType string
		entityName string
		customPath string
		vrooliRoot string
		want       string
		wantErr    bool
	}{
		{
			name:       "custom path takes precedence",
			entityType: "scenario",
			entityName: "my-scenario",
			customPath: "/custom/path/to/prd",
			vrooliRoot: "/home/user/Vrooli",
			want:       "/custom/path/to/prd",
		},
		{
			name:       "custom path with whitespace is detected as non-empty",
			entityType: "scenario",
			entityName: "my-scenario",
			customPath: "  /custom/path  ",
			vrooliRoot: "/home/user/Vrooli",
			want:       "  /custom/path  ", // TrimSpace only used for emptiness check
		},
		{
			name:       "empty custom path falls back to scenario path",
			entityType: "scenario",
			entityName: "my-scenario",
			customPath: "",
			vrooliRoot: repoRootForContractFixture(t),
			want:       filepath.Join(repoRootForContractFixture(t), "scenarios", "my-scenario"),
		},
		{
			name:       "whitespace-only custom path falls back to scenario path",
			entityType: "scenario",
			entityName: "my-scenario",
			customPath: "   ",
			vrooliRoot: repoRootForContractFixture(t),
			want:       filepath.Join(repoRootForContractFixture(t), "scenarios", "my-scenario"),
		},
		{
			name:       "resource entity type uses resources directory",
			entityType: "resource",
			entityName: "my-resource",
			customPath: "",
			vrooliRoot: repoRootForContractFixture(t),
			want:       filepath.Join(repoRootForContractFixture(t), "resources", "my-resource"),
		},
		{
			name:       "custom path ignores entity type entirely",
			entityType: "resource",
			entityName: "my-resource",
			customPath: "/some/arbitrary/dir",
			vrooliRoot: "/home/user/Vrooli",
			want:       "/some/arbitrary/dir",
		},
		{
			name:       "no repo context and no custom path errors",
			entityType: "scenario",
			entityName: "my-scenario",
			customPath: "",
			vrooliRoot: "",
			wantErr:    true,
		},
		{
			name:       "custom path still works without VROOLI_ROOT",
			entityType: "scenario",
			entityName: "my-scenario",
			customPath: "/tmp/my-prd",
			vrooliRoot: "", // will also clear HOME
			want:       "/tmp/my-prd",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("VROOLI_ROOT", tt.vrooliRoot)
			if tt.vrooliRoot == "" && tt.customPath == "" {
				origWD, err := os.Getwd()
				if err != nil {
					t.Fatalf("getwd: %v", err)
				}
				temp := t.TempDir()
				if err := os.Chdir(temp); err != nil {
					t.Fatalf("chdir: %v", err)
				}
				t.Cleanup(func() { _ = os.Chdir(origWD) })
			}

			got, err := resolveEntityBaseDir(tt.entityType, tt.entityName, tt.customPath)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("resolveEntityBaseDir() = %q, want %q", got, tt.want)
			}
		})
	}
}

func repoRootForContractFixture(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(filename), "..", "..", ".."))
}

// ──────────────────────────────────────────────────────────────────────────────
// extractOperationalTargets with customPath
// ──────────────────────────────────────────────────────────────────────────────

func TestExtractOperationalTargets_CustomPath(t *testing.T) {
	customDir := t.TempDir()

	prdContent := `# PRD

## 🎯 Operational Targets
### 🔴 P0 – Must ship for viability
- [ ] OT-P0-001 | Custom path target | Uses custom directory
### 🟠 P1 – Should have post-launch
- [ ] OT-P1-001 | Another target | Secondary
`
	if err := os.WriteFile(filepath.Join(customDir, "PRD.md"), []byte(prdContent), 0o644); err != nil {
		t.Fatalf("write PRD: %v", err)
	}

	// Don't set VROOLI_ROOT — custom path should bypass it entirely
	t.Setenv("VROOLI_ROOT", "")
	t.Setenv("HOME", "")

	targets, err := extractOperationalTargets("scenario", "irrelevant-name", customDir)
	if err != nil {
		t.Fatalf("extractOperationalTargets() error = %v", err)
	}

	if len(targets) < 2 {
		t.Fatalf("expected at least 2 targets, got %d", len(targets))
	}

	hasP0 := false
	hasP1 := false
	for _, target := range targets {
		if target.Criticality == "P0" {
			hasP0 = true
		}
		if target.Criticality == "P1" {
			hasP1 = true
		}
	}
	if !hasP0 {
		t.Error("expected P0 target")
	}
	if !hasP1 {
		t.Error("expected P1 target")
	}
}

func TestExtractOperationalTargets_CustomPathNoPRD(t *testing.T) {
	// Custom path exists but has no PRD.md — returns error (PRD not found)
	customDir := t.TempDir()

	targets, err := extractOperationalTargets("scenario", "irrelevant", customDir)
	if err == nil {
		t.Fatal("expected error for missing PRD.md")
	}
	if len(targets) != 0 {
		t.Errorf("expected 0 targets for missing PRD, got %d", len(targets))
	}
}

func TestExtractOperationalTargets_WithoutCustomPathUsesStandardPath(t *testing.T) {
	vrooliRoot := t.TempDir()
	t.Setenv("VROOLI_ROOT", vrooliRoot)

	scenarioDir := filepath.Join(vrooliRoot, "scenarios", "std-scenario")
	if err := os.MkdirAll(scenarioDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	prdContent := `## 🎯 Operational Targets
### 🔴 P0 – Must ship for viability
- [ ] OT-P0-001 | Standard path target | Verifies default
`
	if err := os.WriteFile(filepath.Join(scenarioDir, "PRD.md"), []byte(prdContent), 0o644); err != nil {
		t.Fatalf("write PRD: %v", err)
	}

	targets, err := extractOperationalTargets("scenario", "std-scenario")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(targets) == 0 {
		t.Error("expected targets from standard path")
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// loadRequirementsForEntity with customPath
// ──────────────────────────────────────────────────────────────────────────────

func TestLoadRequirementsForEntity_CustomPath(t *testing.T) {
	customDir := t.TempDir()
	reqsDir := filepath.Join(customDir, "requirements")
	if err := os.MkdirAll(reqsDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	indexData := requirementsFile{
		Metadata: map[string]any{"description": "Custom path requirements"},
		Requirements: []RequirementRecordInput{
			{
				ID:          "CUSTOM-001",
				Category:    "custom",
				PRDRef:      "OT-P0-001",
				Title:       "Custom requirement",
				Description: "Loaded via custom path",
				Status:      "pending",
			},
		},
	}
	indexBytes, _ := json.MarshalIndent(indexData, "", "  ")
	if err := os.WriteFile(filepath.Join(reqsDir, "index.json"), indexBytes, 0o644); err != nil {
		t.Fatalf("write index.json: %v", err)
	}

	// No VROOLI_ROOT needed
	t.Setenv("VROOLI_ROOT", "")
	t.Setenv("HOME", "")

	// Clear requirements cache to avoid stale data from other tests
	requirementsCache.Range(func(key, _ any) bool {
		requirementsCache.Delete(key)
		return true
	})

	groups, err := loadRequirementsForEntity("scenario", "any-name", customDir)
	if err != nil {
		t.Fatalf("loadRequirementsForEntity() error = %v", err)
	}

	if len(groups) == 0 {
		t.Fatal("expected at least one requirement group")
	}

	found := false
	for _, g := range groups {
		for _, r := range g.Requirements {
			if r.ID == "CUSTOM-001" {
				found = true
			}
		}
	}
	if !found {
		t.Error("expected CUSTOM-001 in loaded requirements")
	}
}

func TestLoadRequirementsForEntity_CustomPathCacheKeyIsolation(t *testing.T) {
	// Two different custom paths should produce different cache keys
	dir1 := t.TempDir()
	dir2 := t.TempDir()

	// Create requirements only in dir1
	reqsDir1 := filepath.Join(dir1, "requirements")
	if err := os.MkdirAll(reqsDir1, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	indexData := requirementsFile{
		Requirements: []RequirementRecordInput{
			{ID: "DIR1-001", Title: "From dir1", Status: "pending"},
		},
	}
	indexBytes, _ := json.MarshalIndent(indexData, "", "  ")
	if err := os.WriteFile(filepath.Join(reqsDir1, "index.json"), indexBytes, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	// dir2 has no requirements dir at all

	requirementsCache.Range(func(key, _ any) bool {
		requirementsCache.Delete(key)
		return true
	})

	groups1, err := loadRequirementsForEntity("scenario", "same-name", dir1)
	if err != nil {
		t.Fatalf("load dir1: %v", err)
	}
	groups2, err := loadRequirementsForEntity("scenario", "same-name", dir2)
	if err != nil {
		t.Fatalf("load dir2: %v", err)
	}

	if len(groups1) == 0 {
		t.Error("dir1 should have requirements")
	}
	if len(groups2) != 0 {
		t.Error("dir2 should have no requirements")
	}
}

func TestLoadRequirementsForEntity_MissingRequirementsDir(t *testing.T) {
	customDir := t.TempDir() // exists but has no requirements/ subdir

	requirementsCache.Range(func(key, _ any) bool {
		requirementsCache.Delete(key)
		return true
	})

	groups, err := loadRequirementsForEntity("scenario", "missing-reqs", customDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(groups) != 0 {
		t.Errorf("expected empty groups for missing requirements dir, got %d", len(groups))
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// validateOperationalTargetsLinkage with customPath
// ──────────────────────────────────────────────────────────────────────────────

func TestValidateOperationalTargetsLinkage_CustomPath(t *testing.T) {
	customDir := t.TempDir()
	reqsDir := filepath.Join(customDir, "requirements")
	if err := os.MkdirAll(reqsDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	// Non-empty imports trigger enforcement
	if err := os.WriteFile(filepath.Join(reqsDir, "index.json"), []byte(`{"imports":["core.json"]}`), 0o644); err != nil {
		t.Fatalf("write index: %v", err)
	}

	// Create the imported file with no requirements that link to OT-1
	coreData := requirementsFile{
		Requirements: []RequirementRecordInput{
			{ID: "REQ-UNRELATED", Title: "Unrelated", PRDRef: "Other > Something", Status: "pending"},
		},
	}
	coreBytes, _ := json.MarshalIndent(coreData, "", "  ")
	if err := os.WriteFile(filepath.Join(reqsDir, "core.json"), coreBytes, 0o644); err != nil {
		t.Fatalf("write core.json: %v", err)
	}

	content := `# PRD

## 🎯 Operational Targets
### 🔴 P0 – Must ship for viability
- [ ] OT-1 | Critical feature | Must have
`

	// Clear cache
	requirementsCache.Range(func(key, _ any) bool {
		requirementsCache.Delete(key)
		return true
	})

	// Should fail because P0 target OT-1 has no linked requirements
	err := validateOperationalTargetsLinkage("scenario", "irrelevant", content, customDir)
	if err == nil {
		t.Fatal("expected error for orphaned P0 target with custom path")
	}
	if !strings.Contains(err.Error(), "Critical feature") {
		t.Errorf("error should mention orphaned target title, got: %v", err)
	}
}

func TestValidateOperationalTargetsLinkage_CustomPathEmptyIndex(t *testing.T) {
	customDir := t.TempDir()
	reqsDir := filepath.Join(customDir, "requirements")
	if err := os.MkdirAll(reqsDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	// Empty imports should skip validation
	if err := os.WriteFile(filepath.Join(reqsDir, "index.json"), []byte(`{"imports":[]}`), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	content := `## 🎯 Operational Targets
### 🔴 P0 – Must ship for viability
- [ ] OT-1 | Critical feature | Should pass
`

	err := validateOperationalTargetsLinkage("scenario", "irrelevant", content, customDir)
	if err != nil {
		t.Fatalf("expected nil error for empty imports with custom path, got: %v", err)
	}
}

func TestValidateOperationalTargetsLinkage_CustomPathNoRequirementsDir(t *testing.T) {
	customDir := t.TempDir() // no requirements/ subdir

	content := `## 🎯 Operational Targets
### 🔴 P0 – Must ship for viability
- [ ] OT-1 | Critical feature | Should pass because no reqs dir
`

	err := validateOperationalTargetsLinkage("scenario", "irrelevant", content, customDir)
	if err != nil {
		t.Fatalf("expected nil error when no requirements dir exists, got: %v", err)
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// buildPrompt with customPath
// ──────────────────────────────────────────────────────────────────────────────

func TestBuildPrompt_CustomPath(t *testing.T) {
	customDir := t.TempDir()
	prdContent := "# Existing PRD\n\nThis is the published content at a custom path."
	if err := os.WriteFile(filepath.Join(customDir, "PRD.md"), []byte(prdContent), 0o644); err != nil {
		t.Fatalf("write PRD: %v", err)
	}

	draft := Draft{
		ID:         "d1",
		EntityType: "scenario",
		EntityName: "test-scenario",
		Content:    "# Draft Content",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		Status:     "draft",
	}

	// With custom path and includeExisting=true, should read from custom dir
	prompt := buildPrompt(draft, "🎯 Full PRD", "some context", "", true, nil, customDir)

	if !strings.Contains(prompt, "published content at a custom path") {
		t.Error("expected prompt to include published PRD from custom path")
	}
}

func TestBuildPrompt_CustomPathMissingPRD(t *testing.T) {
	customDir := t.TempDir() // no PRD.md

	draft := Draft{
		ID:         "d1",
		EntityType: "scenario",
		EntityName: "test-scenario",
		Content:    "# Draft Content",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		Status:     "draft",
	}

	// Should fall back to draft content when no published PRD exists
	prompt := buildPrompt(draft, "🎯 Full PRD", "", "", true, nil, customDir)

	if !strings.Contains(prompt, "Draft Content") {
		t.Error("expected prompt to fall back to draft content when custom path has no PRD")
	}
}

func TestBuildPrompt_WithoutCustomPathUsesStandardPath(t *testing.T) {
	vrooliRoot := t.TempDir()
	t.Setenv("VROOLI_ROOT", vrooliRoot)

	scenarioDir := filepath.Join(vrooliRoot, "scenarios", "my-scenario")
	if err := os.MkdirAll(scenarioDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(scenarioDir, "PRD.md"), []byte("# Standard PRD content"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	draft := Draft{
		ID:         "d1",
		EntityType: "scenario",
		EntityName: "my-scenario",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		Status:     "draft",
	}

	// No custom path — should read from standard location
	prompt := buildPrompt(draft, "🎯 Full PRD", "", "", true, nil)

	if !strings.Contains(prompt, "Standard PRD content") {
		t.Error("expected prompt to include PRD from standard path")
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// computeQualityReport with customPath
// ──────────────────────────────────────────────────────────────────────────────

func TestComputeQualityReport_CustomPath(t *testing.T) {
	customDir := t.TempDir()

	// Create a valid PRD
	prdContent := `# PRD

## 🎯 Operational Targets
### 🔴 P0 – Must ship for viability
- [ ] OT-P0-001 | Core feature | Required
`
	if err := os.WriteFile(filepath.Join(customDir, "PRD.md"), []byte(prdContent), 0o644); err != nil {
		t.Fatalf("write PRD: %v", err)
	}

	// Don't need VROOLI_ROOT
	t.Setenv("VROOLI_ROOT", "")
	t.Setenv("HOME", "")

	report, err := computeQualityReport("scenario", "custom-entity", customDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !report.HasPRD {
		t.Error("expected HasPRD=true for existing PRD at custom path")
	}
	if report.PRDPath != filepath.Join(customDir, "PRD.md") {
		t.Errorf("PRDPath = %q, want %q", report.PRDPath, filepath.Join(customDir, "PRD.md"))
	}
}

func TestComputeQualityReport_CustomPathNoPRD(t *testing.T) {
	customDir := t.TempDir() // empty

	report, err := computeQualityReport("scenario", "no-prd-entity", customDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if report.HasPRD {
		t.Error("expected HasPRD=false when no PRD.md at custom path")
	}
	if report.Status != "missing_prd" {
		t.Errorf("expected status missing_prd, got %q", report.Status)
	}
}

func TestBuildQualityReport_CustomPathCacheKey(t *testing.T) {
	customDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(customDir, "PRD.md"), []byte("# Test PRD"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	// Clear quality cache
	qualityCache.Range(func(key, _ any) bool {
		qualityCache.Delete(key)
		return true
	})

	// First call caches
	report1, err := buildQualityReport("scenario", "cache-test", true, customDir)
	if err != nil {
		t.Fatalf("first call: %v", err)
	}

	// Second call should hit cache
	report2, err := buildQualityReport("scenario", "cache-test", true, customDir)
	if err != nil {
		t.Fatalf("second call: %v", err)
	}
	if !report2.CacheUsed {
		t.Error("expected second call to use cache")
	}

	// Different custom path should NOT hit cache
	otherDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(otherDir, "PRD.md"), []byte("# Other PRD"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	report3, err := buildQualityReport("scenario", "cache-test", true, otherDir)
	if err != nil {
		t.Fatalf("third call: %v", err)
	}
	if report3.CacheUsed {
		t.Error("different custom path should not hit cache")
	}

	// No custom path also should NOT hit cache for the same entity name
	vrooliRoot := t.TempDir()
	t.Setenv("VROOLI_ROOT", vrooliRoot)
	scenarioDir := filepath.Join(vrooliRoot, "scenarios", "cache-test")
	if err := os.MkdirAll(scenarioDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(scenarioDir, "PRD.md"), []byte("# Default PRD"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	report4, err := buildQualityReport("scenario", "cache-test", true)
	if err != nil {
		t.Fatalf("fourth call: %v", err)
	}
	if report4.CacheUsed {
		t.Error("default path should not hit cache for custom-path entry")
	}

	_ = report1 // use all reports
}

// ──────────────────────────────────────────────────────────────────────────────
// requirementsIndexHasImports with custom path context
// ──────────────────────────────────────────────────────────────────────────────

func TestRequirementsIndexHasImports(t *testing.T) {
	tests := []struct {
		name       string
		content    string
		wantResult bool
		wantErr    bool
	}{
		{
			name:       "empty imports array",
			content:    `{"imports":[]}`,
			wantResult: false,
		},
		{
			name:       "non-empty imports array",
			content:    `{"imports":["core.json"]}`,
			wantResult: true,
		},
		{
			name:       "no imports key",
			content:    `{}`,
			wantResult: false,
		},
		{
			name:    "invalid JSON",
			content: `not-json`,
			wantErr: true,
		},
		{
			name:       "multiple imports",
			content:    `{"imports":["core.json","advanced.json","extras.json"]}`,
			wantResult: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpFile := filepath.Join(t.TempDir(), "index.json")
			if err := os.WriteFile(tmpFile, []byte(tt.content), 0o644); err != nil {
				t.Fatalf("write: %v", err)
			}

			got, err := requirementsIndexHasImports(tmpFile)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.wantResult {
				t.Errorf("requirementsIndexHasImports() = %v, want %v", got, tt.wantResult)
			}
		})
	}
}
