// Package policycmd implements the `resource-openrouter policy ...` command
// group: the programmatic authority for OpenRouter role -> concrete-model
// resolution. It mirrors resource-ollama's policy command surface so shared
// consumers (packages/ai-go/openrouter/policy) speak one process protocol.
package policycmd

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/vrooli/vrooli/resources/openrouter/cli/internal/policy"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/vrooli/internal/cliout"
)

type Handlers struct {
	GetEnv func(string) string
	Stdout io.Writer
	Stderr io.Writer
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

func Default() *Handlers {
	return &Handlers{GetEnv: os.Getenv, Stdout: os.Stdout, Stderr: os.Stderr}
}

func Commands(h *Handlers) cliapp.SubcommandGroup {
	if h == nil {
		h = Default()
	}
	return cliapp.SubcommandGroup{
		Name:        "policy",
		Description: "Inspect authoritative OpenRouter role and model policy metadata",
		Subcommands: []cliapp.Command{
			{
				Name:        "resolve",
				Description: "Resolve one role or catalog model to concrete policy metadata",
				Usage:       "resource-openrouter policy resolve (--role <role> | --model <model>) [--json] [--field <name>]",
				Run:         h.Resolve,
			},
			{
				Name:        "roles",
				Description: "List roles with resolved model metadata",
				Usage:       "resource-openrouter policy roles [--json]",
				Run:         h.Roles,
			},
			{
				Name:        "models",
				Description: "List catalog models and metadata",
				Usage:       "resource-openrouter policy models [--json]",
				Run:         h.Models,
			},
			{
				Name:        "constraints",
				Description: "Show policy constraints",
				Usage:       "resource-openrouter policy constraints [--json]",
				Run:         h.Constraints,
			},
		},
	}
}

func (h *Handlers) Resolve(args []string) error {
	fs := flag.NewFlagSet("policy resolve", flag.ContinueOnError)
	fs.SetOutput(h.Stderr)
	role := fs.String("role", "", "OpenRouter role to resolve, e.g. image.generate.logo")
	model := fs.String("model", "", "Catalog model reference to resolve, e.g. bytedance-seed/seedream-4.5")
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

	p, path, err := h.loadPolicy()
	if err != nil {
		return err
	}
	var resolved policy.ResolvedPolicyModel
	if strings.TrimSpace(*role) != "" {
		resolved, err = p.ResolveRole(*role)
	} else {
		resolved, err = p.ResolveModel(*model)
	}
	if err != nil {
		return err
	}
	report := resolveReport{ResolvedPolicyModel: resolved, PolicyPath: path}
	if name := strings.TrimSpace(*field); name != "" {
		return h.printField(report, name)
	}
	if *asJSON {
		return writeJSON(h.Stdout, report)
	}
	return h.printResolved(report)
}

func (h *Handlers) Roles(args []string) error {
	fs := flag.NewFlagSet("policy roles", flag.ContinueOnError)
	fs.SetOutput(h.Stderr)
	asJSON := fs.Bool("json", false, "Emit complete JSON metadata")
	if err := fs.Parse(args); err != nil {
		return err
	}
	p, path, err := h.loadPolicy()
	if err != nil {
		return err
	}
	report := rolesReport{SchemaVersion: p.SchemaVersion, PolicyPath: path, Roles: make([]resolveReport, 0, len(p.Roles))}
	for _, name := range p.RoleNames() {
		resolved, err := p.ResolveRole(name)
		if err != nil {
			return err
		}
		report.Roles = append(report.Roles, resolveReport{ResolvedPolicyModel: resolved, PolicyPath: path})
	}
	if *asJSON {
		return writeJSON(h.Stdout, report)
	}
	for _, role := range report.Roles {
		if _, err := fmt.Fprintf(h.Stdout, "%s -> %s [%s]\n", role.Role, role.Model, role.Endpoint); err != nil {
			return err
		}
	}
	return nil
}

func (h *Handlers) Models(args []string) error {
	fs := flag.NewFlagSet("policy models", flag.ContinueOnError)
	fs.SetOutput(h.Stderr)
	asJSON := fs.Bool("json", false, "Emit complete JSON metadata")
	if err := fs.Parse(args); err != nil {
		return err
	}
	p, path, err := h.loadPolicy()
	if err != nil {
		return err
	}
	report := modelsReport{SchemaVersion: p.SchemaVersion, PolicyPath: path, Models: make([]resolveReport, 0, len(p.Models))}
	for _, ref := range p.ModelRefs() {
		resolved, err := p.ResolveModel(ref)
		if err != nil {
			return err
		}
		report.Models = append(report.Models, resolveReport{ResolvedPolicyModel: resolved, PolicyPath: path})
	}
	if *asJSON {
		return writeJSON(h.Stdout, report)
	}
	for _, model := range report.Models {
		if _, err := fmt.Fprintf(h.Stdout, "%s [%s]\n", model.Model, strings.Join(model.Capabilities, ",")); err != nil {
			return err
		}
	}
	return nil
}

func (h *Handlers) Constraints(args []string) error {
	fs := flag.NewFlagSet("policy constraints", flag.ContinueOnError)
	fs.SetOutput(h.Stderr)
	asJSON := fs.Bool("json", false, "Emit complete JSON metadata")
	if err := fs.Parse(args); err != nil {
		return err
	}
	p, path, err := h.loadPolicy()
	if err != nil {
		return err
	}
	report := constraintsReport{SchemaVersion: p.SchemaVersion, PolicyPath: path, Constraints: p.Constraints}
	if *asJSON {
		return writeJSON(h.Stdout, report)
	}
	_, err = fmt.Fprintf(h.Stdout, "schema_version: %s\npolicy_path: %s\nendpoints: %s\nroles: %s\n",
		report.SchemaVersion, report.PolicyPath,
		strings.Join(report.Constraints.Endpoints, ","),
		strings.Join(report.Constraints.RolePreferenceOrder, ","))
	return err
}

func (h *Handlers) loadPolicy() (policy.Policy, string, error) {
	getenv := h.GetEnv
	if getenv == nil {
		getenv = os.Getenv
	}
	return policy.LoadDefaultFile(getenv)
}

func (h *Handlers) printResolved(report resolveReport) error {
	if report.Role != "" {
		_, err := fmt.Fprintf(h.Stdout, "%s -> %s [%s]\n", report.Role, report.Model, report.Endpoint)
		return err
	}
	_, err := fmt.Fprintln(h.Stdout, report.Model)
	return err
}

func (h *Handlers) printField(report resolveReport, name string) error {
	value, err := fieldValue(report, name)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(h.Stdout, value)
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

func writeJSON(w io.Writer, v any) error {
	return cliout.NewEncoder(w).Encode(v)
}
