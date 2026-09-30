package modelpolicydrift

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vrooli/cli-core/agentcatalog"
	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

const (
	hoursPerDay = 24
)

var runners = []string{"codex", "claude-code", "opencode", "grok"}

type catalog struct {
	Provenance struct {
		ObservedAt string `json:"observed_at"`
	} `json:"provenance"`
	Roles               map[string]agentcatalog.CodingRole `json:"roles"`
	ExcludedModels      []string                           `json:"excluded_models"`
	LiveCatalogMode     string                             `json:"live_catalog_mode"`
	StalenessBudgetDays int                                `json:"staleness_budget_days"`
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
	actionable := false
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
			if finding.Severity == "error" {
				actionable = true
			}
		}
	}
	if measured == 0 {
		status.ExecutionState = hostreqkit.ExecutionPending
		status.Notes = append(status.Notes, "not_measured: no runner catalog could be discovered")
		return status
	}
	if !actionable {
		status.Applied = true
		status.ExecutionState = hostreqkit.ExecutionAlreadyPresent
		status.Notes = append(status.Notes, fmt.Sprintf("measured %d/%d runner catalogs; no blocking model-policy drift", measured, len(runners)))
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
	liveCatalog, err := discoverCatalog(ctx, runner, config...)
	if err != nil {
		return nil, err
	}
	if !liveCatalog.IsAuthoritative() {
		return nil, fmt.Errorf("%s model discovery returned non-authoritative source %q", runner, liveCatalog.Source)
	}
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
	policyCatalog := agentcatalog.CodingRoleCatalog{Roles: policy.Roles, ExcludedModels: policy.ExcludedModels, LiveCatalogMode: policy.LiveCatalogMode}
	for _, modelFinding := range agentcatalog.LiveCatalogFindings(policyCatalog, liveCatalog) {
		findings = append(findings, finding{Type: modelFinding.Type, Role: modelFinding.Role, Model: modelFinding.Model, Message: modelFinding.Message, Severity: modelFinding.Severity})
	}
	return findings, nil
}

// discover is intentionally a thin adapter over the shared agent catalog.
// Keeping model discovery in one package prevents safeguards from silently
// falling back to a stale runner cache while the installed CLI accepts newer
// model slugs (for example gpt-6-luna and gpt-6-sol). The shared comparator
// treats an unlisted model as unconfirmed unless the runner proves its catalog
// is exhaustive, so a partial listing cannot trigger a policy downgrade.
func discover(ctx context.Context, runner string, config ...map[string]any) (map[string]bool, error) {
	catalog, err := discoverCatalog(ctx, runner, config...)
	if err != nil {
		return nil, err
	}
	models := make(map[string]bool, len(catalog.Models))
	for _, model := range catalog.Models {
		models[model] = true
	}
	return models, nil
}

func discoverCatalog(ctx context.Context, runner string, config ...map[string]any) (agentcatalog.LiveModelCatalog, error) {
	if len(config) > 0 {
		if models, ok := configuredModels(config[0], runner); ok {
			list := make([]string, 0, len(models))
			for model := range models {
				list = append(list, model)
			}
			exhaustive := true
			if value, ok := config[0]["models_exhaustive"].(bool); ok {
				exhaustive = value
			}
			return agentcatalog.LiveModelCatalog{Runner: runner, Models: list, Source: "configured test catalog", Authoritative: true, Exhaustive: exhaustive}, nil
		}
	}
	catalog, err := agentcatalog.DiscoverModels(ctx, runner)
	if err != nil {
		return agentcatalog.LiveModelCatalog{}, fmt.Errorf("%s model discovery unavailable: %w", runner, err)
	}
	return catalog, nil
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
