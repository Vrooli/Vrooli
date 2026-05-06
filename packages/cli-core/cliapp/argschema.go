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
}

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
	}
	return nil
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
