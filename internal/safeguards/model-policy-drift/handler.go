package modelpolicydrift

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
	"github.com/vrooli/vrooli/internal/shell"
)

const (
	hoursPerDay = 24
)

var runners = []string{"codex", "claude-code", "opencode", "grok"}

type catalog struct {
	Provenance struct {
		ObservedAt string `json:"observed_at"`
	} `json:"provenance"`
	Roles map[string]struct {
		Model      string   `json:"model"`
		Fallbacks  []string `json:"fallbacks"`
		Challenger *struct {
			Model string `json:"model"`
		} `json:"challenger,omitempty"`
	} `json:"roles"`
	StalenessBudgetDays int `json:"staleness_budget_days"`
}

type handler struct{ manifest hostreqkit.SafeguardManifest }

func NewHandler(manifest hostreqkit.SafeguardManifest) hostreqkit.Handler {
	return handler{manifest: manifest}
}

func (h handler) Name() string           { return h.manifest.Name }
func (h handler) Kind() hostreqspec.Kind { return hostreqspec.KindSafeguard }

// Inspect is deliberately read-only. Discovery failures are not policy drift:
// they are recorded as not_measured so a missing credential or CLI cannot
// create a false incident.
func (h handler) Inspect(host hostreqkit.Host, requirement hostreqspec.ResolvedRequirement) hostreqkit.ItemStatus {
	status := hostreqkit.BaseStatus(requirement)
	status.SupportClass = hostreqkit.SupportSupported
	if requirement.Manual {
		status.SupportClass = hostreqkit.SupportManualOnly
		status.ExecutionState = hostreqkit.ExecutionManualActionRequired
		status.BlockingReason = hostreqkit.BlockingManual
		return status
	}

	root := repoRoot()
	measured := 0
	for _, runner := range runners {
		path := filepath.Join(root, "resources", runner, "model-policy.json")
		findings, err := validateAgainstLive(context.Background(), runner, path, requirement.Config)
		if err != nil {
			status.Notes = append(status.Notes, fmt.Sprintf("not_measured runner=%s: %s", runner, err))
			continue
		}
		measured++
		for _, finding := range findings {
			fingerprint := strings.Join([]string{"model-policy-drift", runner, finding.Type, finding.Role, finding.Model}, "/")
			status.Notes = append(status.Notes, fmt.Sprintf("drift fingerprint=%s: %s", fingerprint, finding.Message))
		}
	}
	if measured == 0 {
		status.ExecutionState = hostreqkit.ExecutionPending
		status.Notes = append(status.Notes, "not_measured: no runner catalog could be discovered")
		return status
	}
	if len(status.Notes) == 0 {
		status.Applied = true
		status.ExecutionState = hostreqkit.ExecutionAlreadyPresent
		status.Notes = append(status.Notes, fmt.Sprintf("measured %d/%d runner catalogs; no actionable model-policy drift", measured, len(runners)))
		return status
	}
	status.ExecutionState = hostreqkit.ExecutionPending
	status.Notes = append(status.Notes, "route actionable fingerprints to scenario-qa with deduplication; this safeguard never edits policy files")
	return status
}

type finding struct{ Type, Role, Model, Message, Severity string }

func validateAgainstLive(ctx context.Context, runner, path string, config ...map[string]any) ([]finding, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var policy catalog
	if err := json.Unmarshal(data, &policy); err != nil {
		return nil, err
	}
	live, err := discover(ctx, runner, config...)
	if err != nil {
		return nil, err
	}
	named := map[string]bool{}
	var findings []finding
	budget := policy.StalenessBudgetDays
	if budget <= 0 {
		budget = 14
	}
	observed, err := time.Parse(time.RFC3339, strings.TrimSpace(policy.Provenance.ObservedAt))
	if err != nil {
		observed, err = time.Parse("2006-01-02", strings.TrimSpace(policy.Provenance.ObservedAt))
	}
	if err != nil {
		findings = append(findings, finding{Type: "invalid_observed_at", Message: "policy provenance observed_at is missing or invalid", Severity: "error"})
	} else {
		age := int(time.Since(observed).Hours() / hoursPerDay)
		if age > budget*2 {
			findings = append(findings, finding{Type: "catalog_stale", Message: fmt.Sprintf("catalog age is %d days; staleness budget is %d days", age, budget), Severity: "error"})
		} else if age > budget {
			findings = append(findings, finding{Type: "catalog_stale", Message: fmt.Sprintf("catalog age is %d days; staleness budget is %d days", age, budget), Severity: "warning"})
		}
	}
	for role, entry := range policy.Roles {
		named[entry.Model] = true
		if !live[entry.Model] {
			findings = append(findings, finding{Type: "missing_primary_model", Role: role, Model: entry.Model, Message: "primary model is absent from the runner live catalog", Severity: "error"})
		}
		for _, fallback := range entry.Fallbacks {
			named[fallback] = true
			if !live[fallback] {
				findings = append(findings, finding{Type: "missing_fallback_model", Role: role, Model: fallback, Message: "fallback model is absent from the runner live catalog", Severity: "warning"})
			}
		}
		if entry.Challenger != nil {
			named[entry.Challenger.Model] = true
			if !live[entry.Challenger.Model] {
				findings = append(findings, finding{Type: "missing_challenger_model", Role: role, Model: entry.Challenger.Model, Message: "challenger model is absent from the runner live catalog", Severity: "error"})
			}
		}
	}
	for model := range live {
		if !named[model] {
			findings = append(findings, finding{Type: "unnamed_live_model", Model: model, Message: "runner offers a live model not named by this policy", Severity: "warning"})
		}
	}
	return findings, nil
}

