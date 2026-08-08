package cliapp

import (
	"fmt"
	"strings"
)

// ArgSchema describes the positionals and flags a Command accepts.
// It feeds two systems:
//   - the parser, which turns ([]string) into a RunContext keyed by these names
//   - helpgen, which renders --help from the same schema (one source of truth)
//
// An empty schema is valid: the command takes no args and the parser still
// honors --help, --, and "no extra positionals" semantics.
type ArgSchema struct {
	Positionals []Positional
	Flags       []Flag
}

// Positional declares a positional argument by lookup name.
//
// Repeated must only be set on the final positional; Validate enforces this.
type Positional struct {
	Name        string
	Description string
	Required    bool
	Repeated    bool
	// LocalOnly marks a parsed CLI control that is intentionally not part of
	// the Connect request.
	LocalOnly bool
	// Bind projects this positional onto a proto request field when its
	// declared name is not sufficient to describe the wire representation.
	Bind FlagBind
}

// Flag declares a flag (long form, optional aliases). Bool flags accept
// no value; valued flags accept --name=value or --name value.
//
// Default applies only to valued flags; for Bool flags, presence implies true
// and absence implies false.
type Flag struct {
	Name        string
	Aliases     []string
	Description string
	Required    bool
	Default     string
	Bool        bool
	// LocalOnly marks a parsed CLI control that is intentionally not part of
	// the Connect request.
	LocalOnly bool
	// Values, when non-empty, is the closed vocabulary this flag accepts.
	// The parser rejects any supplied value that is neither a member of
	// Values nor a key of ValueAliases, and helpgen renders the choices.
	// Only valid on non-Bool flags.
	Values []string
	// ValueAliases maps accepted synonyms to a member of Values
	// (e.g. "low" → "minor"). The parser accepts an alias and passes the
	// raw supplied string through unchanged — canonicalization stays
	// server-side.
	ValueAliases map[string]string
	// Bind, when non-zero, tells the protodispatch handler that this
	// flag's value populates a specific proto field via a specific
	// encoding. Without it, hydrateFromContext falls back to matching
	// the flag name against top-level scalar fields by name.
	Bind FlagBind
}

// FlagBind declares how a flag's value should be projected onto a proto
// request field. Used by protodispatch when the flag's name doesn't
// match the target field (e.g. --flow-file → flow_definition) or when
// the value is a file path or JSON literal rather than a scalar.
type FlagBind struct {
	// Field is the snake_case proto field name on the RPC request.
	Field string
	// Kind selects the decoding strategy:
	//   - "json_file":   value is a path; load + protojson-decode into the field
	//   - "json_inline": value is a JSON literal; protojson-decode into the field
	//   - "raw_string":  value is the literal string; set as a string field
	//   - "":            no special handling (scalar fallback)
	Kind string
}

// IsZero reports whether the binding is unset.
func (b FlagBind) IsZero() bool { return b.Field == "" && b.Kind == "" }

