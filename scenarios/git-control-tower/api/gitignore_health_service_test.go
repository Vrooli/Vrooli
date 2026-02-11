package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func rulesJSON(t *testing.T, rules ...GroupingRule) string {
	t.Helper()
	cfg := GroupingRulesConfig{Enabled: true, Rules: rules}
	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestAnalyzeHealth_PrefixMode_SingleGroup(t *testing.T) {
	configPath := "/config/grouping-rules.json"
	fs := NewFakeFileIO().
		WithFile(configPath, rulesJSON(t, GroupingRule{
			ID: "r1", Label: "Resources", Prefixes: []string{"resources/"}, Mode: "prefix",
		})).
		WithFile("/repo/.gitignore", "resources/postgres/data\n")

	deps := HealthDeps{
		FS:      fs,
		RepoDir: "/repo",
		GroupingDeps: GroupingDeps{
			FS:         fs,
			ConfigPath: configPath,
		},
	}

	result, err := AnalyzeGitignoreHealth(deps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.RootEntryCount != 1 {
		t.Fatalf("expected 1 root entry, got %d", result.RootEntryCount)
	}
	if len(result.Suggestions) != 1 {
		t.Fatalf("expected 1 suggestion, got %d", len(result.Suggestions))
	}
	s := result.Suggestions[0]
	if s.Type != "single_group" {
		t.Fatalf("expected single_group, got %s", s.Type)
	}
	if s.GroupDir != "resources/" {
		t.Fatalf("expected groupDir resources/, got %s", s.GroupDir)
	}
	if s.TargetPattern != "postgres/data" {
		t.Fatalf("expected target postgres/data, got %s", s.TargetPattern)
	}
	if s.GroupLabel != "Resources" {
		t.Fatalf("expected label Resources, got %s", s.GroupLabel)
	}
}

func TestAnalyzeHealth_SegmentMode_SingleGroup(t *testing.T) {
	configPath := "/config/grouping-rules.json"
	fs := NewFakeFileIO().
		WithFile(configPath, rulesJSON(t, GroupingRule{
			ID: "s1", Label: "Scenarios", Prefixes: []string{"scenarios/"}, Mode: "segment",
		})).
		WithFile("/repo/.gitignore", "scenarios/foo/bar.txt\n")

	deps := HealthDeps{
		FS:      fs,
		RepoDir: "/repo",
		GroupingDeps: GroupingDeps{
			FS:         fs,
			ConfigPath: configPath,
		},
	}

	result, err := AnalyzeGitignoreHealth(deps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Suggestions) != 1 {
		t.Fatalf("expected 1 suggestion, got %d", len(result.Suggestions))
	}
	s := result.Suggestions[0]
	if s.Type != "single_group" {
		t.Fatalf("expected single_group, got %s", s.Type)
	}
	if s.GroupDir != "scenarios/foo/" {
		t.Fatalf("expected groupDir scenarios/foo/, got %s", s.GroupDir)
	}
	if s.TargetPattern != "bar.txt" {
		t.Fatalf("expected target bar.txt, got %s", s.TargetPattern)
	}
	if s.GroupLabel != "foo" {
		t.Fatalf("expected label foo, got %s", s.GroupLabel)
	}
}

func TestAnalyzeHealth_SegmentMode_CrossGroup(t *testing.T) {
	configPath := "/config/grouping-rules.json"
	fs := NewFakeFileIO().
		WithFile(configPath, rulesJSON(t, GroupingRule{
			ID: "s1", Label: "Scenarios", Prefixes: []string{"scenarios/"}, Mode: "segment",
		})).
		WithFile("/repo/.gitignore", "scenarios/*/build\n")

	deps := HealthDeps{
		FS:      fs,
		RepoDir: "/repo",
		GroupingDeps: GroupingDeps{
			FS:         fs,
			ConfigPath: configPath,
		},
	}

	result, err := AnalyzeGitignoreHealth(deps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Suggestions) != 1 {
		t.Fatalf("expected 1 suggestion, got %d", len(result.Suggestions))
	}
	s := result.Suggestions[0]
	if s.Type != "cross_group" {
		t.Fatalf("expected cross_group, got %s", s.Type)
	}
	if s.GroupLabel != "Scenarios" {
		t.Fatalf("expected label Scenarios, got %s", s.GroupLabel)
	}
}

func TestAnalyzeHealth_NoMatchingGroup(t *testing.T) {
	configPath := "/config/grouping-rules.json"
	fs := NewFakeFileIO().
		WithFile(configPath, rulesJSON(t, GroupingRule{
			ID: "r1", Label: "Resources", Prefixes: []string{"resources/"}, Mode: "prefix",
		})).
		WithFile("/repo/.gitignore", "vendor/something\n")

	deps := HealthDeps{
		FS:      fs,
		RepoDir: "/repo",
		GroupingDeps: GroupingDeps{
			FS:         fs,
			ConfigPath: configPath,
		},
	}

	result, err := AnalyzeGitignoreHealth(deps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Suggestions) != 0 {
		t.Fatalf("expected 0 suggestions, got %d", len(result.Suggestions))
	}
}

func TestAnalyzeHealth_SkipsNegation(t *testing.T) {
	configPath := "/config/grouping-rules.json"
	fs := NewFakeFileIO().
		WithFile(configPath, rulesJSON(t, GroupingRule{
			ID: "r1", Label: "Resources", Prefixes: []string{"resources/"}, Mode: "prefix",
		})).
		WithFile("/repo/.gitignore", "!resources/keep-me\n")

	deps := HealthDeps{
		FS:      fs,
		RepoDir: "/repo",
		GroupingDeps: GroupingDeps{
			FS:         fs,
			ConfigPath: configPath,
		},
	}

	result, err := AnalyzeGitignoreHealth(deps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Suggestions) != 0 {
		t.Fatalf("expected 0 suggestions for negation lines, got %d", len(result.Suggestions))
	}
}

func TestAnalyzeHealth_SkipsComments(t *testing.T) {
	configPath := "/config/grouping-rules.json"
	fs := NewFakeFileIO().
		WithFile(configPath, rulesJSON(t, GroupingRule{
			ID: "r1", Label: "Resources", Prefixes: []string{"resources/"}, Mode: "prefix",
		})).
		WithFile("/repo/.gitignore", "# resources/postgres/data\n")

	deps := HealthDeps{
		FS:      fs,
		RepoDir: "/repo",
		GroupingDeps: GroupingDeps{
			FS:         fs,
			ConfigPath: configPath,
		},
	}

	result, err := AnalyzeGitignoreHealth(deps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Suggestions) != 0 {
		t.Fatalf("expected 0 suggestions for comment lines, got %d", len(result.Suggestions))
	}
	if result.RootEntryCount != 0 {
		t.Fatalf("expected 0 root entries (comments don't count), got %d", result.RootEntryCount)
	}
}

func TestAnalyzeHealth_NoRulesConfigured(t *testing.T) {
	fs := NewFakeFileIO().
		WithFile("/repo/.gitignore", "resources/postgres/data\n")

	deps := HealthDeps{
		FS:      fs,
		RepoDir: "/repo",
		GroupingDeps: GroupingDeps{
			FS:         fs,
			ConfigPath: "/config/nonexistent.json",
		},
	}

	result, err := AnalyzeGitignoreHealth(deps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Suggestions) != 0 {
		t.Fatalf("expected 0 suggestions with no rules, got %d", len(result.Suggestions))
	}
}

func TestAnalyzeHealth_DetectsExistingGitignore(t *testing.T) {
	configPath := "/config/grouping-rules.json"
	fs := NewFakeFileIO().
		WithFile(configPath, rulesJSON(t, GroupingRule{
			ID: "r1", Label: "Resources", Prefixes: []string{"resources/"}, Mode: "prefix",
		})).
		WithFile("/repo/.gitignore", "resources/postgres/data\n").
		WithFile("/repo/resources/.gitignore", "# existing\n")

	deps := HealthDeps{
		FS:      fs,
		RepoDir: "/repo",
		GroupingDeps: GroupingDeps{
			FS:         fs,
			ConfigPath: configPath,
		},
	}

	result, err := AnalyzeGitignoreHealth(deps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Suggestions) != 1 {
		t.Fatalf("expected 1 suggestion, got %d", len(result.Suggestions))
	}
	if !result.Suggestions[0].HasGitignore {
		t.Fatal("expected HasGitignore=true")
	}
}

func TestMoveEntry_Success(t *testing.T) {
	fs := NewFakeFileIO().
		WithFile("/repo/.gitignore", "# header\nresources/postgres/data\nother/stuff\n")

	deps := HealthDeps{
		FS:      fs,
		RepoDir: "/repo",
		GroupingDeps: GroupingDeps{
			FS:         fs,
			ConfigPath: "/config/grouping-rules.json",
		},
	}

	req := GitignoreMoveRequest{
		Line:          2,
		Pattern:       "resources/postgres/data",
		GroupDir:      "resources/",
		TargetPattern: "postgres/data",
	}

	result, err := MoveGitignoreEntry(deps, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}

	// Verify entry was removed from root.
	rootContent := fs.Files["/repo/.gitignore"]
	if strings.Contains(rootContent, "resources/postgres/data") {
		t.Fatalf("entry should have been removed from root, got: %s", rootContent)
	}

	// Verify entry was added to group.
	groupContent, ok := fs.Files["/repo/resources/.gitignore"]
	if !ok {
		t.Fatal("expected group .gitignore to be created")
	}
	if !strings.Contains(groupContent, "postgres/data") {
		t.Fatalf("expected target entry in group .gitignore, got: %s", groupContent)
	}
}

func TestMoveEntry_CreatesGroupGitignore(t *testing.T) {
	fs := NewFakeFileIO().
		WithFile("/repo/.gitignore", "scenarios/foo/build\n")

	deps := HealthDeps{
		FS:      fs,
		RepoDir: "/repo",
		GroupingDeps: GroupingDeps{
			FS:         fs,
			ConfigPath: "/config/grouping-rules.json",
		},
	}

	req := GitignoreMoveRequest{
		Line:          1,
		Pattern:       "scenarios/foo/build",
		GroupDir:      "scenarios/foo/",
		TargetPattern: "build",
	}

	result, err := MoveGitignoreEntry(deps, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}

	groupContent, ok := fs.Files["/repo/scenarios/foo/.gitignore"]
	if !ok {
		t.Fatal("expected group .gitignore to be created")
	}
	if !strings.Contains(groupContent, "build") {
		t.Fatalf("expected 'build' in group .gitignore, got: %s", groupContent)
	}
}

func TestMoveEntry_StaleLineNumber(t *testing.T) {
	fs := NewFakeFileIO().
		WithFile("/repo/.gitignore", "only-one-line\n")

	deps := HealthDeps{
		FS:      fs,
		RepoDir: "/repo",
		GroupingDeps: GroupingDeps{
			FS:         fs,
			ConfigPath: "/config/grouping-rules.json",
		},
	}

	req := GitignoreMoveRequest{
		Line:          99,
		Pattern:       "only-one-line",
		GroupDir:      "some/",
		TargetPattern: "whatever",
	}

	result, err := MoveGitignoreEntry(deps, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Success {
		t.Fatal("expected failure for stale line number")
	}
	if !strings.Contains(result.Error, "out of range") {
		t.Fatalf("expected out of range error, got: %s", result.Error)
	}
}

func TestMoveEntry_PatternMismatch(t *testing.T) {
	fs := NewFakeFileIO().
		WithFile("/repo/.gitignore", "actual-pattern\n")

	deps := HealthDeps{
		FS:      fs,
		RepoDir: "/repo",
		GroupingDeps: GroupingDeps{
			FS:         fs,
			ConfigPath: "/config/grouping-rules.json",
		},
	}

	req := GitignoreMoveRequest{
		Line:          1,
		Pattern:       "expected-pattern",
		GroupDir:      "some/",
		TargetPattern: "whatever",
	}

	result, err := MoveGitignoreEntry(deps, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Success {
		t.Fatal("expected failure for pattern mismatch")
	}
	if !strings.Contains(result.Error, "pattern mismatch") {
		t.Fatalf("expected pattern mismatch error, got: %s", result.Error)
	}
}

func TestMoveEntry_InvalidGroupDir(t *testing.T) {
	fs := NewFakeFileIO().
		WithFile("/repo/.gitignore", "something\n")

	deps := HealthDeps{
		FS:      fs,
		RepoDir: "/repo",
		GroupingDeps: GroupingDeps{
			FS:         fs,
			ConfigPath: "/config/grouping-rules.json",
		},
	}

	req := GitignoreMoveRequest{
		Line:          1,
		Pattern:       "something",
		GroupDir:      "../../../etc",
		TargetPattern: "whatever",
	}

	result, err := MoveGitignoreEntry(deps, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Success {
		t.Fatal("expected failure for invalid group dir")
	}
	if !strings.Contains(result.Error, "invalid group directory") {
		t.Fatalf("expected invalid group directory error, got: %s", result.Error)
	}
}