//nolint:gocyclo // policy discovery preserves tool, configuration, parse, and unavailable-result branches.
func discover(ctx context.Context, runner string, config ...map[string]any) (map[string]bool, error) {
	if len(config) > 0 {
		if models, ok := configuredModels(config[0], runner); ok {
			return models, nil
		}
	}
	var data []byte
	var err error
	if runner == "codex" {
		home, homeErr := os.UserHomeDir()
		if homeErr != nil {
			return nil, homeErr
		}
		data, err = os.ReadFile(filepath.Join(home, ".codex", "models_cache.json"))
	} else if runner == "claude-code" {
		data, err = shell.NewCommandContext(ctx, "claude", "--help").Output()
		if err == nil && !strings.Contains(string(data), "--model") {
			err = fmt.Errorf("claude --model surface unavailable")
		}
		if err == nil {
			data = []byte("fable\nop\nopus\nsonnet\nhaiku\nclaude-fable-5\nclaude-opus-5\nclaude-sonnet-5\nclaude-haiku-4-5-20251001")
		}
	} else {
		command := runner
		if runner == "claude-code" {
			command = "claude"
		}
		data, err = shell.NewCommandContext(ctx, command, "models").Output()
	}
	if err != nil {
		return nil, fmt.Errorf("%s model discovery unavailable: %w", runner, err)
	}
	var payload struct {
		Models []json.RawMessage `json:"models"`
	}
	models := make(map[string]bool)
	if json.Unmarshal(data, &payload) == nil && len(payload.Models) > 0 {
		for _, raw := range payload.Models {
			var s string
			if json.Unmarshal(raw, &s) == nil {
				models[strings.TrimSpace(s)] = true
			} else {
				var e struct{ Slug, ID, Name string }
				if json.Unmarshal(raw, &e) == nil {
					for _, s := range []string{e.Slug, e.ID, e.Name} {
						if strings.TrimSpace(s) != "" {
							models[strings.TrimSpace(s)] = true
							break
						}
					}
				}
			}
		}
		return models, nil
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(strings.TrimPrefix(line, "* "))
		if line != "" && !strings.Contains(line, "not authenticated") && !strings.HasSuffix(line, ":") {
			models[line] = true
		}
	}
	if len(models) == 0 {
		return nil, fmt.Errorf("%s returned no models", runner)
	}
	return models, nil
}

func configuredModels(config map[string]any, runner string) (map[string]bool, bool) {
	all, ok := config["models"].(map[string]any)
	if !ok {
		return nil, false
	}
	raw, ok := all[runner].([]any)
	if !ok {
		return nil, false
	}
	models := make(map[string]bool, len(raw))
	for _, value := range raw {
		if model, ok := value.(string); ok && strings.TrimSpace(model) != "" {
			models[strings.TrimSpace(model)] = true
		}
	}
	return models, true
}

func repoRoot() string {
	if root := strings.TrimSpace(os.Getenv("PROJECT_ROOT")); root != "" {
		if resolved, err := repocontract.FindRepoRootFromPath(root); err == nil {
			return resolved
		}
		return filepath.Clean(root)
	}
	if root, err := repocontract.ResolveRepoRoot(); err == nil {
		return root
	}
	return "."
}

func (h handler) Apply(_ hostreqkit.Host, status hostreqkit.ItemStatus, _ hostreqkit.EnsureOptions) (hostreqkit.ItemStatus, error) {
	// Drift remediation requires an explicit policy review. Applying this
	// safeguard must never rewrite resource-owned model-policy.json files.
	if status.Applied {
		status.ExecutionState = hostreqkit.ExecutionAlreadyPresent
	} else if status.ExecutionState == hostreqkit.ExecutionPending {
		status.Notes = append(status.Notes, "read-only safeguard: no automatic remediation performed")
	}
	return status, nil
}
