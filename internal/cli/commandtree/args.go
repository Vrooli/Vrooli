package commandtree

import (
	"fmt"
	"strings"

	"github.com/vrooli/vrooli/internal/cli/clipolicy"
)

const (
	argsParameterA = 2
)

type ParsedArgs struct {
	Positionals []string
	flags       map[string][]string
}

func (p ParsedArgs) HasFlag(name string) bool {
	return len(p.flags[name]) > 0
}

func (p ParsedArgs) FlagValue(name string) string {
	values := p.flags[name]
	if len(values) == 0 {
		return ""
	}
	return values[len(values)-1]
}

func (p ParsedArgs) FlagValues(name string) []string {
	values := p.flags[name]
	return append([]string(nil), values...)
}

func ParseArgs(command, helpText string, schema ArgSchema, args []string) (ParsedArgs, error) {
	index := optionIndex(schema)
	parsed := ParsedArgs{
		Positionals: make([]string, 0, len(args)),
		flags:       make(map[string][]string, len(schema.Options)),
	}

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--help", "-h":
			return ParsedArgs{}, clipolicy.CommandHelpOnly(helpText)
		case "--":
			parsed.Positionals = append(parsed.Positionals, args[i+1:]...)
			i = len(args)
			continue
		}

		if strings.HasPrefix(arg, "-") {
			next, err := consumeOption(command, args, i, index, parsed.flags)
			if err != nil {
				return ParsedArgs{}, err
			}
			i = next
			continue
		}

		parsed.Positionals = append(parsed.Positionals, arg)
	}

	if err := validatePositionals(command, schema.Positionals, parsed.Positionals); err != nil {
		return ParsedArgs{}, err
	}
	return parsed, nil
}

func consumeOption(command string, args []string, position int, index map[string]OptionArg, flags map[string][]string) (int, error) {
	arg := args[position]
	name, value, hasValue := splitOptionToken(arg)
	option, ok := index[name]
	if !ok {
		return position, clipolicy.UnknownOptionError(command, arg)
	}
	if option.ValueName == "" {
		if hasValue {
			return position, clipolicy.UsageErrorf(command, "%s does not accept a value", arg)
		}
		flags[option.Name] = append(flags[option.Name], "true")
		return position, nil
	}
	if !hasValue {
		position++
		if position >= len(args) {
			return position, clipolicy.UsageErrorf(command, "missing value for %s", arg)
		}
		value = args[position]
	}
	flags[option.Name] = append(flags[option.Name], value)
	return position, nil
}

func ParseNoArgs(command, helpText string, args []string) error {
	_, err := ParseArgs(command, helpText, ArgSchema{}, args)
	return err
}

func ParseSinglePositional(command, helpText, name string, args []string) (string, error) {
	parsed, err := ParseArgs(command, helpText, ArgSchema{
		Positionals: []PositionalArg{{Name: name, Required: true}},
	}, args)
	if err != nil {
		return "", err
	}
	return parsed.Positionals[0], nil
}

func optionIndex(schema ArgSchema) map[string]OptionArg {
	index := make(map[string]OptionArg, len(schema.Options)*argsParameterA)
	for _, option := range schema.Options {
		index[option.Name] = option
		for _, alias := range option.Aliases {
			index[alias] = option
		}
	}
	return index
}

func splitOptionToken(arg string) (string, string, bool) {
	if !strings.Contains(arg, "=") {
		return arg, "", false
	}
	parts := strings.SplitN(arg, "=", argsParameterA)
	return parts[0], parts[1], true
}

func validatePositionals(command string, specs []PositionalArg, values []string) error {
	if len(specs) == 0 {
		if len(values) > 0 {
			return clipolicy.UsageErrorf(command, "%s does not accept positional arguments", command)
		}
		return nil
	}

	minRequired := 0
	maxAllowed := 0
	repeatable := false
	for i, spec := range specs {
		if spec.Required {
			minRequired++
		}
		if spec.Repeatable {
			if i != len(specs)-1 {
				return fmt.Errorf("repeatable positional %q must be the final positional", spec.Name)
			}
			repeatable = true
			maxAllowed = -1
			break
		}
		maxAllowed++
	}

	if len(values) < minRequired {
		if len(specs) == 1 {
			label := positionalLabel(specs[0])
			if specs[0].Repeatable {
				return clipolicy.UsageErrorf(command, "%s requires at least one %s", command, label)
			}
			if specs[0].Required {
				return clipolicy.UsageErrorf(command, "%s requires exactly one %s", command, label)
			}
		}
		return clipolicy.UsageErrorf(command, "%s requires more positional arguments", command)
	}

	if !repeatable && len(values) > maxAllowed {
		if len(specs) == 1 {
			label := positionalLabel(specs[0])
			if specs[0].Required {
				return clipolicy.UsageErrorf(command, "%s requires exactly one %s", command, label)
			}
			return clipolicy.UsageErrorf(command, "%s accepts at most one %s", command, label)
		}
		return clipolicy.UsageErrorf(command, "%s accepts at most %d positional arguments", command, maxAllowed)
	}

	return nil
}

func positionalLabel(spec PositionalArg) string {
	label := strings.TrimSpace(spec.Description)
	if label == "" {
		label = strings.TrimSpace(spec.Name)
	}
	if label == "" {
		return "argument"
	}
	return label
}
