package cliapp

import (
	"fmt"
	"sort"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
)

// Command-name resolution and recovery for every cli-core CLI.
//
// Agents guess command shapes from the vocabulary of other tools. Two kinds of
// guess are handled here, at the shared dispatcher, so no CLI needs its own
// alias table for them:
//
//   - A singular/plural form of exactly one existing name ("runs" for "run",
//     "execution" for "executions") is the same command. It dispatches
//     directly; an exact name or alias always wins first.
//   - Anything else is unknown. The error names the closest real commands by
//     full path with their usage lines: a sibling verb with the same meaning
//     ("show" for "get"), the same leaf in another group ("run" for
//     "library run"), or a near spelling. One corrected retry, no help lookup.
//
// Suggestions never execute: a synonym can differ in effect, so only the
// owner may declare it as an alias (Command.Aliases / manifest "aliases").

// maxSuggestions bounds the recovery list so an error stays short.
const maxSuggestions = 3

// verbFamilies groups command verbs that agents use interchangeably. Members
// are suggested for each other; they are never silently substituted.
var verbFamilies = [][]string{
	{"show", "get", "describe", "info", "inspect", "view", "read", "cat", "details", "display"},
	{"list", "ls", "all", "index", "enumerate"},
	{"delete", "remove", "rm", "del", "destroy", "purge"},
	{"create", "add", "new", "make", "register"},
	{"update", "edit", "set", "modify", "patch", "change"},
	{"logs", "log", "output", "events", "tail"},
	{"run", "exec", "execute", "start", "invoke", "launch", "trigger"},
	{"stop", "cancel", "abort", "kill", "halt", "terminate"},
	{"status", "state", "health", "check", "ready", "readiness"},
	{"watch", "follow", "wait", "stream"},
	{"search", "find", "query", "lookup", "discover"},
	{"download", "fetch", "pull", "export", "save"},
	{"artifacts", "artifact", "evidence", "outputs", "files"},
}

// sameWordForm reports whether a and b are singular/plural forms of one word
// ("run"/"runs", "process"/"processes", "policy"/"policies"). The shorter
// form must keep at least three letters so short tokens never collide.
func sameWordForm(a, b string) bool {
	a, b = strings.ToLower(strings.TrimSpace(a)), strings.ToLower(strings.TrimSpace(b))
	if a == "" || b == "" || a == b {
		return false
	}
	if len(a) > len(b) {
		a, b = b, a
	}
	if len(a) < 3 {
		return false
	}
	switch {
	case b == a+"s", b == a+"es":
		return true
	case strings.HasSuffix(a, "y") && b == strings.TrimSuffix(a, "y")+"ies":
		return true
	default:
		return false
	}
}

func verbSynonyms(word string) []string {
	word = strings.ToLower(strings.TrimSpace(word))
	var out []string
	for _, family := range verbFamilies {
		for _, member := range family {
			if member == word || sameWordForm(member, word) {
				out = append(out, family...)
				break
			}
		}
	}
	return out
}

func commandNames(cmd Command) []string {
	names := make([]string, 0, 1+len(cmd.Aliases))
	names = append(names, cmd.Name)
	for _, alias := range cmd.Aliases {
		if !strings.HasPrefix(alias, "-") {
			names = append(names, alias)
		}
	}
	return names
}

// resolveTopLevelForm finds the one top-level command or group whose name is
// a singular/plural form of token. ok is false when none or several match.
func (a *App) resolveTopLevelForm(token string) (cmd Command, group *SubcommandGroup, ok bool) {
	var foundCmd *Command
	var foundGroup *SubcommandGroup
	matches := 0
	for i := range a.commands {
		for _, name := range commandNames(a.commands[i]) {
			if sameWordForm(token, name) {
				if foundCmd == nil || foundCmd.Name != a.commands[i].Name {
					matches++
				}
				foundCmd = &a.commands[i]
				break
			}
		}
	}
	for i := range a.opts.SubcommandGroups {
		g := &a.opts.SubcommandGroups[i]
		for _, name := range append([]string{g.Name}, g.Aliases...) {
			if sameWordForm(token, name) {
				matches++
				foundGroup = g
				break
			}
		}
	}
	if matches != 1 {
		return Command{}, nil, false
	}
	if foundGroup != nil {
		return Command{}, foundGroup, true
	}
	return *foundCmd, nil, true
}

// findSubcommand returns the leaf of group named token: exact name or alias
// first, then the one leaf whose name is a singular/plural form of token.
func findSubcommand(group *SubcommandGroup, token string) *Command {
	for i := range group.Subcommands {
		for _, name := range commandNames(group.Subcommands[i]) {
			if name == token {
				return &group.Subcommands[i]
			}
		}
	}
	var found *Command
	for i := range group.Subcommands {
		for _, name := range commandNames(group.Subcommands[i]) {
			if sameWordForm(token, name) {
				if found != nil && found.Name != group.Subcommands[i].Name {
					return nil
				}
				found = &group.Subcommands[i]
				break
			}
		}
	}
	return found
}

// commandSuggestion is one real command offered for an unknown token.
type commandSuggestion struct {
	path        string // command path below the app name, e.g. "library run"
	usage       string // full usage line
	description string
}

type suggestionSet struct {
	appName string
	seen    map[string]bool
	items   []commandSuggestion
}

func newSuggestionSet(appName string) *suggestionSet {
	return &suggestionSet{appName: appName, seen: map[string]bool{}}
}

func (s *suggestionSet) addCommand(groupName string, cmd Command) {
	path := strings.TrimSpace(groupName + " " + cmd.Name)
	if path == "" || s.seen[path] || cmd.Name == "help" {
		return
	}
	s.seen[path] = true
	s.items = append(s.items, commandSuggestion{
		path:        path,
		usage:       commandUsageLine(strings.TrimSpace(s.appName+" "+groupName), cmd),
		description: strings.TrimSpace(cmd.Description),
	})
}