// Validate returns an error if the schema is malformed. Schemas should be
// validated at registration time so programmer errors surface before any
// user input is parsed.
func (s ArgSchema) Validate() error {
	seen := make(map[string]string, len(s.Positionals)+len(s.Flags))

	for i, p := range s.Positionals {
		name := strings.TrimSpace(p.Name)
		if name == "" {
			return fmt.Errorf("positional[%d] has empty Name", i)
		}
		if existing, ok := seen[name]; ok {
			return fmt.Errorf("positional %q duplicates %s", name, existing)
		}
		seen[name] = "positional " + name
		if p.Repeated && i != len(s.Positionals)-1 {
			return fmt.Errorf("positional %q is Repeated but is not the final positional", name)
		}
		if p.Required && i > 0 && !s.Positionals[i-1].Required {
			return fmt.Errorf("positional %q is Required after optional positional %q", name, s.Positionals[i-1].Name)
		}
		if err := validateBind(p.Bind, "positional "+name, false); err != nil {
			return err
		}
	}

	for i, f := range s.Flags {
		name := strings.TrimSpace(f.Name)
		if name == "" {
			return fmt.Errorf("flag[%d] has empty Name", i)
		}
		if strings.HasPrefix(name, "-") {
			return fmt.Errorf("flag %q must not include the leading dash", name)
		}
		if existing, ok := seen[name]; ok {
			return fmt.Errorf("flag %q duplicates %s", name, existing)
		}
		seen[name] = "flag --" + name
		for _, alias := range f.Aliases {
			alias = strings.TrimSpace(alias)
			if alias == "" {
				return fmt.Errorf("flag %q has empty alias", name)
			}
			if strings.HasPrefix(alias, "-") {
				return fmt.Errorf("flag %q alias %q must not include the leading dash", name, alias)
			}
			if existing, ok := seen[alias]; ok {
				return fmt.Errorf("flag alias %q (for --%s) duplicates %s", alias, name, existing)
			}
			seen[alias] = "alias for --" + name
		}
		if f.Bool && f.Default != "" {
			return fmt.Errorf("flag %q is Bool; Default must be empty (presence implies true)", name)
		}
		if err := validateFlagValues(f, name); err != nil {
			return err
		}
		if err := validateBind(f.Bind, "flag "+name, f.Bool); err != nil {
			return err
		}
	}
	return nil
}

func validateBind(bind FlagBind, owner string, boolFlag bool) error {
	if bind.IsZero() {
		return nil
	}
	if strings.TrimSpace(bind.Field) == "" {
		return fmt.Errorf("%s has bind with empty field", owner)
	}
	switch bind.Kind {
	case "", "raw_string", "json_inline", "json_file":
	default:
		return fmt.Errorf("%s bind.kind %q not in {raw_string,json_inline,json_file}", owner, bind.Kind)
	}
	if boolFlag && bind.Kind != "" && bind.Kind != "raw_string" {
		return fmt.Errorf("%s is Bool; bind.kind must be empty or raw_string", owner)
	}
	return nil
}

// validateFlagValues checks the Values/ValueAliases declaration on one flag.
func validateFlagValues(f Flag, name string) error {
	if len(f.Values) == 0 {
		if len(f.ValueAliases) > 0 {
			return fmt.Errorf("flag %q declares ValueAliases without Values", name)
		}
		return nil
	}
	if f.Bool {
		return fmt.Errorf("flag %q is Bool; Values requires a valued flag", name)
	}
	members := make(map[string]bool, len(f.Values))
	for i, v := range f.Values {
		if strings.TrimSpace(v) == "" {
			return fmt.Errorf("flag %q Values[%d] is empty", name, i)
		}
		if members[v] {
			return fmt.Errorf("flag %q Values contains duplicate %q", name, v)
		}
		members[v] = true
	}
	for alias, target := range f.ValueAliases {
		if strings.TrimSpace(alias) == "" {
			return fmt.Errorf("flag %q has empty value alias", name)
		}
		if members[alias] {
			return fmt.Errorf("flag %q value alias %q duplicates a declared value", name, alias)
		}
		if !members[target] {
			return fmt.Errorf("flag %q value alias %q maps to %q which is not a declared value", name, alias, target)
		}
	}
	if f.Default != "" && !f.acceptsValue(f.Default) {
		return fmt.Errorf("flag %q Default %q is neither a declared value nor an alias", name, f.Default)
	}
	return nil
}

// acceptsValue reports whether v is a member of the flag's declared
// vocabulary (a value or an alias). Flags without Values accept anything.
func (f Flag) acceptsValue(v string) bool {
	if len(f.Values) == 0 {
		return true
	}
	for _, allowed := range f.Values {
		if v == allowed {
			return true
		}
	}
	_, ok := f.ValueAliases[v]
	return ok
}

// flagByName returns the Flag matching a parser token (canonical name or alias).
// The boolean is false if no flag matches.
func (s ArgSchema) flagByName(name string) (Flag, bool) {
	for _, f := range s.Flags {
		if f.Name == name {
			return f, true
		}
		for _, alias := range f.Aliases {
			if alias == name {
				return f, true
			}
		}
	}
	return Flag{}, false
}
