package models

import (
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/vrooli/internal/cliout"
)

// PolicyHandlers owns the `policy` subcommand group. It emits the role/model
// contract ai-gateway's provider adapter consumes (`policy roles --json`,
// `policy resolve --role <id> --json`), sourced from model-policy.json. This
// mirrors the ollama/reranker provider policy contract so the gateway can treat
// pii-detector as a uniform provider.
type PolicyHandlers struct {
	GetEnv func(string) string
	Stdout io.Writer
	Stderr io.Writer
}

func DefaultPolicy() *PolicyHandlers {
	return &PolicyHandlers{GetEnv: os.Getenv, Stdout: os.Stdout, Stderr: os.Stderr}
}

func PolicyCommands(h *PolicyHandlers) cliapp.SubcommandGroup {
	if h == nil {
		h = DefaultPolicy()
	}
	return cliapp.SubcommandGroup{
		Name:        "policy",
		Description: "Emit the PII-detection role/model policy for the gateway",
		Subcommands: []cliapp.Command{
			{Name: "roles", Description: "List policy roles and their capabilities", Usage: "resource-pii-detector policy roles [--json]", Run: h.Roles},
			{Name: "resolve", Description: "Resolve a role to its concrete model", Usage: "resource-pii-detector policy resolve --role <id> [--json]", Run: h.Resolve},
		},
	}
}

type roleEntry struct {
	SchemaVersion        string   `json:"schema_version"`
	Role                 string   `json:"role"`
	RequiredCapabilities []string `json:"required_capabilities"`
	Capabilities         []string `json:"capabilities"`
}

func (h *PolicyHandlers) Roles(args []string) error {
	fs := flag.NewFlagSet("policy roles", flag.ContinueOnError)
	fs.SetOutput(h.Stderr)
	jsonOutput := fs.Bool("json", false, "Emit JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	policy, err := Load(policyPath(h.GetEnv))
	if err != nil {
		return err
	}
	names := make([]string, 0, len(policy.Roles))
	for name := range policy.Roles {
		names = append(names, name)
	}
	sort.Strings(names)
	entries := make([]roleEntry, 0, len(names))
	for _, name := range names {
		r := policy.Roles[name]
		caps := r.RequiredCapabilities
		if len(caps) == 0 {
			if m, ok := policy.Models[r.Model]; ok {
				caps = m.Capabilities
			}
		}
		entries = append(entries, roleEntry{
			SchemaVersion:        policy.SchemaVersion,
			Role:                 name,
			RequiredCapabilities: caps,
			Capabilities:         caps,
		})
	}
	payload := map[string]any{"roles": entries}
	if *jsonOutput {
		return cliout.NewEncoder(h.Stdout).Encode(payload)
	}
	for _, e := range entries {
		fmt.Fprintf(h.Stdout, "%s\t%s\n", e.Role, strings.Join(e.RequiredCapabilities, ","))
	}
	return nil
}

func (h *PolicyHandlers) Resolve(args []string) error {
	fs := flag.NewFlagSet("policy resolve", flag.ContinueOnError)
	fs.SetOutput(h.Stderr)
	role := fs.String("role", "", "Policy role to resolve")
	jsonOutput := fs.Bool("json", false, "Emit JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*role) == "" {
		return fmt.Errorf("--role is required")
	}
	policy, err := Load(policyPath(h.GetEnv))
	if err != nil {
		return err
	}
	r, ok := policy.Roles[*role]
	if !ok {
		return fmt.Errorf("unknown role %q", *role)
	}
	model := r.Model
	// Honor a durable active-model override when it is a member of the role.
	if active := readActiveModel(h.GetEnv); active != "" && policy.Allowed(*role, active) {
		model = active
	}
	m := policy.Models[model]
	ctxWindow := int32(m.ContextWindow)
	payload := map[string]any{
		"role":            *role,
		"model":           model,
		"canonical_model": model,
		"capabilities":    m.Capabilities,
		"context_window":  ctxWindow,
	}
	if *jsonOutput {
		return cliout.NewEncoder(h.Stdout).Encode(payload)
	}
	fmt.Fprintf(h.Stdout, "role: %s\nmodel: %s\n", *role, model)
	return nil
}
