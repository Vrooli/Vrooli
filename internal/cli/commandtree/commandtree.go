package commandtree

import (
	"fmt"
	"io"
	"sort"
	"strings"
)

type RootPolicy struct {
	RequiresRoot      bool
	CanRunWithoutRoot func(args []string) bool
}

type Help struct {
	Title        string
	Usage        string
	DefaultGroup string
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
	Handler     H
}

type Entry struct {
	Name    string
	Group   string
	Summary string
}

func NormalizeName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

func BuildHandlerMap[H any](specs []Spec[H]) map[string]H {
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
	if strings.TrimSpace(help.Usage) != "" {
		_, _ = io.WriteString(w, "Usage:\n")
		_, _ = io.WriteString(w, "  "+help.Usage+"\n\n")
	}
	RenderGroups(w, VisibleEntries(specs, help.DefaultGroup))
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
