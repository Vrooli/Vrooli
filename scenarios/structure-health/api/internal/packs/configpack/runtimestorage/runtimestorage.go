package runtimestorage

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

/*
Rule: No Legacy Runtime Storage Paths
Description: Prevent scenarios from storing mutable runtime state under repo-local data paths or legacy user-home resource directories
Reason: Scenario source trees are deployable inputs, not runtime storage roots. Repo-local and ad hoc home-scoped runtime paths break the storage standard and make deployments non-portable.
Category: config
Severity: high
Standard: storage-v1
Targets: api, cli, ui, test

<test-case id="scenario-go-relative-data-path" should-fail="true" path="api/main.go">
  <description>Go code should not write runtime state into ../data</description>
  <input language="go">
package main

import "os"

func save() error {
	return os.WriteFile("../data/tasks/task.json", []byte("x"), 0o644)
}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>Repo-local runtime storage</expected-message>
</test-case>

<test-case id="scenario-shell-root-data-path" should-fail="true" path="cli/start.sh">
  <description>Shell code should not rely on repo-root data directories for runtime state</description>
  <input language="bash">
#!/usr/bin/env bash
mkdir -p "${APP_ROOT}/data/session-profiles"
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>${APP_ROOT}/data</expected-message>
</test-case>

<test-case id="scenario-storage-package-usage" should-fail="false" path="api/main.go">
  <description>Scenario code using api-core storage is allowed</description>
  <input language="go">
package main

import "github.com/vrooli/api-core/storage"

func save(store *storage.Storage) (string, error) {
	return store.Path(storage.ClassData, "tasks/task.json")
}
  </input>
</test-case>
*/

var repoLocalStoragePatterns = []repoLocalStoragePattern{
	{
		pattern: regexp.MustCompile(`\$\{APP_ROOT\}/data(?:/|")`),
		label:   "${APP_ROOT}/data",
	},
	{
		pattern: regexp.MustCompile(`\$\{ROOT\}/data(?:/|")`),
		label:   "${ROOT}/data",
	},
	{
		pattern: regexp.MustCompile(`\$\{VROOLI_ROOT\}/data(?:/|")`),
		label:   "${VROOLI_ROOT}/data",
	},
	{
		pattern: regexp.MustCompile(`\.\./data(?:/|")`),
		label:   "../data",
	},
	{
		pattern: regexp.MustCompile(`\./data(?:/|")`),
		label:   "./data",
	},
	{
		pattern: regexp.MustCompile(`filepath\.(?:Join|Clean)\(\s*"data"`),
		label:   `filepath.Join("data", ...)`,
	},
	{
		pattern: regexp.MustCompile(`filepath\.Join\(\s*"\.\."\s*,\s*"data"`),
		label:   `filepath.Join("..", "data", ...)`,
	},
	{
		pattern: regexp.MustCompile(`\$\{HOME\}/\.(?:browserless|comfyui|minio|ollama|qdrant|searxng|whisper)(?:/|")`),
		label:   "${HOME}/.<resource>",
	},
	{
		pattern: regexp.MustCompile(`\$HOME/\.(?:browserless|comfyui|minio|ollama|qdrant|searxng|whisper)(?:/|")`),
		label:   "$HOME/.<resource>",
	},
	{
		pattern: regexp.MustCompile(`~/\.(?:browserless|comfyui|minio|ollama|qdrant|searxng|whisper)(?:/|")`),
		label:   "~/.<resource>",
	},
}

type repoLocalStoragePattern struct {
	pattern *regexp.Regexp
	label   string
}

// CheckRepoLocalRuntimeStorage detects scenario code that writes runtime state into repo-local data paths
// or legacy home-scoped resource directories.
func CheckRepoLocalRuntimeStorage(content []byte, filePath string, scenario string) []Violation {
	_ = scenario

	if shouldSkipRepoLocalStorageFile(filePath) {
		return nil
	}

	lines := strings.Split(string(content), "\n")
	violations := []Violation{}
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "#") {
			continue
		}
		for _, candidate := range repoLocalStoragePatterns {
			if !candidate.pattern.MatchString(line) {
				continue
			}
			violations = append(violations, Violation{
				Type:           "repo_local_runtime_storage",
				Severity:       "high",
				Title:          "Repo-local runtime storage",
				Message:        fmt.Sprintf("Mutable runtime state should not use legacy storage path %s in scenario code", candidate.label),
				Description:    "Scenario runtime state belongs in api-core/storage-backed classed directories, not under repo-local paths or legacy home-scoped resource directories.",
				File:           filepath.Base(filePath),
				FilePath:       filePath,
				Line:           i + 1,
				LineNumber:     i + 1,
				CodeSnippet:    strings.TrimSpace(line),
				Recommendation: "Use github.com/vrooli/api-core/storage for mutable filesystem state and keep durable resource-backed persistence declared in .vrooli/service.json.",
				Standard:       "storage-v1",
				Category:       "config",
			})
			break
		}
	}
	return violations
}

func shouldSkipRepoLocalStorageFile(filePath string) bool {
	normalized := filepath.ToSlash(filePath)
	if strings.HasSuffix(normalized, "_test.go") {
		return true
	}
	for _, segment := range []string{
		"/fixtures/",
		"/testdata/",
		"/vendor/",
		"/docs/",
	} {
		if strings.Contains(normalized, segment) {
			return true
		}
	}
	return false
}
