package agentcatalog

// Package agentcatalog contains the read-only model catalog contracts and discovery engine shared by control-plane consumers.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

const ModelDiscoverySchemaVersion = "v1"

var ErrModelDiscoveryUnavailable = errors.New("model discovery unavailable")

type LiveModelCatalog struct {
	SchemaVersion string   `json:"schema_version"`
	Runner        string   `json:"runner"`
	Models        []string `json:"models"`
	Source        string   `json:"source"`
	FetchedAt     string   `json:"fetched_at,omitempty"`
	Aliases       bool     `json:"aliases,omitempty"`
}

// ModelResolution is the resource-owned answer for one runner model. The
// control plane treats these strings as opaque and never derives one from the
// other.
type ModelResolution struct {
	SchemaVersion  string `json:"schema_version"`
	Runner         string `json:"runner"`
	Model          string `json:"model"`
	CanonicalModel string `json:"canonical_model"`
	Provider       string `json:"provider,omitempty"`
	Source         string `json:"source,omitempty"`
	PolicyPath     string `json:"policy_path,omitempty"`
	PolicyDigest   string `json:"policy_digest,omitempty"`
}

type ModelDiscoveryFunc func(context.Context) (LiveModelCatalog, error)

func DiscoverModels(ctx context.Context, runner string) (LiveModelCatalog, error) {
	runner = strings.TrimSpace(runner)
	if runner == "" {
		return LiveModelCatalog{}, fmt.Errorf("%w: runner is required", ErrModelDiscoveryUnavailable)
	}
	if override := strings.TrimSpace(os.Getenv(discoveryOverrideEnv(runner))); override != "" {
		return readModelCatalogFile(runner, override)
	}
	if inline := strings.TrimSpace(os.Getenv(discoveryInlineEnv(runner))); inline != "" {
		return parseModelCatalog(runner, []byte(inline), "environment override")
	}

	switch runner {
	case "codex":
		path, err := os.UserHomeDir()
		if err != nil {
			return LiveModelCatalog{}, fmt.Errorf("%w: resolve home directory: %v", ErrModelDiscoveryUnavailable, err)
		}
		return readModelCatalogFile(runner, filepath.Join(path, ".codex", "models_cache.json"))
	case "claude-code":
		return discoverClaudeAliases(ctx)
	case "opencode":
		return discoverCommandModels(ctx, runner, "opencode", "models")
	case "grok":
		return discoverCommandModels(ctx, runner, "grok", "models")
	default:
		return LiveModelCatalog{}, fmt.Errorf("%w: no discovery adapter for runner %q", ErrModelDiscoveryUnavailable, runner)
	}
}

func discoveryOverrideEnv(runner string) string {
	return "VROOLI_" + strings.ToUpper(strings.ReplaceAll(runner, "-", "_")) + "_MODELS_FILE"
}

func discoveryInlineEnv(runner string) string {
	return "VROOLI_" + strings.ToUpper(strings.ReplaceAll(runner, "-", "_")) + "_MODELS"
}

func readModelCatalogFile(runner, path string) (LiveModelCatalog, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return LiveModelCatalog{}, fmt.Errorf("%w: read %s model catalog %s: %v", ErrModelDiscoveryUnavailable, runner, path, err)
	}
	return parseModelCatalog(runner, data, path)
}

