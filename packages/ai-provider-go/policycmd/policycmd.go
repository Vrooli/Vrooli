// Package policycmd implements the generic `policy ...` command behavior shared
// by AI provider resources. Resource CLIs wrap these handlers in their own
// cliapp command group and supply the provider-specific catalog loader.
package policycmd

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/vrooli/ai-provider-go/policy"
)

// Handlers owns the loader and output sinks for one provider.
type Handlers struct {
	ResourceName string
	Load         func() (policy.Catalog, string, error)
	Stdout       io.Writer
	Stderr       io.Writer
}

// New returns handlers wired to the process environment.
func New(resourceName string, load func() (policy.Catalog, string, error)) *Handlers {
	return &Handlers{ResourceName: resourceName, Load: load, Stdout: os.Stdout, Stderr: os.Stderr}
}

type resolveReport struct {
	policy.ResolvedPolicyModel
	PolicyPath string `json:"policy_path"`
}

type rolesReport struct {
	SchemaVersion string          `json:"schema_version"`
	PolicyPath    string          `json:"policy_path"`
	Roles         []resolveReport `json:"roles"`
}

type modelsReport struct {
	SchemaVersion string          `json:"schema_version"`
	PolicyPath    string          `json:"policy_path"`
	Models        []resolveReport `json:"models"`
}

type constraintsReport struct {
	SchemaVersion string             `json:"schema_version"`
	PolicyPath    string             `json:"policy_path"`
	Constraints   policy.Constraints `json:"constraints"`
}

// Resolve implements `policy resolve`.
func (h *Handlers) Resolve(args []string) error {
	fs := flag.NewFlagSet("policy resolve", flag.ContinueOnError)
	fs.SetOutput(h.stderr())
	role := fs.String("role", "", "Role to resolve")
	model := fs.String("model", "", "Catalog model reference to resolve")
	asJSON := fs.Bool("json", false, "Emit complete JSON metadata")
	field := fs.String("field", "", "Emit one scalar field for scripts")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*role) == "" && strings.TrimSpace(*model) == "" {
		return errors.New("--role or --model is required")
	}
	if strings.TrimSpace(*role) != "" && strings.TrimSpace(*model) != "" {
		return errors.New("--role and --model are mutually exclusive")
	}
	catalog, path, err := h.load()
	if err != nil {
		return err
	}
	var resolved policy.ResolvedPolicyModel
	if strings.TrimSpace(*role) != "" {
		resolved, err = policy.ResolveRole(catalog, *role)
	} else {
		resolved, err = policy.ResolveModel(catalog, *model)
	}
	if err != nil {
		return err
	}
	report := resolveReport{ResolvedPolicyModel: resolved, PolicyPath: path}
	if name := strings.TrimSpace(*field); name != "" {
		return h.printField(report, name)
	}
	if *asJSON {
		return writeJSON(h.stdout(), report)
	}
	return h.printResolved(report)
}

// Roles implements `policy roles`.
func (h *Handlers) Roles(args []string) error {
	fs := flag.NewFlagSet("policy roles", flag.ContinueOnError)
	fs.SetOutput(h.stderr())
	asJSON := fs.Bool("json", false, "Emit complete JSON metadata")
	if err := fs.Parse(args); err != nil {
		return err
	}
	catalog, path, err := h.load()
	if err != nil {
		return err
	}
	report := rolesReport{SchemaVersion: catalog.SchemaVersion, PolicyPath: path, Roles: make([]resolveReport, 0, len(catalog.Roles))}
	for _, name := range catalog.RoleNames() {
		resolved, err := policy.ResolveRole(catalog, name)
		if err != nil {
			return err
		}
		report.Roles = append(report.Roles, resolveReport{ResolvedPolicyModel: resolved, PolicyPath: path})
	}
	if *asJSON {
		return writeJSON(h.stdout(), report)
	}
	for _, role := range report.Roles {
		if _, err := fmt.Fprintf(h.stdout(), "%s -> %s [%s]\n", role.Role, role.Model, role.Endpoint); err != nil {
			return err
		}
	}
	return nil
}

