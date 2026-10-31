package commands

import (
	"context"
	"flag"
	"fmt"
	"strings"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/vrooli/resources/ollaya/cli/internal/policy"
	"github.com/vrooli/vrooli/resources/ollaya/cli/internal/upstream"
)

func PolicyCommands(h *Handlers) cliapp.SubcommandGroup {
	if h == nil {
		h = Default()
	}
	return cliapp.SubcommandGroup{
		Name:        "policy",
		Description: "Emit the decision role policy ai-gateway routes on",
		Subcommands: []cliapp.Command{
			{Name: "roles", Description: "List decision roles with locality and configuration status", Usage: "resource-ollaya policy roles [--json]", Run: h.Roles},
			{Name: "resolve", Description: "Resolve a role to its upstream and model; fails typed when it cannot serve", Usage: "resource-ollaya policy resolve --role <id> [--json]", Run: h.Resolve},
		},
	}
}

type roleEntry struct {
	SchemaVersion        string   `json:"schema_version"`
	Role                 string   `json:"role"`
	RequiredCapabilities []string `json:"required_capabilities"`
	Capabilities         []string `json:"capabilities"`
	Modalities           struct {
		Input []string `json:"input"`
	} `json:"modalities"`
	Locality string `json:"locality"`
	Upstream string `json:"upstream"`
	Model    string `json:"model"`
	// Status is "available" or "unconfigured" (credentials absent). Roles
	// reports configuration only; resolve also proves the local model is pulled.
	Status             string   `json:"status"`
	MissingCredentials []string `json:"missing_credentials,omitempty"`
	Description        string   `json:"description"`
}

func (h *Handlers) Roles(args []string) error {
	fs := flag.NewFlagSet("policy roles", flag.ContinueOnError)
	fs.SetOutput(h.Stderr)
	jsonOutput := fs.Bool("json", false, "Emit JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	p, err := h.policy()
	if err != nil {
		return err
	}
	entries := make([]roleEntry, 0, len(p.Roles))
	for _, name := range p.RoleNames() {
		resolution, _ := p.Resolve(name, h.GetEnv)
		entry := roleEntry{
			SchemaVersion: p.SchemaVersion, Role: name,
			RequiredCapabilities: []string{policy.Capability}, Capabilities: []string{policy.Capability},
			Locality: resolution.Locality, Upstream: resolution.Upstream, Model: resolution.Model,
			Status: "available", MissingCredentials: resolution.Missing, Description: p.Roles[name].Description,
		}
		entry.Modalities.Input = []string{"text"}
		if !resolution.Configured() {
			entry.Status = "unconfigured"
		}
		entries = append(entries, entry)
	}
	if *jsonOutput {
		return writeJSON(h.Stdout, map[string]any{"roles": entries})
	}
	for _, entry := range entries {
		fmt.Fprintf(h.Stdout, "%-24s %-7s %-12s %-22s %s\n", entry.Role, entry.Locality, entry.Status, entry.Model, strings.Join(entry.MissingCredentials, ","))
	}
	return nil
}

func (h *Handlers) Resolve(args []string) error {
	fs := flag.NewFlagSet("policy resolve", flag.ContinueOnError)
	fs.SetOutput(h.Stderr)
	role := fs.String("role", "", "Decision role to resolve")
	jsonOutput := fs.Bool("json", false, "Emit JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*role) == "" {
		return fmt.Errorf("--role is required")
	}
	p, err := h.policy()
	if err != nil {
		return err
	}
	resolution, err := p.Resolve(*role, h.GetEnv)
	if err != nil {
		return err
	}
	if !resolution.Configured() {
		return h.fail(&upstream.ProviderError{Code: upstream.CodeUnconfigured, Message: fmt.Sprintf("role %s needs %s", resolution.Role, strings.Join(resolution.Missing, ", "))})
	}
	if resolution.Locality == policy.LocalityLocal {
		if err := h.localModelReady(context.Background(), p, resolution.Model); err != nil {
			return h.fail(err)
		}
	}
	payload := map[string]any{
		"role": resolution.Role, "model": resolution.Model, "canonical_model": resolution.Model,
		"capabilities": []string{policy.Capability}, "locality": resolution.Locality, "upstream": resolution.Upstream,
	}
	if *jsonOutput {
		return writeJSON(h.Stdout, payload)
	}
	fmt.Fprintf(h.Stdout, "role: %s\nupstream: %s (%s)\nmodel: %s\n", resolution.Role, resolution.Upstream, resolution.Locality, resolution.Model)
	return nil
}
