package cliapp

import (
	"fmt"
	"io"
	"os"
	"strings"
)

// jsonFlagName is reserved as a built-in pseudo-flag for any command using
// the declarative ArgSchema path. Handlers see it via RunContext.JSON().
const jsonFlagName = "json"

// parseArgs turns ([]string) into a RunContext using the ArgSchema. Returns
// ErrHelpRequested when the user passes --help or -h.
func parseArgs(schema ArgSchema, args []string, core *ScenarioApp, stdout, stderr io.Writer) (RunContext, error) {
	if err := schema.Validate(); err != nil {
		return nil, fmt.Errorf("invalid ArgSchema: %w", err)
	}

	ctx := &runContext{
		schema:      schema,
		flagValues:  make(map[string]string),
		flagSet:     make(map[string]bool),
		positionals: make(map[string]string),
		repeated:    make(map[string][]string),
		core:        core,
		stdout:      stdout,
		stderr:      stderr,
	}
	if ctx.stdout == nil {
		ctx.stdout = os.Stdout
	}
	if ctx.stderr == nil {
		ctx.stderr = os.Stderr
	}

	for _, f := range schema.Flags {
		if f.Default != "" && !f.Bool {
			ctx.flagValues[f.Name] = f.Default
		}
	}

	rawPositionals := make([]string, 0, len(args))
	endOfFlags := false
	for i := 0; i < len(args); i++ {
		arg := args[i]

		if endOfFlags {
			rawPositionals = append(rawPositionals, arg)
			continue
		}

		if arg == "--" {
			endOfFlags = true
			continue
		}

		if arg == "--help" || arg == "-h" {
			return nil, ErrHelpRequested
		}

		if !strings.HasPrefix(arg, "-") || arg == "-" {
			rawPositionals = append(rawPositionals, arg)
			continue
		}

		name, inlineValue, hasInline := splitFlagToken(arg)
		canonical := strings.TrimLeft(name, "-")

		if canonical == jsonFlagName {
			if hasInline {
				return nil, fmt.Errorf("--%s does not accept a value", jsonFlagName)
			}
			ctx.jsonOutput = true
			continue
		}

		flag, ok := schema.flagByName(canonical)
		if !ok {
			return nil, fmt.Errorf("unknown option: %s", arg)
		}

		if flag.Bool {
			if hasInline {
				return nil, fmt.Errorf("--%s does not accept a value", flag.Name)
			}
			ctx.flagSet[flag.Name] = true
			continue
		}

		var value string
		if hasInline {
			value = inlineValue
		} else {
			if i+1 >= len(args) {
				return nil, fmt.Errorf("missing value for --%s", flag.Name)
			}
			i++
			value = args[i]
		}
		ctx.flagValues[flag.Name] = value
		ctx.flagSet[flag.Name] = true
	}

	for _, f := range schema.Flags {
		if f.Required && !ctx.flagSet[f.Name] {
			return nil, fmt.Errorf("missing required flag --%s", f.Name)
		}
	}

	if err := assignPositionals(ctx, schema, rawPositionals); err != nil {
		return nil, err
	}
	ctx.rawArgs = rawPositionals
	return ctx, nil
}

func splitFlagToken(arg string) (name, value string, hasValue bool) {
	if eq := strings.Index(arg, "="); eq != -1 {
		return arg[:eq], arg[eq+1:], true
	}
	return arg, "", false
}

func assignPositionals(ctx *runContext, schema ArgSchema, values []string) error {
	specs := schema.Positionals

	if len(specs) == 0 {
		if len(values) > 0 {
			return fmt.Errorf("unexpected positional arguments: %s", strings.Join(values, " "))
		}
		return nil
	}

	minRequired := 0
	for _, p := range specs {
		if p.Required {
			minRequired++
		}
	}
	if len(values) < minRequired {
		missing := specs[len(values)].Name
		return fmt.Errorf("missing required positional <%s>", missing)
	}

	last := specs[len(specs)-1]
	if last.Repeated {
		fixed := specs[:len(specs)-1]
		for i, p := range fixed {
			ctx.positionals[p.Name] = values[i]
		}
		ctx.repeated[last.Name] = append([]string(nil), values[len(fixed):]...)
		if last.Required && len(ctx.repeated[last.Name]) == 0 {
			return fmt.Errorf("missing required positional <%s>", last.Name)
		}
		return nil
	}

	if len(values) > len(specs) {
		return fmt.Errorf("unexpected positional arguments: %s", strings.Join(values[len(specs):], " "))
	}
	for i := range values {
		ctx.positionals[specs[i].Name] = values[i]
	}
	return nil
}
