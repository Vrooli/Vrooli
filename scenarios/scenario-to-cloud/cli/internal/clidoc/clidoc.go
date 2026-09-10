// Package clidoc renders the operator command reference from the CLI
// manifest so help examples and governance metadata never drift from the
// declared surface. A test fails when the checked-in document is stale.
package clidoc

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// Manifest is the subset of cli/manifest.json the renderer reads.
type Manifest struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Groups      []Group   `json:"groups"`
	Omitted     []Omitted `json:"omitted"`
}

// Group is one command group (possibly nested).
type Group struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Flat        bool      `json:"flat"`
	Commands    []Command `json:"commands"`
	Groups      []Group   `json:"groups"`
}

// Command is one declared command.
type Command struct {
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Positionals []Positional `json:"positionals"`
	Flags       []Flag       `json:"flags"`
	Binding     Binding      `json:"binding"`
	Governance  Governance   `json:"governance"`
}

// Positional is one positional argument.
type Positional struct {
	Name        string `json:"name"`
	Required    bool   `json:"required"`
	Description string `json:"description"`
}

// Flag is one flag.
type Flag struct {
	Name        string `json:"name"`
	Bool        bool   `json:"bool"`
	Required    bool   `json:"required"`
	Default     string `json:"default"`
	Description string `json:"description"`
}

// Binding is the command binding.
type Binding struct {
	Kind    string `json:"kind"`
	Service string `json:"service"`
	Method  string `json:"method"`
}

// Governance is the command governance block.
type Governance struct {
	Effect               string `json:"effect"`
	RunEligible          bool   `json:"run_eligible"`
	RequiresConfirmation bool   `json:"requires_confirmation"`
}

// Walk visits every command with its slash-joined path (flat groups
// contribute no segment), the same key scopecatalog uses.
func (m *Manifest) Walk(visit func(path string, c Command)) {
	var walk func(g Group, parents []string)
	walk = func(g Group, parents []string) {
		path := parents
		if !g.Flat {
			path = append(append([]string(nil), parents...), g.Name)
		}
		for _, c := range g.Commands {
			visit(strings.Join(append(append([]string(nil), path...), c.Name), "/"), c)
		}
		for _, child := range g.Groups {
			walk(child, path)
		}
	}
	for _, g := range m.Groups {
		walk(g, nil)
	}
}

// Omitted is one intentionally unbound RPC.
type Omitted struct {
	Service string `json:"service"`
	Method  string `json:"method"`
	Reason  string `json:"reason"`
}

// Parse decodes the manifest.
func Parse(raw []byte) (*Manifest, error) {
	var m Manifest
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, fmt.Errorf("parse CLI manifest: %w", err)
	}
	return &m, nil
}

// Render produces the Markdown reference.
func Render(m *Manifest) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s CLI commands\n\n", m.Name)
	b.WriteString("<!-- Generated from cli/manifest.json by `STC_WRITE_CLI_DOC=1 go test ./cli/ -run TestCLIDocMatchesManifest`. Do not edit by hand. -->\n\n")
	b.WriteString(m.Description + "\n\n")
	b.WriteString("## Exit codes\n\n")
	b.WriteString("| Code | Meaning |\n|---|---|\n")
	b.WriteString("| 0 | Succeeded, or nothing to change (`no_op`) |\n")
	b.WriteString("| 1 | Failed: the operation ended `failed`, `failed_recovery` or `cancelled`, or the server answered an untyped error |\n")
	b.WriteString("| 2 | Refused before any effect: authority, ambiguous selector, stale or mismatched plan digest, request-key conflict, invalid flag grammar |\n")
	b.WriteString("| 3 | Pending: an operation is admitted but not terminal, or input is required (the durable handoff reference is printed) |\n")
	b.WriteString("| 124 | Observer timeout: the server-side wait bound elapsed; the operation is unchanged and the reattach command is printed |\n\n")
	b.WriteString("## Deployment selectors\n\n")
	b.WriteString("Every command that targets a deployment accepts exactly one selector: `--deployment <id>` (or the id as the positional argument), `--scenario <id> --environment <env>`, `--scenario <id> --domain <domain>`, or `--scenario <id> --host <host>`. The selector is resolved through `DeploymentsService.ResolveDeployment`; more than one match is refused with `deployment_selector_ambiguous` (exit 2) and the candidate ids are listed. Identities are printed the same way everywhere: `deployment: <id>  scenario: <id>  environment: <env>  target: <machine:id|host:host> (<transport>)`.\n\n")
	b.WriteString("Machine output (`--json`) is the server's typed message as proto JSON (snake_case field names, lossless); human output never has to be parsed.\n\n")
	b.WriteString("## Command groups\n\n")
	for _, g := range m.Groups {
		renderGroup(&b, m.Name, g, nil)
	}
	if len(m.Omitted) > 0 {
		b.WriteString("## RPCs without a direct command\n\n")
		b.WriteString("| Service | Method | Reason |\n|---|---|---|\n")
		for _, o := range m.Omitted {
			fmt.Fprintf(&b, "| %s | %s | %s |\n", o.Service, o.Method, o.Reason)
		}
		b.WriteString("\n")
	}
	return b.String()
}

func renderGroup(b *strings.Builder, cli string, g Group, parents []string) {
	path := parents
	if !g.Flat {
		path = append(append([]string(nil), parents...), g.Name)
	}
	heading := strings.Join(path, " ")
	if heading == "" {
		heading = g.Name + " (top level)"
	}
	fmt.Fprintf(b, "### %s\n\n", heading)
	if g.Description != "" {
		b.WriteString(g.Description + "\n\n")
	}
	if len(g.Commands) > 0 {
		b.WriteString("| Command | Effect | Agent-runnable | Binding | Description |\n|---|---|---|---|---|\n")
		for _, c := range g.Commands {
			binding := c.Binding.Kind
			if c.Binding.Kind == "connect-rpc" {
				binding = c.Binding.Service + "." + c.Binding.Method
			}
			fmt.Fprintf(b, "| `%s` | %s | %t | %s | %s |\n", strings.Join(append(append([]string(nil), path...), c.Name), " "), c.Governance.Effect, c.Governance.RunEligible, binding, escape(c.Description))
		}
		b.WriteString("\nExamples:\n\n```bash\n")
		for _, c := range g.Commands {
			b.WriteString(example(cli, path, c) + "\n")
		}
		b.WriteString("```\n\n")
	}
	for _, child := range g.Groups {
		renderGroup(b, cli, child, path)
	}
}

func example(cli string, path []string, c Command) string {
	parts := []string{cli}
	parts = append(parts, path...)
	parts = append(parts, c.Name)
	for _, p := range c.Positionals {
		if p.Required {
			parts = append(parts, "<"+p.Name+">")
		} else {
			parts = append(parts, "["+p.Name+"]")
		}
	}
	flags := append([]Flag(nil), c.Flags...)
	sort.SliceStable(flags, func(i, j int) bool { return flags[i].Required && !flags[j].Required })
	for _, f := range flags {
		switch {
		case f.Required && f.Bool:
			parts = append(parts, "--"+f.Name)
		case f.Required:
			parts = append(parts, "--"+f.Name+" <"+f.Name+">")
		case f.Bool:
			parts = append(parts, "[--"+f.Name+"]")
		default:
			parts = append(parts, "[--"+f.Name+" <value>]")
		}
	}
	return strings.Join(parts, " ")
}

func escape(s string) string {
	return strings.ReplaceAll(s, "|", "\\|")
}
