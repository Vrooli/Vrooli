package agentcatalog

// This file defines the transport contract shared by coding-agent resources.
// Catalog content stays beside its owning resource; this package has no model
// inventory and deliberately knows nothing about Agent Manager.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"
)

const CodingRolePolicySchemaVersion = "v1"

// OverrideFlag is the shared explicit-authorization spelling used by control
// plane commands that expose a mutating policy operation.
const OverrideFlag = "--i-was-explicitly-authorized"

type CodingRoleCatalog struct {
	SchemaVersion string                `json:"schema_version"`
	Runner        string                `json:"runner"`
	Roles         map[string]CodingRole `json:"roles"`
	// ModelAliases is the resource-owned translation table from the runner's
	// model vocabulary to the canonical identity used by pricing providers.
	// Agent Manager consumes this through the resource CLI; it never owns this
	// vocabulary.
	ModelAliases        map[string]ModelAlias `json:"model_aliases,omitempty"`
	Provenance          CatalogProvenance     `json:"provenance"`
	Billing             *Billing              `json:"billing,omitempty"`
	StalenessBudgetDays int                   `json:"staleness_budget_days,omitempty"`
}

type Billing struct {
	Mode        string `json:"mode"`
	Provider    string `json:"provider,omitempty"`
	AccountRef  string `json:"account_ref,omitempty"`
	PlanRef     string `json:"plan_ref,omitempty"`
	PlanLabel   string `json:"plan_label,omitempty"`
	QuotaWindow string `json:"quota_window,omitempty"`
	Source      string `json:"source,omitempty"`
}

type CodingRole struct {
	Model          string      `json:"model"`
	CanonicalModel string      `json:"canonical_model,omitempty"`
	Fallbacks      []string    `json:"fallbacks,omitempty"`
	Description    string      `json:"description"`
	Capabilities   []string    `json:"capabilities"`
	Challenger     *Challenger `json:"challenger,omitempty"`
}

// ModelAlias is deliberately generic. Resources decide whether their
// canonical identity is a provider/model slug, a local model ID, or another
// pricing-provider key.
type ModelAlias struct {
	CanonicalModel string `json:"canonical_model"`
	Provider       string `json:"provider,omitempty"`
}

type Challenger struct {
	Model      string  `json:"model"`
	SampleRate float64 `json:"sample_rate"`
}

type CatalogProvenance struct {
	Source     string `json:"source"`
	ObservedAt string `json:"observed_at"`
}

// CatalogFreshness is the small, runner-neutral health record used by
// control-plane surfaces. It carries age and policy provenance without
// exposing the resource's role/model vocabulary.
type CatalogFreshness struct {
	Runner     string `json:"runner"`
	Source     string `json:"source,omitempty"`
	ObservedAt string `json:"observed_at,omitempty"`
	AgeDays    int    `json:"age_days"`
	BudgetDays int    `json:"budget_days"`
	Status     string `json:"status"`
	PolicyPath string `json:"policy_path,omitempty"`
	Error      string `json:"error,omitempty"`
}

const DefaultStalenessBudgetDays = 14

type PolicyValidationFinding struct {
	Type     string `json:"type"`
	Severity string `json:"severity"`
	Role     string `json:"role,omitempty"`
	Model    string `json:"model,omitempty"`
	Message  string `json:"message"`
}

type PolicyValidationError struct {
	Code string
	Err  error
}

func (e *PolicyValidationError) Error() string { return e.Code + ": " + e.Err.Error() }
func (e *PolicyValidationError) Unwrap() error { return e.Err }

func catalogStalenessFindings(c CodingRoleCatalog, now time.Time, againstLive bool) []PolicyValidationFinding {
	observed, err := parseObservedAt(c.Provenance.ObservedAt)
	if err != nil {
		return []PolicyValidationFinding{{Type: "invalid_observed_at", Severity: "error", Message: err.Error()}}
	}
	budget := c.StalenessBudgetDays
	if budget <= 0 {
		budget = DefaultStalenessBudgetDays
	}
	age := int(now.Sub(observed).Hours() / 24)
	if age <= budget {
		return nil
	}
	severity := "warning"
	if againstLive && age > budget*2 {
		severity = "error"
	}
	return []PolicyValidationFinding{{Type: "catalog_stale", Severity: severity, Message: fmt.Sprintf("catalog age is %d days; staleness budget is %d days", age, budget)}}
}

