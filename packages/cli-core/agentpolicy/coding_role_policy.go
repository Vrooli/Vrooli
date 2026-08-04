package agentpolicy

// This file defines the transport contract shared by coding-agent resources.
// Catalog content stays beside its owning resource; this package has no model
// inventory and deliberately knows nothing about Agent Manager.

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

const CodingRolePolicySchemaVersion = "v1"

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

type EnforcementPosture struct {
	Permissions string   `json:"permissions"`
	Caveats     []string `json:"caveats,omitempty"`
}

type CodingPolicyConfig struct {
	Runner      string
	CatalogPath string
	Posture     EnforcementPosture
	Stdout      io.Writer
	Stderr      io.Writer
	Discovery   ModelDiscoveryFunc
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

type codingRoleResponse struct {
	SchemaVersion  string             `json:"schema_version"`
	Runner         string             `json:"runner"`
	Role           string             `json:"role"`
	Model          string             `json:"model"`
	CanonicalModel string             `json:"canonical_model,omitempty"`
	Fallbacks      []string           `json:"fallbacks,omitempty"`
	Description    string             `json:"description"`
	Capabilities   []string           `json:"capabilities"`
	Provenance     CatalogProvenance  `json:"provenance"`
	Enforcement    EnforcementPosture `json:"enforcement"`
	PolicyPath     string             `json:"policy_path"`
	PolicyDigest   string             `json:"policy_digest"`
	Billing        Billing            `json:"billing"`
	Challenger     *Challenger        `json:"challenger,omitempty"`
}

// CodingPolicyCommands supplies a uniform, machine-readable protocol while
// each resource supplies its catalog path, runner identity, and enforcement.
func CodingPolicyCommands(cfg CodingPolicyConfig) cliapp.SubcommandGroup {
	if cfg.Stdout == nil {
		cfg.Stdout = os.Stdout
	}
	if cfg.Stderr == nil {
		cfg.Stderr = os.Stderr
	}
	return cliapp.SubcommandGroup{Name: "policy", Description: "Inspect this resource's coding-role policy", Subcommands: []cliapp.Command{
		{Name: "validate", Description: "Validate the resource-owned policy catalog", Run: func(args []string) error { return codingPolicyValidate(cfg, args) }},
		{Name: "roles", Description: "List semantic coding roles", Run: func(args []string) error { return codingPolicyRoles(cfg, args) }},
		{Name: "resolve", Description: "Resolve one semantic coding role", Run: func(args []string) error { return codingPolicyResolve(cfg, args) }},
	}}
}

func codingPolicyValidate(cfg CodingPolicyConfig, args []string) error {
	fs := policyFlagSet("policy validate", cfg.Stderr)
	jsonOut := fs.Bool("json", false, "Emit JSON")
	againstLive := fs.Bool("against-live", false, "Compare the catalog with the runner's live model catalog")
	if err := fs.Parse(args); err != nil {
		return err
	}
	catalog, data, err := loadCodingRoleCatalog(cfg)
	if err != nil {
		return err
	}
	findings := append([]PolicyValidationFinding{}, catalogStalenessFindings(catalog, time.Now().UTC(), *againstLive)...)
	valid := true
	var live *LiveModelCatalog
	if *againstLive {
		discover := cfg.Discovery
		if discover == nil {
			discover = func(ctx context.Context) (LiveModelCatalog, error) { return DiscoverModels(ctx, cfg.Runner) }
		}
		result, discoverErr := discover(context.Background())
		if discoverErr != nil {
			payload := map[string]any{"schema_version": CodingRolePolicySchemaVersion, "runner": catalog.Runner, "policy_path": cfg.CatalogPath, "policy_digest": digest(data), "valid": false, "discovery_status": "not_measured", "findings": findings, "error": discoverErr.Error()}
			if *jsonOut {
				_ = writeJSON(cfg.Stdout, payload)
			}
			return &PolicyValidationError{Code: "discovery_unavailable", Err: fmt.Errorf("%w: %v", ErrModelDiscoveryUnavailable, discoverErr)}
		}
		live = &result
		findings = append(findings, liveCatalogFindings(catalog, result)...)
	}
	for _, finding := range findings {
		if finding.Severity == "error" {
			valid = false
		}
	}
	if catalogIsHardStale(catalog, time.Now().UTC()) && *againstLive {
		valid = false
	}
	if *jsonOut {
		payload := map[string]any{"schema_version": CodingRolePolicySchemaVersion, "runner": catalog.Runner, "policy_path": cfg.CatalogPath, "policy_digest": digest(data), "valid": valid, "findings": findings}
		if live != nil {
			payload["discovery_status"] = "measured"
			payload["live_catalog"] = live
		}
		if err := writeJSON(cfg.Stdout, payload); err != nil {
			return err
		}
	} else if *againstLive {
		fmt.Fprintf(cfg.Stdout, "coding role policy: runner=%s roles=%d findings=%d valid=%t\n", catalog.Runner, len(catalog.Roles), len(findings), valid)
		for _, finding := range findings {
			fmt.Fprintf(cfg.Stdout, "%s %s %s: %s\n", finding.Severity, finding.Type, finding.Role, finding.Message)
		}
	} else if len(findings) > 0 {
		for _, finding := range findings {
			fmt.Fprintf(cfg.Stdout, "%s %s: %s\n", finding.Severity, finding.Type, finding.Message)
		}
	}
	if !valid {
		return &PolicyValidationError{Code: "policy_invalid", Err: errors.New("catalog has blocking validation findings")}
	}
	if !*againstLive {
		fmt.Fprintf(cfg.Stdout, "valid coding role policy: runner=%s roles=%d path=%s\n", catalog.Runner, len(catalog.Roles), cfg.CatalogPath)
	}
	return nil
}

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

func codingPolicyRoles(cfg CodingPolicyConfig, args []string) error {
	fs := policyFlagSet("policy roles", cfg.Stderr)
	jsonOut := fs.Bool("json", false, "Emit JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	catalog, data, err := loadCodingRoleCatalog(cfg)
	if err != nil {
		return err
	}
	roles := sortedRoles(catalog)
	responses := make([]codingRoleResponse, 0, len(roles))
	for _, role := range roles {
		responses = append(responses, responseFor(cfg, catalog, role, data))
	}
	if *jsonOut {
		return writeJSON(cfg.Stdout, map[string]any{"schema_version": CodingRolePolicySchemaVersion, "runner": catalog.Runner, "roles": responses})
	}
	for _, response := range responses {
		fmt.Fprintf(cfg.Stdout, "%s -> %s\n", response.Role, response.Model)
	}
	return nil
}

func codingPolicyResolve(cfg CodingPolicyConfig, args []string) error {
	fs := policyFlagSet("policy resolve", cfg.Stderr)
	role := fs.String("role", "", "Semantic role to resolve, e.g. code.default")
	jsonOut := fs.Bool("json", false, "Emit JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*role) == "" {
		return errors.New("--role is required")
	}
	catalog, data, err := loadCodingRoleCatalog(cfg)
	if err != nil {
		return err
	}
	if _, ok := catalog.Roles[*role]; !ok {
		return fmt.Errorf("unknown coding role %q", *role)
	}
	response := responseFor(cfg, catalog, *role, data)
	if *jsonOut {
		return writeJSON(cfg.Stdout, response)
	}
	fmt.Fprintf(cfg.Stdout, "%s -> %s\n", response.Role, response.Model)
	return nil
}

func loadCodingRoleCatalog(cfg CodingPolicyConfig) (CodingRoleCatalog, []byte, error) {
	data, err := os.ReadFile(cfg.CatalogPath)
	if err != nil {
		return CodingRoleCatalog{}, nil, fmt.Errorf("read coding role policy %s: %w", cfg.CatalogPath, err)
	}
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	var catalog CodingRoleCatalog
	if err := decoder.Decode(&catalog); err != nil {
		return CodingRoleCatalog{}, nil, fmt.Errorf("parse coding role policy %s: %w", cfg.CatalogPath, err)
	}
	if err := validateCodingRoleCatalog(catalog, cfg.Runner); err != nil {
		return CodingRoleCatalog{}, nil, fmt.Errorf("invalid coding role policy %s: %w", cfg.CatalogPath, err)
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

func responseFor(cfg CodingPolicyConfig, catalog CodingRoleCatalog, role string, data []byte) codingRoleResponse {
	r := catalog.Roles[role]
	billing := Billing{Mode: "unknown", Source: "default"}
	if catalog.Billing != nil {
		billing = *catalog.Billing
	}
	envPrefix := "VROOLI_" + strings.ToUpper(strings.ReplaceAll(catalog.Runner, "-", "_")) + "_BILLING_"
	if value := strings.TrimSpace(os.Getenv(envPrefix + "MODE")); value != "" {
		billing.Mode = value
		billing.Source = "environment"
	}
	if value := strings.TrimSpace(os.Getenv(envPrefix + "PROVIDER")); value != "" {
		billing.Provider = value
	}
	if value := strings.TrimSpace(os.Getenv(envPrefix + "ACCOUNT_REF")); value != "" {
		billing.AccountRef = value
	}
	if value := strings.TrimSpace(os.Getenv(envPrefix + "PLAN_REF")); value != "" {
		billing.PlanRef = value
	}
	if value := strings.TrimSpace(os.Getenv(envPrefix + "PLAN_LABEL")); value != "" {
		billing.PlanLabel = value
	}
	if value := strings.TrimSpace(os.Getenv(envPrefix + "QUOTA_WINDOW")); value != "" {
		billing.QuotaWindow = value
	}
	var challenger *Challenger
	if r.Challenger != nil {
		copy := *r.Challenger
		challenger = &copy
	}
	return codingRoleResponse{SchemaVersion: CodingRolePolicySchemaVersion, Runner: catalog.Runner, Role: role, Model: r.Model, CanonicalModel: r.CanonicalModel, Fallbacks: append([]string(nil), r.Fallbacks...), Description: r.Description, Capabilities: append([]string(nil), r.Capabilities...), Provenance: catalog.Provenance, Enforcement: cfg.Posture, PolicyPath: cfg.CatalogPath, PolicyDigest: digest(data), Billing: billing, Challenger: challenger}
}

func sortedRoles(c CodingRoleCatalog) []string {
	out := make([]string, 0, len(c.Roles))
	for name := range c.Roles {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

func policyFlagSet(name string, stderr io.Writer) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(stderr)
	return fs
}

func writeJSON(w io.Writer, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(w, string(data))
	return err
}
func digest(data []byte) string { sum := sha256.Sum256(data); return hex.EncodeToString(sum[:]) }

// ResourceCatalogPath locates an installed resource catalog through the normal
// repo-root resolver. Tests may pass an explicit path directly in the config.
func ResourceCatalogPath(resource string) string {
	return filepath.Join(cliutil.ResolveRepoRoot(), "resources", resource, "model-policy.json")
}

// ReadCodingRoleCatalog returns a validated resource-owned catalog for
// control-plane readers that need provenance or model resolution. Callers
// still receive the full resource contract; no Agent Manager vocabulary is
// introduced here.
func ReadCodingRoleCatalog(runner, path string) (CodingRoleCatalog, error) {
	catalog, _, err := loadCodingRoleCatalog(CodingPolicyConfig{Runner: runner, CatalogPath: path})
	return catalog, err
}

// ReadCatalogFreshness reads only catalog metadata for a health/status view.
// It intentionally uses the same strict loader as policy validation so a
// malformed catalog cannot appear healthy merely because its file exists.
func ReadCatalogFreshness(runner, path string, now time.Time) CatalogFreshness {
	result := CatalogFreshness{Runner: runner, PolicyPath: path, Status: "unknown"}
	catalog, _, err := loadCodingRoleCatalog(CodingPolicyConfig{Runner: runner, CatalogPath: path})
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
	catalog, _, err := loadCodingRoleCatalog(CodingPolicyConfig{Runner: runner, CatalogPath: path})
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
