package commandtree

import (
	"bytes"
	"fmt"
	"io"
	"sort"
	"strings"
)

const (
	commandtreeParameterA = 2
	commandtreeParameterB = 26
)

type RootPolicy struct {
	RequiresRoot      bool
	CanRunWithoutRoot func(args []string) bool
}

type Help struct {
	Title        string
	Usage        string
	DefaultGroup string
	Description  string
	Options      []OptionArg
	Examples     []string
	Notes        []string
}

type PositionalArg struct {
	Name        string
	Required    bool
	Repeatable  bool
	Description string
}

type OptionArg struct {
	Name        string
	Aliases     []string
	ValueName   string
	Repeatable  bool
	Description string
}

type ArgSchema struct {
	Positionals []PositionalArg
	Options     []OptionArg
}

type Spec[H any] struct {
	Name        string
	Aliases     []string
	Group       string
	Summary     string
	Hidden      bool
	Suggestable bool
	RootPolicy  RootPolicy
	Help        Help
	Args        ArgSchema
	Handler     H
}

type Entry struct {
	Name    string
	Group   string
	Summary string
}

func ValidateSpecs[H any](specs []Spec[H]) error {
	seen := make(map[string]string, len(specs))
	for _, spec := range specs {
		names := append([]string{spec.Name}, spec.Aliases...)
		for _, raw := range names {
			name := NormalizeName(raw)
			if name == "" {
				return fmt.Errorf("command name or alias cannot be empty")
			}
			if existing, ok := seen[name]; ok {
				return fmt.Errorf("duplicate command name or alias %q shared by %q and %q", raw, existing, spec.Name)
			}
			seen[name] = spec.Name
		}
	}
	return nil
}

func MustValidateSpecs[H any](specs []Spec[H]) {
	if err := ValidateSpecs(specs); err != nil {
		panic(err)
	}
}

func NormalizeName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

func BindSpec[ID any, H any](spec Spec[ID], handler H) Spec[H] {
	return Spec[H]{
		Name:        spec.Name,
		Aliases:     append([]string(nil), spec.Aliases...),
		Group:       spec.Group,
		Summary:     spec.Summary,
		Hidden:      spec.Hidden,
		Suggestable: spec.Suggestable,
		RootPolicy:  spec.RootPolicy,
		Help:        spec.Help,
		Args:        spec.Args,
		Handler:     handler,
	}
}

func BindSpecs[ID comparable, H any](specs []Spec[ID], handlers map[ID]H) []Spec[H] {
	return BindSpecsFunc(specs, func(spec Spec[ID]) (Spec[H], bool) {
		handler, ok := handlers[spec.Handler]
		if !ok {
			return Spec[H]{}, false
		}
		return BindSpec(spec, handler), true
	})
}

func BindSpecsFunc[ID any, H any](specs []Spec[ID], bind func(Spec[ID]) (Spec[H], bool)) []Spec[H] {
	items := make([]Spec[H], 0, len(specs))
	for _, spec := range specs {
		bound, ok := bind(spec)
		if !ok {
			continue
		}
		items = append(items, bound)
	}
	MustValidateSpecs(items)
	return items
}

func BuildHandlerMap[H any](specs []Spec[H]) map[string]H {
	MustValidateSpecs(specs)
	items := make(map[string]H, len(specs))
	for _, spec := range specs {
		items[NormalizeName(spec.Name)] = spec.Handler
		for _, alias := range spec.Aliases {
			items[NormalizeName(alias)] = spec.Handler
		}
	}
	return items
}

func BuildSpecMap[H any](specs []Spec[H]) map[string]Spec[H] {
	MustValidateSpecs(specs)
	items := make(map[string]Spec[H], len(specs))
	for _, spec := range specs {
		items[NormalizeName(spec.Name)] = spec
		for _, alias := range spec.Aliases {
			items[NormalizeName(alias)] = spec
		}
	}
	return items
}

func SuggestableNames[H any](specs []Spec[H]) []string {
	names := make([]string, 0, len(specs))
	for _, spec := range specs {
		if spec.Suggestable {
			names = append(names, spec.Name)
		}
	}
	sort.Strings(names)
	return names
}

func VisibleEntries[H any](specs []Spec[H], defaultGroup string) []Entry {
	entries := make([]Entry, 0, len(specs))
	for _, spec := range specs {
		if spec.Hidden {
			continue
		}
		group := strings.TrimSpace(spec.Group)
		if group == "" {
			group = defaultGroup
		}
		entries = append(entries, Entry{
			Name:    spec.Name,
			Group:   group,
			Summary: spec.Summary,
		})
	}
	return entries
}

func RenderGroups(w io.Writer, entries []Entry) {
	currentGroup := ""
	for _, entry := range entries {
		group := entry.Group
		if group != currentGroup {
			if currentGroup != "" {
				_, _ = fmt.Fprintln(w)
			}
			_, _ = fmt.Fprintln(w, group+":")
			currentGroup = group
		}
		_, _ = fmt.Fprintf(w, "    %-18s %s\n", entry.Name, entry.Summary)
	}
}

