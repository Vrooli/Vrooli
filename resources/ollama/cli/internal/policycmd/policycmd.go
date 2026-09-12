// Package policycmd implements the `resource-ollama policy ...` command group.
package policycmd

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/resources/ollama/cli/internal/policy"
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

type embeddingMetadata struct {
	Role                string `json:"role"`
	Model               string `json:"model"`
	Dimensions          int    `json:"dimensions"`
	PolicySchemaVersion string `json:"policy_schema_version,omitempty"`
}

type retargetPlanReport struct {
	Role           string            `json:"role"`
	Old            embeddingMetadata `json:"old"`
	New            embeddingMetadata `json:"new"`
	AffectedStores []string          `json:"affected_stores,omitempty"`
	Compatibility  string            `json:"compatibility"`
	RequiredAction string            `json:"required_action"`
	ApplySafety    string            `json:"apply_safety_status"`
	PolicyPath     string            `json:"policy_path"`
}

func Default() *Handlers {
	return &Handlers{
		GetEnv: os.Getenv,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
}

func Commands(h *Handlers) cliapp.SubcommandGroup {
	if h == nil {
		h = Default()
	}
	return cliapp.SubcommandGroup{
		Name:        "policy",
		Description: "Inspect authoritative Ollama role and model policy metadata",
		Subcommands: []cliapp.Command{
			{
				Name:        "resolve",
				Description: "Resolve one role or catalog model to concrete policy metadata",
				Usage:       "resource-ollama policy resolve (--role <role> | --model <model>) [--json] [--field <name>]",
				Run:         h.Resolve,
			},
			{
				Name:        "roles",
				Description: "List roles with resolved model metadata",
				Usage:       "resource-ollama policy roles [--json]",
				Run:         h.Roles,
			},
			{
				Name:        "models",
				Description: "List catalog models and metadata",
				Usage:       "resource-ollama policy models [--json]",
				Run:         h.Models,
			},
			{
				Name:        "constraints",
				Description: "Show policy constraints",
				Usage:       "resource-ollama policy constraints [--json]",
				Run:         h.Constraints,
			},
			{
				Name:        "retarget-plan",
				Description: "Dry-run an embedding role retarget from old metadata to the current policy",
				Usage:       "resource-ollama policy retarget-plan --role <role> --old-model <model> --old-dimensions <n> [--old-schema-version <version>] [--store <id>]... [--json]",
				Run:         h.RetargetPlan,
			},
		},
	}
}

func (h *Handlers) Resolve(args []string) error {
	fs := flag.NewFlagSet("policy resolve", flag.ContinueOnError)
	fs.SetOutput(h.Stderr)
	role := fs.String("role", "", "Ollama role to resolve, e.g. embedding.default")
	model := fs.String("model", "", "Catalog model reference to resolve, e.g. nomic-embed-text:latest")
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
	report := rolesReport{
		SchemaVersion: p.SchemaVersion,
		PolicyPath:    path,
		Roles:         make([]resolveReport, 0, len(p.Roles)),
	}
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
		if _, err := fmt.Fprintf(h.Stdout, "%s -> %s\n", role.Role, role.Model); err != nil {
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
	report := modelsReport{
		SchemaVersion: p.SchemaVersion,
		PolicyPath:    path,
		Models:        make([]resolveReport, 0, len(p.Models)),
	}
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
	_, err = fmt.Fprintf(h.Stdout, "schema_version: %s\npolicy_path: %s\n", report.SchemaVersion, report.PolicyPath)
	return err
}

func (h *Handlers) RetargetPlan(args []string) error {
	fs := flag.NewFlagSet("policy retarget-plan", flag.ContinueOnError)
	fs.SetOutput(h.Stderr)
	role := fs.String("role", "", "Embedding role to compare, e.g. embedding.default")
	oldModel := fs.String("old-model", "", "Previously resolved embedding model")
	oldDimensions := fs.Int("old-dimensions", 0, "Previously resolved embedding dimensions")
	oldSchema := fs.String("old-schema-version", "", "Previously resolved policy schema version")
	asJSON := fs.Bool("json", false, "Emit JSON plan")
	var stores repeatStrings
	fs.Var(&stores, "store", "Affected store identifier; may be repeated")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*role) == "" {
		return errors.New("--role is required")
	}
	if strings.TrimSpace(*oldModel) == "" {
		return errors.New("--old-model is required")
	}
	if *oldDimensions <= 0 {
		return errors.New("--old-dimensions must be positive")
	}
	p, path, err := h.loadPolicy()
	if err != nil {
		return err
	}
	resolved, err := p.ResolveRole(*role)
	if err != nil {
		return err
	}
	if resolved.EmbeddingDimensions <= 0 {
		return fmt.Errorf("role %q resolved model %q without embedding_dimensions", *role, resolved.Model)
	}
	report := buildRetargetPlanReport(
		embeddingMetadata{
			Role:                strings.TrimSpace(*role),
			Model:               strings.TrimSpace(*oldModel),
			Dimensions:          *oldDimensions,
			PolicySchemaVersion: strings.TrimSpace(*oldSchema),
		},
		embeddingMetadata{
			Role:                strings.TrimSpace(*role),
			Model:               resolved.Model,
			Dimensions:          resolved.EmbeddingDimensions,
			PolicySchemaVersion: resolved.SchemaVersion,
		},
		stores,
		path,
	)
	if *asJSON {
		return writeJSON(h.Stdout, report)
	}
	_, err = fmt.Fprintf(h.Stdout, "role: %s\ncompatibility: %s\nrequired_action: %s\napply_safety_status: %s\n", report.Role, report.Compatibility, report.RequiredAction, report.ApplySafety)
	return err
}

func buildRetargetPlanReport(oldMeta, newMeta embeddingMetadata, stores []string, policyPath string) retargetPlanReport {
	oldMeta = normalizeEmbeddingMetadata(oldMeta)
	newMeta = normalizeEmbeddingMetadata(newMeta)
	report := retargetPlanReport{
		Role:           newMeta.Role,
		Old:            oldMeta,
		New:            newMeta,
		AffectedStores: append([]string{}, stores...),
		ApplySafety:    "dry-run only; no destructive apply is implemented",
		PolicyPath:     policyPath,
	}
	switch {
	case oldMeta.Role == newMeta.Role && oldMeta.Model == newMeta.Model && oldMeta.Dimensions == newMeta.Dimensions && oldMeta.PolicySchemaVersion == newMeta.PolicySchemaVersion:
		report.Compatibility = "compatible_noop"
		report.RequiredAction = "none"
	case oldMeta.Dimensions == newMeta.Dimensions:
		report.Compatibility = "compatible_reembed"
		report.RequiredAction = "reembed affected stores before serving mixed vector spaces"
	default:
		report.Compatibility = "incompatible_shape"
		report.RequiredAction = "create shadow storage with the new dimensions, reembed, validate, then cut over"
	}
	return report
}

func normalizeEmbeddingMetadata(meta embeddingMetadata) embeddingMetadata {
	meta.Role = strings.TrimSpace(meta.Role)
	meta.Model = strings.TrimSpace(meta.Model)
	meta.PolicySchemaVersion = strings.TrimSpace(meta.PolicySchemaVersion)
	return meta
}

type repeatStrings []string

func (r *repeatStrings) String() string { return strings.Join(*r, ",") }

func (r *repeatStrings) Set(value string) error {
	value = strings.TrimSpace(value)
	if value != "" {
		*r = append(*r, value)
	}
	return nil
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
		_, err := fmt.Fprintf(h.Stdout, "%s -> %s\n", report.Role, report.Model)
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
	case "embedding_dimensions":
		return strconv.Itoa(report.EmbeddingDimensions), nil
	case "context_window_tokens":
		return strconv.Itoa(report.ContextWindowTokens), nil
	case "disk_size_gb_estimate":
		return strconv.FormatFloat(report.DiskSizeGBEstimate, 'f', -1, 64), nil
	case "ram_gb_estimate":
		return strconv.FormatFloat(report.RAMGBEstimate, 'f', -1, 64), nil
	case "vram_gb_estimate":
		return strconv.FormatFloat(report.VRAMGBEstimate, 'f', -1, 64), nil
	case "default_eligible":
		return strconv.FormatBool(report.DefaultEligible), nil
	default:
		return "", fmt.Errorf("unknown policy field %q", name)
	}
}

func writeJSON(w io.Writer, v any) error {
	return cliout.NewEncoder(w).Encode(v)
}
