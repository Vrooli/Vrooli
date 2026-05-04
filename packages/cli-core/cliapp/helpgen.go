package cliapp

import (
	"fmt"
	"io"
	"strings"
)

// renderHelp generates help text for a Command from its metadata + ArgSchema.
// One source of truth: the schema feeds both the parser and the help output,
// so a flag added to the schema automatically appears in --help.
func renderHelp(prefix string, cmd Command, w io.Writer) error {
	fullName := strings.TrimSpace(prefix + " " + cmd.Name)
	title := strings.TrimSpace(fullName)

	if cmd.Description != "" {
		fmt.Fprintf(w, "%s - %s\n\n", title, cmd.Description)
	} else {
		fmt.Fprintf(w, "%s\n\n", title)
	}

	usage := strings.TrimSpace(cmd.Usage)
	if usage == "" {
		usage = usageLine(fullName, cmd.Args)
	}
	fmt.Fprintf(w, "Usage:\n  %s\n", usage)

	if len(cmd.Aliases) > 0 {
		fmt.Fprintf(w, "\nAliases:\n  %s\n", strings.Join(cmd.Aliases, ", "))
	}

	if len(cmd.Args.Positionals) > 0 {
		fmt.Fprintln(w, "\nArguments:")
		for _, p := range cmd.Args.Positionals {
			label := positionalUsageLabel(p)
			fmt.Fprintf(w, "  %-22s %s\n", label, strings.TrimSpace(p.Description))
		}
	}

	hasFlags := len(cmd.Args.Flags) > 0
	if hasFlags {
		fmt.Fprintln(w, "\nOptions:")
		for _, f := range cmd.Args.Flags {
			synopsis := flagSynopsis(f)
			suffix := strings.TrimSpace(f.Description)
			if f.Required {
				suffix = strings.TrimSpace("(required) " + suffix)
			}
			if !f.Bool && f.Default != "" {
				suffix = strings.TrimSpace(suffix + fmt.Sprintf(" (default: %s)", f.Default))
			}
			fmt.Fprintf(w, "  %-22s %s\n", synopsis, suffix)
		}
		fmt.Fprintf(w, "  %-22s %s\n", "--json", "Emit JSON output instead of human format")
		fmt.Fprintf(w, "  %-22s %s\n", "--help, -h", "Show help for this command")
	} else {
		fmt.Fprintln(w, "\nOptions:")
		fmt.Fprintf(w, "  %-22s %s\n", "--json", "Emit JSON output instead of human format")
		fmt.Fprintf(w, "  %-22s %s\n", "--help, -h", "Show help for this command")
	}

	if longText := strings.TrimSpace(cmd.LongDescription); longText != "" {
		fmt.Fprintf(w, "\n%s\n", longText)
	}

	if helpText := strings.TrimSpace(cmd.HelpText); helpText != "" {
		fmt.Fprintf(w, "\n%s\n", helpText)
	}

	return nil
}

func usageLine(name string, schema ArgSchema) string {
	parts := []string{name}
	for _, p := range schema.Positionals {
		parts = append(parts, positionalUsageLabel(p))
	}
	if len(schema.Flags) > 0 {
		parts = append(parts, "[options]")
	}
	return strings.Join(parts, " ")
}

func positionalUsageLabel(p Positional) string {
	name := strings.TrimSpace(p.Name)
	if name == "" {
		name = "arg"
	}
	switch {
	case p.Repeated && p.Required:
		return "<" + name + ">..."
	case p.Repeated:
		return "[" + name + "...]"
	case p.Required:
		return "<" + name + ">"
	default:
		return "[" + name + "]"
	}
}

func flagSynopsis(f Flag) string {
	parts := []string{"--" + f.Name}
	for _, alias := range f.Aliases {
		if len(alias) == 1 {
			parts = append(parts, "-"+alias)
		} else {
			parts = append(parts, "--"+alias)
		}
	}
	out := strings.Join(parts, ", ")
	if !f.Bool {
		out += " <value>"
	}
	return out
}
