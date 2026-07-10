package agentpolicy

// This file defines the transport contract shared by coding-agent resources.
// Catalog content stays beside its owning resource; this package has no model
// inventory and deliberately knows nothing about Agent Manager.

import (
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

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

const CodingRolePolicySchemaVersion = "v1"

type CodingRoleCatalog struct {
	SchemaVersion string                `json:"schema_version"`
	Runner        string                `json:"runner"`
	Roles         map[string]CodingRole `json:"roles"`
	Provenance    CatalogProvenance     `json:"provenance"`
}

type CodingRole struct {
	Model        string   `json:"model"`
	Fallbacks    []string `json:"fallbacks,omitempty"`
	Description  string   `json:"description"`
	Capabilities []string `json:"capabilities"`
}

type CatalogProvenance struct {
	Source     string `json:"source"`
	ObservedAt string `json:"observed_at"`
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
}

type codingRoleResponse struct {
	SchemaVersion string             `json:"schema_version"`
	Runner        string             `json:"runner"`
	Role          string             `json:"role"`
	Model         string             `json:"model"`
	Fallbacks     []string           `json:"fallbacks,omitempty"`
	Description   string             `json:"description"`
	Capabilities  []string           `json:"capabilities"`
	Provenance    CatalogProvenance  `json:"provenance"`
	Enforcement   EnforcementPosture `json:"enforcement"`
	PolicyPath    string             `json:"policy_path"`
	PolicyDigest  string             `json:"policy_digest"`
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
	if err := fs.Parse(args); err != nil {
		return err
	}
	catalog, data, err := loadCodingRoleCatalog(cfg)
	if err != nil {
		return err
	}
	if *jsonOut {
		return writeJSON(cfg.Stdout, map[string]any{"schema_version": CodingRolePolicySchemaVersion, "runner": catalog.Runner, "policy_path": cfg.CatalogPath, "policy_digest": digest(data), "valid": true})
	}
	fmt.Fprintf(cfg.Stdout, "valid coding role policy: runner=%s roles=%d path=%s\n", catalog.Runner, len(catalog.Roles), cfg.CatalogPath)
	return nil
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
		if !strings.HasPrefix(name, "code.") {
			errs = append(errs, fmt.Errorf("role %q must use the code.* namespace", name))
		}
		if strings.TrimSpace(role.Model) == "" {
			errs = append(errs, fmt.Errorf("role %q has an empty model", name))
		}
	}
	return errors.Join(errs...)
}

func responseFor(cfg CodingPolicyConfig, catalog CodingRoleCatalog, role string, data []byte) codingRoleResponse {
	r := catalog.Roles[role]
	return codingRoleResponse{SchemaVersion: CodingRolePolicySchemaVersion, Runner: catalog.Runner, Role: role, Model: r.Model, Fallbacks: append([]string(nil), r.Fallbacks...), Description: r.Description, Capabilities: append([]string(nil), r.Capabilities...), Provenance: catalog.Provenance, Enforcement: cfg.Posture, PolicyPath: cfg.CatalogPath, PolicyDigest: digest(data)}
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