func (s *suggestionSet) addGroup(group *SubcommandGroup) {
	if group == nil || s.seen[group.Name] {
		return
	}
	s.seen[group.Name] = true
	s.items = append(s.items, commandSuggestion{
		path:        group.Name,
		usage:       fmt.Sprintf("%s %s <subcommand> [options]", s.appName, group.Name),
		description: strings.TrimSpace(group.Description),
	})
}

// commandUsageLine is the usage line help would print for cmd under prefix.
func commandUsageLine(prefix string, cmd Command) string {
	if usage := strings.TrimSpace(cmd.Usage); usage != "" {
		return usage
	}
	fullName := strings.TrimSpace(prefix + " " + cmd.Name)
	if cmd.RunCtx != nil {
		return usageLine(fullName, cmd.Args)
	}
	return fullName
}

// render returns the " (did you mean "x"?)" fragment for the error's first
// line and the indented usage lines that follow it.
func (s *suggestionSet) render() (fragment, detail string) {
	if len(s.items) == 0 {
		return "", ""
	}
	items := s.items
	if len(items) > maxSuggestions {
		items = items[:maxSuggestions]
	}
	fragment = fmt.Sprintf(" (did you mean %q?)", items[0].path)
	width := 0
	for _, item := range items {
		if len(item.usage) > width {
			width = len(item.usage)
		}
	}
	var b strings.Builder
	for _, item := range items {
		if item.description == "" {
			fmt.Fprintf(&b, "\n  %s", item.usage)
			continue
		}
		fmt.Fprintf(&b, "\n  %-*s  %s", width, item.usage, item.description)
	}
	return fragment, b.String()
}

func (a *App) groupsInOrder() []*SubcommandGroup {
	out := make([]*SubcommandGroup, 0, len(a.opts.SubcommandGroups))
	for i := range a.opts.SubcommandGroups {
		out = append(out, &a.opts.SubcommandGroups[i])
	}
	return out
}

func (a *App) topLevelCommands() []Command {
	out := make([]Command, 0, len(a.commands))
	for _, cmd := range a.commands {
		if cmd.Name == "help" || cmd.Name == "version" {
			continue
		}
		out = append(out, cmd)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// suggestTopLevel ranks real commands for an unknown first token: the same
// leaf inside a group, a verb with the same meaning, then a near spelling.
func (a *App) suggestTopLevel(token string) *suggestionSet {
	set := newSuggestionSet(a.opts.Name)
	groups := a.groupsInOrder()
	matchesLeaf := func(cmd Command, match func(string) bool) bool {
		for _, name := range commandNames(cmd) {
			if match(name) {
				return true
			}
		}
		return false
	}
	sameName := func(name string) bool { return name == token || sameWordForm(name, token) }
	for _, group := range groups {
		for _, cmd := range group.Subcommands {
			if matchesLeaf(cmd, sameName) {
				set.addCommand(group.Name, cmd)
			}
		}
	}
	if synonyms := verbSynonyms(token); len(synonyms) > 0 {
		isSynonym := func(name string) bool { return containsString(synonyms, name) }
		for _, cmd := range a.topLevelCommands() {
			if matchesLeaf(cmd, isSynonym) {
				set.addCommand("", cmd)
			}
		}
		for _, group := range groups {
			for _, cmd := range group.Subcommands {
				if matchesLeaf(cmd, isSynonym) {
					set.addCommand(group.Name, cmd)
				}
			}
		}
	}
	names := make([]string, 0, len(a.commandLookup)+len(groups))
	for name := range a.commandLookup {
		if !strings.HasPrefix(name, "-") {
			names = append(names, name)
		}
	}
	for _, group := range groups {
		names = append(names, group.Name)
	}
	if nearest := cliutil.NearestString(token, names, 2); nearest != "" {
		if group, ok := a.subcommandGroupLookup[nearest]; ok {
			set.addGroup(group)
		} else if cmd, ok := a.commandLookup[nearest]; ok {
			set.addCommand("", cmd)
		}
	}
	return set
}

// suggestSubcommand ranks real commands for an unknown leaf inside group: a
// sibling verb with the same meaning, a near spelling, then the same leaf in
// another group.
func (a *App) suggestSubcommand(group *SubcommandGroup, token string) *suggestionSet {
	set := newSuggestionSet(a.opts.Name)
	if synonyms := verbSynonyms(token); len(synonyms) > 0 {
		for _, cmd := range group.Subcommands {
			for _, name := range commandNames(cmd) {
				if containsString(synonyms, name) {
					set.addCommand(group.Name, cmd)
					break
				}
			}
		}
	}
	names := make([]string, 0, len(group.Subcommands))
	for _, cmd := range group.Subcommands {
		names = append(names, commandNames(cmd)...)
	}
	if nearest := cliutil.NearestString(token, names, 2); nearest != "" {
		if cmd := findSubcommand(group, nearest); cmd != nil {
			set.addCommand(group.Name, *cmd)
		}
	}
	for _, other := range a.groupsInOrder() {
		if other.Name == group.Name {
			continue
		}
		if cmd := findSubcommand(other, token); cmd != nil {
			set.addCommand(other.Name, *cmd)
		}
	}
	if cmd, ok := a.commandLookup[token]; ok && !strings.HasPrefix(token, "-") {
		set.addCommand("", cmd)
	}
	if other, ok := a.subcommandGroupLookup[token]; ok && other.Name != group.Name {
		set.addGroup(other)
	}
	return set
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