func catalogIsHardStale(c CodingRoleCatalog, now time.Time) bool {
	observed, err := parseObservedAt(c.Provenance.ObservedAt)
	if err != nil {
		return true
	}
	budget := c.StalenessBudgetDays
	if budget <= 0 {
		budget = DefaultStalenessBudgetDays
	}
	return now.Sub(observed) > time.Duration(budget*2)*24*time.Hour
}

func parseObservedAt(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, errors.New("provenance.observed_at is required")
	}
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return parsed, nil
	}
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		return time.Time{}, fmt.Errorf("provenance.observed_at must be YYYY-MM-DD or RFC3339: %w", err)
	}
	return parsed.UTC(), nil
}

// ParseObservedAt exposes the catalog timestamp grammar to resource command
// tests without exposing the catalog loader internals.
func ParseObservedAt(value string) (time.Time, error) { return parseObservedAt(value) }

func liveCatalogFindings(c CodingRoleCatalog, live LiveModelCatalog) []PolicyValidationFinding {
	findings := make([]PolicyValidationFinding, 0)
	named := make(map[string]struct{})
	for roleName, role := range c.Roles {
		named[role.Model] = struct{}{}
		if !live.Contains(role.Model) && !live.Aliases {
			findings = append(findings, PolicyValidationFinding{Type: "missing_primary_model", Severity: "error", Role: roleName, Model: role.Model, Message: "primary model is absent from the runner live catalog"})
		}
		for _, fallback := range role.Fallbacks {
			named[fallback] = struct{}{}
			if !live.Contains(fallback) && !live.Aliases {
				findings = append(findings, PolicyValidationFinding{Type: "missing_fallback_model", Severity: "warning", Role: roleName, Model: fallback, Message: "fallback model is absent from the runner live catalog"})
			}
		}
		if role.Challenger != nil {
			named[role.Challenger.Model] = struct{}{}
			if !live.Contains(role.Challenger.Model) && !live.Aliases {
				findings = append(findings, PolicyValidationFinding{Type: "missing_challenger_model", Severity: "error", Role: roleName, Model: role.Challenger.Model, Message: "challenger model is absent from the runner live catalog"})
			}
		}
	}
	for _, model := range live.Models {
		if _, ok := named[model]; !ok {
			findings = append(findings, PolicyValidationFinding{Type: "unnamed_live_model", Severity: "warning", Model: model, Message: "runner offers a live model not named by this policy"})
		}
	}
	sort.Slice(findings, func(i, j int) bool {
		return findings[i].Type+findings[i].Role+findings[i].Model < findings[j].Type+findings[j].Role+findings[j].Model
	})
	return findings
}

func loadCodingRoleCatalog(runner, path string) (CodingRoleCatalog, []byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return CodingRoleCatalog{}, nil, fmt.Errorf("read coding role policy %s: %w", path, err)
	}
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	var catalog CodingRoleCatalog
	if err := decoder.Decode(&catalog); err != nil {
		return CodingRoleCatalog{}, nil, fmt.Errorf("parse coding role policy %s: %w", path, err)
	}
	if err := validateCodingRoleCatalog(catalog, runner); err != nil {
		return CodingRoleCatalog{}, nil, fmt.Errorf("invalid coding role policy %s: %w", path, err)
	}
	return catalog, data, nil
}

func validateCodingRoleCatalog(c CodingRoleCatalog, expectedRunner string) error {
	var errs []error
	if c.SchemaVersion != CodingRolePolicySchemaVersion {
		errs = append(errs, fmt.Errorf("schema_version must be %q", CodingRolePolicySchemaVersion))
	}
	if c.Runner != expectedRunner {
		errs = append(errs, fmt.Errorf("runner must be %q", expectedRunner))
	}
	if strings.TrimSpace(c.Provenance.Source) == "" || strings.TrimSpace(c.Provenance.ObservedAt) == "" {
		errs = append(errs, errors.New("provenance.source and provenance.observed_at are required"))
	}
	if _, err := parseObservedAt(c.Provenance.ObservedAt); err != nil {
		errs = append(errs, err)
	}
	for _, required := range []string{"code.default", "code.fast", "code.smart", "code.cheap"} {
		r, ok := c.Roles[required]
		if !ok {
			errs = append(errs, fmt.Errorf("missing required role %q", required))
			continue
		}
		if strings.TrimSpace(r.Model) == "" || strings.TrimSpace(r.Description) == "" || len(r.Capabilities) == 0 {
			errs = append(errs, fmt.Errorf("role %q requires model, description, and capabilities", required))
		}
	}
	for name, role := range c.Roles {
		if !hasAllowedCodingRoleNamespace(name) {
			errs = append(errs, fmt.Errorf("role %q must use an allowed namespace (code.* or write.*)", name))
		}
		if strings.TrimSpace(role.Model) == "" {
			errs = append(errs, fmt.Errorf("role %q has an empty model", name))
		}
		if role.Challenger != nil && (strings.TrimSpace(role.Challenger.Model) == "" || role.Challenger.SampleRate < 0 || role.Challenger.SampleRate > 1) {
			errs = append(errs, fmt.Errorf("role %q challenger requires a model and sample_rate between 0 and 1", name))
		}
	}
	return errors.Join(errs...)
}

