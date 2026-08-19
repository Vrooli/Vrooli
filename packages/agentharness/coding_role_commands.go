package agentharness

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/vrooli/cli-core/agentcatalog"
	"github.com/vrooli/cli-core/cliapp"
)

type codingRoleResponse struct {
	SchemaVersion  string                         `json:"schema_version"`
	Runner         string                         `json:"runner"`
	Role           string                         `json:"role"`
	Model          string                         `json:"model"`
	CanonicalModel string                         `json:"canonical_model,omitempty"`
	Fallbacks      []string                       `json:"fallbacks,omitempty"`
	Description    string                         `json:"description"`
	Capabilities   []string                       `json:"capabilities"`
	Provenance     agentcatalog.CatalogProvenance `json:"provenance"`
	Enforcement    EnforcementPosture             `json:"enforcement"`
	PolicyPath     string                         `json:"policy_path"`
	PolicyDigest   string                         `json:"policy_digest"`
	Billing        agentcatalog.Billing           `json:"billing"`
	Challenger     *agentcatalog.Challenger       `json:"challenger,omitempty"`
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
	catalog, data, err := agentcatalog.LoadCodingRoleCatalog(cfg.Runner, cfg.CatalogPath)
	if err != nil {
		return err
	}
	findings := append([]PolicyValidationFinding{}, agentcatalog.CatalogStalenessFindings(catalog, time.Now().UTC(), *againstLive)...)
	valid := true
	var live *LiveModelCatalog
	if *againstLive {
		discover := cfg.Discovery
		if discover == nil {
			discover = func(ctx context.Context) (LiveModelCatalog, error) {
				return agentcatalog.DiscoverModels(ctx, cfg.Runner)
			}
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
		findings = append(findings, agentcatalog.LiveCatalogFindings(catalog, result)...)
	}
	for _, finding := range findings {
		if finding.Severity == "error" {
			valid = false
		}
	}
	if agentcatalog.CatalogIsHardStale(catalog, time.Now().UTC()) && *againstLive {
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

func codingPolicyRoles(cfg CodingPolicyConfig, args []string) error {
	fs := policyFlagSet("policy roles", cfg.Stderr)
	jsonOut := fs.Bool("json", false, "Emit JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	catalog, data, err := agentcatalog.LoadCodingRoleCatalog(cfg.Runner, cfg.CatalogPath)
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
	catalog, data, err := agentcatalog.LoadCodingRoleCatalog(cfg.Runner, cfg.CatalogPath)
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

func responseFor(cfg CodingPolicyConfig, catalog CodingRoleCatalog, role string, data []byte) codingRoleResponse {
	r := catalog.Roles[role]
	billing := agentcatalog.Billing{Mode: "unknown", Source: "default"}
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
	var challenger *agentcatalog.Challenger
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