func parseModelCatalog(runner string, data []byte, source string) (LiveModelCatalog, error) {
	var payload struct {
		FetchedAt string            `json:"fetched_at"`
		Models    []json.RawMessage `json:"models"`
		Source    string            `json:"source"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		// Inline overrides and command fixtures may be newline-separated model ids.
		var models []string
		for _, line := range strings.Split(string(data), "\n") {
			if model := strings.TrimSpace(line); model != "" {
				models = append(models, strings.TrimPrefix(model, "* "))
			}
		}
		if len(models) == 0 {
			return LiveModelCatalog{}, fmt.Errorf("%w: parse %s: %v", ErrModelDiscoveryUnavailable, source, err)
		}
		return normalizeLiveCatalog(runner, models, source, "", false), nil
	}
	models := make([]string, 0, len(payload.Models))
	for _, raw := range payload.Models {
		var model string
		if json.Unmarshal(raw, &model) == nil {
			models = append(models, model)
			continue
		}
		var entry struct {
			Slug string `json:"slug"`
			ID   string `json:"id"`
			Name string `json:"name"`
		}
		if err := json.Unmarshal(raw, &entry); err != nil {
			return LiveModelCatalog{}, fmt.Errorf("%w: parse model entry in %s: %v", ErrModelDiscoveryUnavailable, source, err)
		}
		for _, candidate := range []string{entry.Slug, entry.ID, entry.Name} {
			if strings.TrimSpace(candidate) != "" {
				models = append(models, candidate)
				break
			}
		}
	}
	if len(models) == 0 {
		return LiveModelCatalog{}, fmt.Errorf("%w: %s contained no models", ErrModelDiscoveryUnavailable, source)
	}
	if payload.Source != "" {
		source = payload.Source + " (" + source + ")"
	}
	return normalizeLiveCatalog(runner, models, source, payload.FetchedAt, false), nil
}

func discoverClaudeAliases(ctx context.Context) (LiveModelCatalog, error) {
	command := exec.CommandContext(ctx, "claude", "--help")
	output, err := command.Output()
	if err != nil {
		return LiveModelCatalog{}, fmt.Errorf("%w: claude --help: %v", ErrModelDiscoveryUnavailable, err)
	}
	text := string(output)
	if !strings.Contains(text, "--model") {
		return LiveModelCatalog{}, fmt.Errorf("%w: claude help has no --model surface", ErrModelDiscoveryUnavailable)
	}
	// Read examples from the installed CLI's own help text. Claude's alias
	// vocabulary changes independently of this package, so keeping a second
	// list here would create false drift and stale policy health.
	return normalizeLiveCatalog("claude-code", extractModelExamples(text), "claude --help --model alias surface", time.Now().UTC().Format(time.RFC3339), true), nil
}

var modelExamplePattern = regexp.MustCompile(`['"]([^'"]+)['"]`)

func extractModelExamples(help string) []string {
	lines := strings.Split(help, "\n")
	models := make([]string, 0)
	inModelOption := false
	for _, line := range lines {
		if strings.Contains(line, "--model <model>") {
			inModelOption = true
		} else if inModelOption && (strings.HasPrefix(line, "  --") || strings.HasPrefix(line, "  -")) {
			break
		}
		if !inModelOption {
			continue
		}
		for _, match := range modelExamplePattern.FindAllStringSubmatch(line, -1) {
			if len(match) == 2 && strings.TrimSpace(match[1]) != "" {
				models = append(models, match[1])
			}
		}
	}
	return models
}

func discoverCommandModels(ctx context.Context, runner, command string, args ...string) (LiveModelCatalog, error) {
	cmd := exec.CommandContext(ctx, command, args...)
	stdout, err := cmd.Output()
	if err != nil {
		if exitErr := new(exec.ExitError); errors.As(err, &exitErr) {
			stderr := strings.TrimSpace(string(exitErr.Stderr))
			if stderr != "" {
				return LiveModelCatalog{}, fmt.Errorf("%w: %s %s: %s", ErrModelDiscoveryUnavailable, command, strings.Join(args, " "), stderr)
			}
		}
		return LiveModelCatalog{}, fmt.Errorf("%w: %s %s: %v", ErrModelDiscoveryUnavailable, command, strings.Join(args, " "), err)
	}
	models := make([]string, 0)
	for _, line := range strings.Split(string(stdout), "\n") {
		line = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "*"))
		if strings.HasPrefix(line, "Default model:") {
			line = strings.TrimSpace(strings.TrimPrefix(line, "Default model:"))
		}
		if strings.HasSuffix(line, " (default)") {
			line = strings.TrimSpace(strings.TrimSuffix(line, " (default)"))
		}
		if line == "" || strings.HasSuffix(line, ":") || strings.Contains(line, "not authenticated") || strings.Contains(line, "Available models") {
			continue
		}
		if line != "" {
			models = append(models, line)
		}
	}
	if len(models) == 0 {
		return LiveModelCatalog{}, fmt.Errorf("%w: %s %s returned no models", ErrModelDiscoveryUnavailable, command, strings.Join(args, " "))
	}
	return normalizeLiveCatalog(runner, models, command+" "+strings.Join(args, " "), time.Now().UTC().Format(time.RFC3339), false), nil
}

func normalizeLiveCatalog(runner string, models []string, source, fetchedAt string, aliases bool) LiveModelCatalog {
	seen := make(map[string]struct{}, len(models))
	clean := make([]string, 0, len(models))
	for _, model := range models {
		model = strings.TrimSpace(model)
		if model == "" {
			continue
		}
		if _, ok := seen[model]; ok {
			continue
		}
		seen[model] = struct{}{}
		clean = append(clean, model)
	}
	sort.Strings(clean)
	return LiveModelCatalog{SchemaVersion: ModelDiscoverySchemaVersion, Runner: runner, Models: clean, Source: source, FetchedAt: fetchedAt, Aliases: aliases}
}

func (c LiveModelCatalog) Contains(model string) bool {
	model = strings.TrimSpace(model)
	for _, candidate := range c.Models {
		if candidate == model {
			return true
		}
	}
	return false
}

func (c LiveModelCatalog) Write(w io.Writer) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(w, string(data))
	return err
}