func hasAllowedCodingRoleNamespace(name string) bool {
	for _, prefix := range []string{"code.", "write."} {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

// LoadCodingRoleCatalog returns a validated catalog and its original bytes.
func LoadCodingRoleCatalog(runner, path string) (CodingRoleCatalog, []byte, error) {
	return loadCodingRoleCatalog(runner, path)
}

// CatalogStalenessFindings exposes the shared staleness evaluation to command surfaces.
func CatalogStalenessFindings(c CodingRoleCatalog, now time.Time, againstLive bool) []PolicyValidationFinding {
	return catalogStalenessFindings(c, now, againstLive)
}

// CatalogIsHardStale reports whether the catalog exceeds twice its declared budget.
func CatalogIsHardStale(c CodingRoleCatalog, now time.Time) bool {
	return catalogIsHardStale(c, now)
}

// LiveCatalogFindings exposes the shared live-catalog comparison to command surfaces.
func LiveCatalogFindings(c CodingRoleCatalog, live LiveModelCatalog) []PolicyValidationFinding {
	return liveCatalogFindings(c, live)
}

// ReadCodingRoleCatalog returns a validated resource-owned catalog for
// control-plane readers that need provenance or model resolution. Callers
// still receive the full resource contract; no Agent Manager vocabulary is
// introduced here.
func ReadCodingRoleCatalog(runner, path string) (CodingRoleCatalog, error) {
	catalog, _, err := loadCodingRoleCatalog(runner, path)
	return catalog, err
}

// ReadCatalogFreshness reads only catalog metadata for a health/status view.
// It intentionally uses the same strict loader as policy validation so a
// malformed catalog cannot appear healthy merely because its file exists.
func ReadCatalogFreshness(runner, path string, now time.Time) CatalogFreshness {
	result := CatalogFreshness{Runner: runner, PolicyPath: path, Status: "unknown"}
	catalog, _, err := loadCodingRoleCatalog(runner, path)
	if err != nil {
		result.Status = "invalid"
		result.Error = err.Error()
		return result
	}
	result.Source = catalog.Provenance.Source
	result.ObservedAt = catalog.Provenance.ObservedAt
	result.BudgetDays = catalog.StalenessBudgetDays
	if result.BudgetDays <= 0 {
		result.BudgetDays = DefaultStalenessBudgetDays
	}
	observed, parseErr := parseObservedAt(catalog.Provenance.ObservedAt)
	if parseErr != nil {
		result.Status = "invalid"
		result.Error = parseErr.Error()
		return result
	}
	result.AgeDays = max(0, int(now.Sub(observed).Hours()/24))
	result.Status = "fresh"
	if result.AgeDays > result.BudgetDays {
		result.Status = "stale"
	}
	if result.AgeDays > result.BudgetDays*2 {
		result.Status = "hard_stale"
	}
	return result
}

// ValidateCatalogAgainstLive exposes the same comparison used by
// `policy validate --against-live` to non-CLI safeguards. It returns findings
// as data so callers can preserve the distinction between a measured drift and
// an unavailable measurement.
func ValidateCatalogAgainstLive(ctx context.Context, runner, path string) ([]PolicyValidationFinding, LiveModelCatalog, error) {
	catalog, _, err := loadCodingRoleCatalog(runner, path)
	if err != nil {
		return nil, LiveModelCatalog{}, err
	}
	live, err := DiscoverModels(ctx, runner)
	if err != nil {
		return nil, LiveModelCatalog{}, err
	}
	findings := append([]PolicyValidationFinding{}, catalogStalenessFindings(catalog, time.Now().UTC(), true)...)
	findings = append(findings, liveCatalogFindings(catalog, live)...)
	return findings, live, nil
}
