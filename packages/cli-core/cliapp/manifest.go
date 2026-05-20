package cliapp

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// Manifest is the parsed in-memory shape of a scenario's cli/manifest.json.
// Mirrors .vrooli/schemas/cli-manifest.schema.json (id "cli-manifest/v1").
// LoadFromManifest builds a SubcommandGroup from this shape; tests use
// RequireProtoServiceCoverage to assert proto methods are bound or omitted.
type Manifest struct {
	Name        string             `json:"name"`
	Description string             `json:"description,omitempty"`
	Groups      []ManifestGroup    `json:"groups"`
	Omitted     []ManifestOmission `json:"omitted,omitempty"`
}

type ManifestGroup struct {
	Name        string             `json:"name"`
	Description string             `json:"description,omitempty"`
	Commands    []ManifestCommand  `json:"commands"`
}

type ManifestCommand struct {
	Name        string                  `json:"name"`
	Description string                  `json:"description,omitempty"`
	Positionals []ManifestPositional    `json:"positionals,omitempty"`
	Flags       []ManifestFlag          `json:"flags,omitempty"`
	Binding     ManifestBinding         `json:"binding"`
	Governance  ManifestGovernance      `json:"governance"`
}

type ManifestPositional struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required,omitempty"`
	Repeated    bool   `json:"repeated,omitempty"`
}

type ManifestFlag struct {
	Name        string             `json:"name"`
	Aliases     []string           `json:"aliases,omitempty"`
	Description string             `json:"description,omitempty"`
	Required    bool               `json:"required,omitempty"`
	Default     string             `json:"default,omitempty"`
	Bool        bool               `json:"bool,omitempty"`
	Bind        *ManifestFlagBind  `json:"bind,omitempty"`
}

// ManifestFlagBind, when set, declares how the flag's parsed value maps
// onto the RPC request message. See FlagBind for the runtime contract.
type ManifestFlagBind struct {
	Field string `json:"field"`
	Kind  string `json:"kind,omitempty"`
}

type ManifestBinding struct {
	Kind    string `json:"kind"`
	Service string `json:"service,omitempty"`
	Method  string `json:"method,omitempty"`
}

type ManifestGovernance struct {
	Effect               string   `json:"effect"`
	RunEligible          bool     `json:"run_eligible"`
	RequiresConfirmation *bool    `json:"requires_confirmation,omitempty"`
	Permissions          []string `json:"permissions,omitempty"`
}

type ManifestOmission struct {
	Service string `json:"service"`
	Method  string `json:"method"`
	Reason  string `json:"reason"`
}

// ParseManifest decodes raw JSON manifest bytes into a *Manifest and
// performs basic structural validation (required fields, known binding kinds).
// Full JSON-Schema validation is intentionally deferred to test-time via
// RequireProtoServiceCoverage; at runtime, scenarios that have shipped a
// broken manifest will fail loudly here.
func ParseManifest(raw []byte) (*Manifest, error) {
	var m Manifest
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&m); err != nil {
		// Retry permissively for forward-compat with $schema and other
		// editor-only top-level fields. Schema validation in tests catches
		// genuinely unknown structural fields.
		if jerr := json.Unmarshal(raw, &m); jerr != nil {
			return nil, fmt.Errorf("parse cli manifest: %w", jerr)
		}
	}
	if strings.TrimSpace(m.Name) == "" {
		return nil, fmt.Errorf("cli manifest: name is required")
	}
	if len(m.Groups) == 0 {
		return nil, fmt.Errorf("cli manifest %q: at least one group is required", m.Name)
	}
	for gi, g := range m.Groups {
		if strings.TrimSpace(g.Name) == "" {
			return nil, fmt.Errorf("cli manifest %q: group[%d] missing name", m.Name, gi)
		}
		if len(g.Commands) == 0 {
			return nil, fmt.Errorf("cli manifest %q: group %q must declare at least one command", m.Name, g.Name)
		}
		for ci, c := range g.Commands {
			if strings.TrimSpace(c.Name) == "" {
				return nil, fmt.Errorf("cli manifest %q: group %q command[%d] missing name", m.Name, g.Name, ci)
			}
			if c.Binding.Kind != "connect-rpc" {
				return nil, fmt.Errorf("cli manifest %q: command %s/%s binding.kind %q is not supported (only connect-rpc in v1)", m.Name, g.Name, c.Name, c.Binding.Kind)
			}
			if strings.TrimSpace(c.Binding.Service) == "" || strings.TrimSpace(c.Binding.Method) == "" {
				return nil, fmt.Errorf("cli manifest %q: command %s/%s connect-rpc binding requires service+method", m.Name, g.Name, c.Name)
			}
			switch c.Governance.Effect {
			case "read", "write", "destructive":
			default:
				return nil, fmt.Errorf("cli manifest %q: command %s/%s governance.effect %q not in {read,write,destructive}", m.Name, g.Name, c.Name, c.Governance.Effect)
			}
		}
	}
	return &m, nil
}