func RenderHelp[H any](w io.Writer, help Help, specs []Spec[H]) {
	if strings.TrimSpace(help.Title) != "" {
		_, _ = io.WriteString(w, help.Title+"\n\n")
	}
	if strings.TrimSpace(help.Description) != "" {
		_, _ = io.WriteString(w, help.Description+"\n\n")
	}
	if strings.TrimSpace(help.Usage) != "" {
		_, _ = io.WriteString(w, "Usage:\n")
		_, _ = io.WriteString(w, "  "+help.Usage+"\n\n")
	}
	RenderGroups(w, VisibleEntries(specs, help.DefaultGroup))
	if len(help.Options) > 0 {
		_, _ = io.WriteString(w, "\nOptions:\n")
		renderOptions(w, help.Options)
	}
	if len(help.Examples) > 0 {
		_, _ = io.WriteString(w, "\nExamples:\n")
		for _, example := range help.Examples {
			_, _ = io.WriteString(w, "  "+example+"\n")
		}
	}
	if len(help.Notes) > 0 {
		_, _ = io.WriteString(w, "\n")
		for _, note := range help.Notes {
			_, _ = io.WriteString(w, note+"\n")
		}
	}
}

func RenderHelpText[H any](help Help, specs []Spec[H]) string {
	var buffer bytes.Buffer
	RenderHelp(&buffer, help, specs)
	return buffer.String()
}

func UsageLine(command string, schema ArgSchema) string {
	command = strings.TrimSpace(command)
	parts := []string{command}
	for _, positional := range schema.Positionals {
		label := usageLabel(positional.Name)
		switch {
		case positional.Repeatable && positional.Required:
			parts = append(parts, "<"+label+">...")
		case positional.Repeatable:
			parts = append(parts, "["+label+"...]")
		case positional.Required:
			parts = append(parts, "<"+label+">")
		default:
			parts = append(parts, "["+label+"]")
		}
	}
	if len(schema.Options) > 0 {
		parts = append(parts, "[options]")
	}
	return strings.TrimSpace(strings.Join(parts, " "))
}

func SpecHelpText[H any](title, command string, spec Spec[H]) string {
	return HelpText(title, command, spec.Summary, spec.Help, spec.Args)
}

func HelpText(title, command, fallbackDescription string, help Help, schema ArgSchema) string {
	var builder strings.Builder

	title = strings.TrimSpace(title)
	description := strings.TrimSpace(help.Description)
	if description == "" {
		description = strings.TrimSpace(fallbackDescription)
	}

	if title != "" {
		builder.WriteString(title)
		builder.WriteString("\n\n")
	}
	builder.WriteString("Usage:\n")
	if usage := strings.TrimSpace(help.Usage); usage != "" {
		builder.WriteString("  ")
		builder.WriteString(usage)
		builder.WriteString("\n")
	} else {
		builder.WriteString("  ")
		builder.WriteString(UsageLine(command, schema))
		builder.WriteString("\n")
	}

	if description != "" {
		builder.WriteString("\n")
		builder.WriteString(description)
		builder.WriteString("\n")
	}

	options := schema.Options
	if len(help.Options) > 0 {
		options = append(append([]OptionArg(nil), schema.Options...), help.Options...)
	}
	if len(options) > 0 {
		builder.WriteString("\nOptions:\n")
		renderOptions(&builder, options)
	}
	builder.WriteString("  --help, -h")
	if len(options) > 0 {
		padding := commandtreeParameterB - len("--help, -h")
		if padding < commandtreeParameterA {
			padding = 2
		}
		builder.WriteString(strings.Repeat(" ", padding))
	} else {
		builder.WriteString("  ")
	}
	builder.WriteString("Show help for this command\n")

	if len(help.Examples) > 0 {
		builder.WriteString("\nExamples:\n")
		for _, example := range help.Examples {
			builder.WriteString("  ")
			builder.WriteString(example)
			builder.WriteString("\n")
		}
	}
	if len(help.Notes) > 0 {
		builder.WriteString("\n")
		for _, note := range help.Notes {
			builder.WriteString(note)
			builder.WriteString("\n")
		}
	}

	return builder.String()
}

func renderOptions(w io.Writer, options []OptionArg) {
	for _, option := range options {
		synopsis := optionSynopsis(option)
		_, _ = io.WriteString(w, "  "+synopsis)
		if description := strings.TrimSpace(option.Description); description != "" {
			padding := commandtreeParameterB - len(synopsis)
			if padding < commandtreeParameterA {
				padding = 2
			}
			_, _ = io.WriteString(w, strings.Repeat(" ", padding)+description)
		}
		_, _ = io.WriteString(w, "\n")
	}
}

func usageLabel(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "arg"
	}
	replacer := strings.NewReplacer(" ", "-", "_", "-", "/", "-")
	return strings.ToLower(replacer.Replace(value))
}

func optionSynopsis(option OptionArg) string {
	parts := []string{strings.TrimSpace(option.Name)}
	for _, alias := range option.Aliases {
		if trimmed := strings.TrimSpace(alias); trimmed != "" {
			parts = append(parts, trimmed)
		}
	}
	synopsis := strings.Join(parts, ", ")
	if valueName := usageLabel(option.ValueName); valueName != "" && strings.TrimSpace(option.ValueName) != "" {
		synopsis += " <" + valueName + ">"
	}
	return synopsis
}

func JSONOption() OptionArg {
	return OptionArg{Name: "--json", Description: "Emit JSON output"}
}

func RunSubcommandSet[H any](
	args []string,
	usage func(io.Writer),
	command string,
	handlers map[string]H,
	stdout io.Writer,
	invoke func(H, []string) error,
) error {
	if len(args) == 0 || WantsHelp(args) {
		usage(stdout)
		return nil
	}
	handler, ok := handlers[NormalizeName(args[0])]
	if !ok {
		return fmt.Errorf("unknown %s command: %s", command, args[0])
	}
	return invoke(handler, args[1:])
}

func WantsHelp(args []string) bool {
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			return true
		}
	}
	return false
}
