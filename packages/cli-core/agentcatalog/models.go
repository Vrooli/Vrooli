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

var (
	ErrModelDiscoveryUnavailable    = errors.New("model discovery unavailable")
	ErrModelCatalogNonAuthoritative = errors.New("model catalog is non-authoritative")
)

type LiveModelCatalog struct {
	SchemaVersion string   `json:"schema_version"`
	Runner        string   `json:"runner"`
	Models        []string `json:"models"`
	Source        string   `json:"source"`
	FetchedAt     string   `json:"fetched_at,omitempty"`
	Aliases       bool     `json:"aliases,omitempty"`
	// BinaryPath and RunnerVersion identify the executable that produced a
	// live catalog. Hosts can have a Vrooli shim and a separately installed
	// runner; without this identity a partial older shim can be mistaken for
	// the authoritative runner that Agent Manager will actually launch.
	BinaryPath    string `json:"binary_path,omitempty"`
	RunnerVersion string `json:"runner_version,omitempty"`
	// Authoritative distinguishes a live runner surface from a compatibility
	// cache. Consumers that make availability or policy claims must not treat a
	// non-authoritative cache as proof that a model is absent.
	Authoritative bool `json:"authoritative"`
	// Exhaustive is true only when the runner contract guarantees that every
	// selectable model is present in Models. Most runner discovery commands are
	// visibility or alias surfaces: they can prove a model is offered when it
	// appears, but omission is not proof that the model is unavailable. Always
	// emit this field so a partial listing cannot be mistaken for an inventory.
	Exhaustive bool `json:"exhaustive"`
}

// IsAuthoritative reports whether this catalog came from a runner surface.
// Catalogs constructed by older callers without Source/Authoritative metadata
// remain compatible when they contain models; named fallback sources must opt
// in explicitly and cannot be mistaken for live evidence.
func (c LiveModelCatalog) IsAuthoritative() bool {
	return c.Authoritative || (strings.TrimSpace(c.Source) == "" && len(c.Models) > 0)
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
		// The Codex cache is a compatibility fallback, not the authoritative
		// model surface. Older app-server versions can leave it populated with
		// a partial catalog even though the installed CLI accepts newer model
		// slugs (for example gpt-6-luna and gpt-6-sol). Ask the installed runner
		// first so policy validation can record what this machine visibly offers.
		// The runner may expose only a partial visibility surface; absence is not
		// an availability claim unless the result explicitly says exhaustive.
		if catalog, err := discoverCodexModels(ctx); err == nil {
			return catalog, nil
		}
		path, err := os.UserHomeDir()
		if err != nil {
			return LiveModelCatalog{}, fmt.Errorf("%w: resolve home directory: %v", ErrModelDiscoveryUnavailable, err)
		}
		catalog, cacheErr := readModelCatalogFile(runner, filepath.Join(path, ".codex", "models_cache.json"))
		if cacheErr != nil {
			return LiveModelCatalog{}, cacheErr
		}
		catalog.Authoritative = false
		return catalog, nil
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

func discoverCodexModels(ctx context.Context) (LiveModelCatalog, error) {
	binary, err := resolveRunnerBinary("codex")
	if err != nil {
		return LiveModelCatalog{}, fmt.Errorf("%w: codex: %v", ErrModelDiscoveryUnavailable, err)
	}
	cmd := exec.CommandContext(ctx, binary, "debug", "models")
	stdout, err := cmd.Output()
	if err != nil {
		if exitErr := new(exec.ExitError); errors.As(err, &exitErr) {
			stderr := strings.TrimSpace(string(exitErr.Stderr))
			if stderr != "" {
				return LiveModelCatalog{}, fmt.Errorf("%w: codex debug models: %s", ErrModelDiscoveryUnavailable, stderr)
			}
		}
		return LiveModelCatalog{}, fmt.Errorf("%w: codex debug models: %v", ErrModelDiscoveryUnavailable, err)
	}
	catalog, err := parseModelCatalog("codex", stdout, "codex debug models")
	if err != nil {
		return LiveModelCatalog{}, err
	}
	catalog.BinaryPath = binary
	catalog.RunnerVersion = runnerVersion(ctx, binary)
	return catalog, nil
}

// resolveRunnerBinary follows the same managed-runner rule used by Agent
// Manager: a Vrooli shim is a control-plane wrapper, not the executable whose
// model surface should be measured. Prefer the installed system binary when
// PATH resolves through .vrooli/shims so discovery and launch observe the
// same runner.
func resolveRunnerBinary(command string) (string, error) {
	path, err := exec.LookPath(command)
	if err != nil {
		return "", err
	}
	if command != "codex" {
		return path, nil
	}
	parts := strings.Split(filepath.ToSlash(filepath.Clean(path)), "/")
	for i, part := range parts {
		if part != ".vrooli" || i+1 >= len(parts) || parts[i+1] != "shims" {
			continue
		}
		for _, candidate := range []string{filepath.Join("/usr/bin", command), filepath.Join("/bin", command)} {
			if info, statErr := os.Stat(candidate); statErr == nil && info.Mode().IsRegular() && info.Mode()&0o111 != 0 {
				return candidate, nil
			}
		}
		break
	}
	return path, nil
}

func runnerVersion(ctx context.Context, binary string) string {
	command := exec.CommandContext(ctx, binary, "--version")
	output, err := command.Output()
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(output), "\n") {
		version := strings.TrimSpace(line)
		if version != "" && strings.Contains(strings.ToLower(version), "codex") && strings.ContainsAny(version, "0123456789") {
			return version
		}
	}
	return ""
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
		FetchedAt  string            `json:"fetched_at"`
		Models     []json.RawMessage `json:"models"`
		Source     string            `json:"source"`
		Exhaustive bool              `json:"exhaustive"`
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
	catalog := normalizeLiveCatalog(runner, models, source, payload.FetchedAt, false)
	catalog.Exhaustive = payload.Exhaustive
	return catalog, nil
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
	return LiveModelCatalog{SchemaVersion: ModelDiscoverySchemaVersion, Runner: runner, Models: clean, Source: source, FetchedAt: fetchedAt, Aliases: aliases, Authoritative: true}
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