// FindGroup returns the named group, or nil if absent.
func (m *Manifest) FindGroup(name string) *ManifestGroup {
	for i := range m.Groups {
		if m.Groups[i].Name == name {
			return &m.Groups[i]
		}
	}
	return nil
}

// BindingKey returns the canonical "Service.Method" lookup key for a binding.
func (b ManifestBinding) BindingKey() string {
	return b.Service + "." + b.Method
}

// LoadFromManifest parses raw cli/manifest.json bytes, selects the named
// group, and assembles a SubcommandGroup whose Subcommands wire each
// command's binding ("<Service>.<Method>") to a handler in `bindings`.
//
// Errors on:
//   - manifest parse / structural validation failure
//   - groupName not present in the manifest
//   - a command's binding has no handler in `bindings`
//   - `bindings` contains a key not referenced by any command in the group
//     (catches dead handlers / typos)
//
// Schema validation against cli-manifest.schema.json is a test-time concern
// (see RequireProtoServiceCoverage); broken manifests still fail loudly at
// startup via ParseManifest.
func LoadFromManifest(raw []byte, groupName string, bindings map[string]func(RunContext) error) (SubcommandGroup, error) {
	m, err := ParseManifest(raw)
	if err != nil {
		return SubcommandGroup{}, err
	}
	group := m.FindGroup(groupName)
	if group == nil {
		return SubcommandGroup{}, fmt.Errorf("cli manifest %q: group %q not found (have: %s)", m.Name, groupName, listGroupNames(m))
	}

	used := make(map[string]struct{}, len(bindings))
	subs := make([]Command, 0, len(group.Commands))
	for _, c := range group.Commands {
		key := c.Binding.BindingKey()
		handler, ok := bindings[key]
		if !ok {
			return SubcommandGroup{}, fmt.Errorf("cli manifest %q: command %s/%s binding %s has no registered handler", m.Name, group.Name, c.Name, key)
		}
		used[key] = struct{}{}

		args, err := manifestArgs(c)
		if err != nil {
			return SubcommandGroup{}, fmt.Errorf("cli manifest %q: command %s/%s: %w", m.Name, group.Name, c.Name, err)
		}
		subs = append(subs, Command{
			Name:        c.Name,
			Description: c.Description,
			Args:        args,
			RunCtx:      handler,
		})
	}
	var unused []string
	for k := range bindings {
		if _, ok := used[k]; !ok {
			unused = append(unused, k)
		}
	}
	if len(unused) > 0 {
		sort.Strings(unused)
		return SubcommandGroup{}, fmt.Errorf("cli manifest %q: group %q has registered handlers with no matching command binding: %s", m.Name, group.Name, strings.Join(unused, ", "))
	}

	return SubcommandGroup{
		Name:        group.Name,
		Description: group.Description,
		NeedsAPI:    true,
		Subcommands: subs,
	}, nil
}

// manifestArgs converts a ManifestCommand's positionals/flags into an ArgSchema.
// Validates the resulting schema so contract bugs surface at startup, not on
// first invocation.
func manifestArgs(c ManifestCommand) (ArgSchema, error) {
	args := ArgSchema{}
	for _, p := range c.Positionals {
		args.Positionals = append(args.Positionals, Positional{
			Name:        p.Name,
			Description: p.Description,
			Required:    p.Required,
			Repeated:    p.Repeated,
		})
	}
	for _, f := range c.Flags {
		flag := Flag{
			Name:        f.Name,
			Aliases:     f.Aliases,
			Description: f.Description,
			Required:    f.Required,
			Default:     f.Default,
			Bool:        f.Bool,
		}
		if f.Bind != nil {
			flag.Bind = FlagBind{Field: f.Bind.Field, Kind: f.Bind.Kind}
		}
		args.Flags = append(args.Flags, flag)
	}
	if err := args.Validate(); err != nil {
		return ArgSchema{}, err
	}
	return args, nil
}

func listGroupNames(m *Manifest) string {
	names := make([]string, 0, len(m.Groups))
	for _, g := range m.Groups {
		names = append(names, g.Name)
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}