// Models implements `policy models`.
func (h *Handlers) Models(args []string) error {
	fs := flag.NewFlagSet("policy models", flag.ContinueOnError)
	fs.SetOutput(h.stderr())
	asJSON := fs.Bool("json", false, "Emit complete JSON metadata")
	if err := fs.Parse(args); err != nil {
		return err
	}
	catalog, path, err := h.load()
	if err != nil {
		return err
	}
	report := modelsReport{SchemaVersion: catalog.SchemaVersion, PolicyPath: path, Models: make([]resolveReport, 0, len(catalog.Models))}
	for _, ref := range catalog.ModelRefs() {
		resolved, err := policy.ResolveModel(catalog, ref)
		if err != nil {
			return err
		}
		report.Models = append(report.Models, resolveReport{ResolvedPolicyModel: resolved, PolicyPath: path})
	}
	if *asJSON {
		return writeJSON(h.stdout(), report)
	}
	for _, model := range report.Models {
		if _, err := fmt.Fprintf(h.stdout(), "%s [%s]\n", model.Model, strings.Join(model.Capabilities, ",")); err != nil {
			return err
		}
	}
	return nil
}

// Constraints implements `policy constraints`.
func (h *Handlers) Constraints(args []string) error {
	fs := flag.NewFlagSet("policy constraints", flag.ContinueOnError)
	fs.SetOutput(h.stderr())
	asJSON := fs.Bool("json", false, "Emit complete JSON metadata")
	if err := fs.Parse(args); err != nil {
		return err
	}
	catalog, path, err := h.load()
	if err != nil {
		return err
	}
	report := constraintsReport{SchemaVersion: catalog.SchemaVersion, PolicyPath: path, Constraints: catalog.Constraints}
	if *asJSON {
		return writeJSON(h.stdout(), report)
	}
	_, err = fmt.Fprintf(h.stdout(), "schema_version: %s\npolicy_path: %s\nendpoints: %s\nroles: %s\n",
		report.SchemaVersion, report.PolicyPath,
		strings.Join(report.Constraints.Endpoints, ","),
		strings.Join(report.Constraints.RolePreferenceOrder, ","))
	return err
}

func (h *Handlers) load() (policy.Catalog, string, error) {
	if h.Load == nil {
		return policy.Catalog{}, "", errors.New("policy loader is not configured")
	}
	return h.Load()
}

func (h *Handlers) stdout() io.Writer {
	if h.Stdout != nil {
		return h.Stdout
	}
	return os.Stdout
}

func (h *Handlers) stderr() io.Writer {
	if h.Stderr != nil {
		return h.Stderr
	}
	return os.Stderr
}

func (h *Handlers) printResolved(report resolveReport) error {
	if report.Role != "" {
		_, err := fmt.Fprintf(h.stdout(), "%s -> %s [%s]\n", report.Role, report.Model, report.Endpoint)
		return err
	}
	_, err := fmt.Fprintln(h.stdout(), report.Model)
	return err
}

func (h *Handlers) printField(report resolveReport, name string) error {
	value, err := fieldValue(report, name)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(h.stdout(), value)
	return err
}

func fieldValue(report resolveReport, name string) (string, error) {
	switch name {
	case "schema_version":
		return report.SchemaVersion, nil
	case "policy_path":
		return report.PolicyPath, nil
	case "role":
		return report.Role, nil
	case "source":
		return report.Source, nil
	case "model":
		return report.Model, nil
	case "endpoint":
		return report.Endpoint, nil
	case "provider":
		return report.Provider, nil
	case "family":
		return report.Family, nil
	case "context_window_tokens":
		return strconv.Itoa(report.ContextWindowTokens), nil
	case "embedding_dimensions":
		return strconv.Itoa(report.EmbeddingDimensions), nil
	case "default_eligible":
		return strconv.FormatBool(report.DefaultEligible), nil
	case "capabilities":
		return strings.Join(report.Capabilities, ","), nil
	case "fallbacks":
		return strings.Join(report.Fallbacks, ","), nil
	default:
		return "", fmt.Errorf("unknown policy field %q", name)
	}
}

func writeJSON(w io.Writer, value any) error {
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	return encoder.Encode(value)
}
