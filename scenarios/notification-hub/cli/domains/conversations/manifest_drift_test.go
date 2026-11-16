package conversations

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// The manifest is the one source of the CLI's help: the parser, `--help` and the
// docs all read it. What can drift is the rest: code that reads a flag the
// manifest no longer declares, a declared flag nothing reads, or prose (the
// skill, the packet guide, a flag's own description) that names a flag that is
// gone. These tests fail when they do.

type manifestFlag struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Bind        json.RawMessage `json:"bind"`
}

type manifestCommand struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Flags       []manifestFlag `json:"flags"`
	Positionals []struct {
		Name string `json:"name"`
	} `json:"positionals"`
}

type manifestGroup struct {
	Name     string            `json:"name"`
	Commands []manifestCommand `json:"commands"`
}

func readManifestGroups(t *testing.T) []manifestGroup {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "manifest.json"))
	require.NoError(t, err)
	var manifest struct {
		Groups []manifestGroup `json:"groups"`
	}
	require.NoError(t, json.Unmarshal(raw, &manifest))
	return manifest.Groups
}

// argumentsRead lists the flag and positional names the package's code reads.
func argumentsRead(t *testing.T) map[string]bool {
	t.Helper()
	files, err := filepath.Glob("*.go")
	require.NoError(t, err)
	read := map[string]bool{}
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		parsed, err := parser.ParseFile(token.NewFileSet(), file, nil, 0)
		require.NoError(t, err)
		ast.Inspect(parsed, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok || len(call.Args) == 0 {
				return true
			}
			// given(ctx, "flag") and givenText(ctx, "flag", value) read a flag too.
			if ident, isIdent := call.Fun.(*ast.Ident); isIdent && (ident.Name == "given" || ident.Name == "givenText") && len(call.Args) > 1 {
				if literal, ok := call.Args[1].(*ast.BasicLit); ok && literal.Kind == token.STRING {
					name, err := strconv.Unquote(literal.Value)
					require.NoError(t, err)
					read[name] = true
				}
				return true
			}
			selector, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			switch selector.Sel.Name {
			case "Flag", "BoolFlag", "FlagValues", "Positional":
				if literal, ok := call.Args[0].(*ast.BasicLit); ok && literal.Kind == token.STRING {
					name, err := strconv.Unquote(literal.Value)
					require.NoError(t, err)
					read[name] = true
				}
			}
			return true
		})
	}
	return read
}

func TestConversationsCodeAndManifestDeclareTheSameArguments(t *testing.T) {
	var declared, bound []string
	declaredSet := map[string]bool{}
	for _, group := range readManifestGroups(t) {
		if group.Name != GroupName && group.Name != "pages" {
			continue
		}
		for _, command := range group.Commands {
			// This package implements `pages check` (PageCheck); the pages
			// package implements the rest of its group.
			if group.Name == "pages" && command.Name != "check" {
				continue
			}
			for _, flag := range command.Flags {
				declaredSet[flag.Name] = true
				if len(flag.Bind) > 0 {
					bound = append(bound, flag.Name)
					continue
				}
				declared = append(declared, flag.Name)
			}
			for _, positional := range command.Positionals {
				declaredSet[positional.Name] = true
				declared = append(declared, positional.Name)
			}
		}
	}
	read := argumentsRead(t)

	var unknown []string
	for name := range read {
		if !declaredSet[name] {
			unknown = append(unknown, name)
		}
	}
	sort.Strings(unknown)
	require.Empty(t, unknown, "the code reads arguments the manifest does not declare; the parser would refuse them or return nothing")

	var unread []string
	for _, name := range declared {
		if !read[name] {
			unread = append(unread, name)
		}
	}
	sort.Strings(unread)
	require.Empty(t, unread, "the manifest declares arguments no code reads; help promises behavior that does not exist (bound flags %v are read by the generic dispatcher)", bound)
}

var flagMention = regexp.MustCompile(`(?:^|[^\w-])--([a-z][a-z0-9-]*)`)

// helpWords are flags every command has, or that name another tool's.
var helpWords = map[string]bool{"json": true, "help": true, "dry-run": true, "apply": true}

func TestProseNamesOnlyFlagsTheManifestDeclares(t *testing.T) {
	known := map[string]bool{}
	sources := map[string]string{}
	for _, group := range readManifestGroups(t) {
		for _, command := range group.Commands {
			if group.Name == GroupName {
				sources["manifest: "+command.Name+" description"] = command.Description
			}
			for _, flag := range command.Flags {
				known[flag.Name] = true
				if group.Name == GroupName {
					sources["manifest: --"+flag.Name+" description"] = flag.Description
				}
			}
		}
	}
	for _, doc := range []string{
		filepath.Join("..", "..", "..", "skills", "notification-hub", "SKILL.md"),
		filepath.Join("..", "..", "..", "skills", "notification-hub", "references", "cli-results.md"),
		filepath.Join("..", "..", "..", "docs", "concepts", "DECISION_PACKETS.md"),
	} {
		text, err := os.ReadFile(doc)
		require.NoError(t, err)
		sources[doc] = string(text)
	}
	for where, text := range sources {
		for _, match := range flagMention.FindAllStringSubmatch(text, -1) {
			name := match[1]
			// --hub-* are the page kit's CSS custom properties, not flags.
			if !known[name] && !helpWords[name] && !strings.HasPrefix(name, "hub-") {
				t.Errorf("%s names --%s, which no command in cli/manifest.json declares", where, name)
			}
		}
	}
}

// commandMention finds a command named in prose: `conversations ask`.
// bareCommandMention also finds a bare `ask` in a code span, which is only
// unambiguous in this scenario's own documents, where every command is a
// notification-hub one.
var (
	commandMention     = regexp.MustCompile("(?:notification-hub )?conversations ([a-z][a-z-]*)")
	bareCommandMention = regexp.MustCompile("(?:(?:notification-hub )?conversations |`)([a-z][a-z-]*)")
)

// globalFlags are accepted by every command, before or after the subcommand.
var globalFlags = map[string]bool{"api-base": true, "instance": true, "node": true, "auto-start": true, "no-color": true, "color": true}

// TestOtherScenariosNameOnlyFlagsOfTheCommandTheyCall covers the callers this
// package does not own: a skill or doc elsewhere in the repository that tells an
// agent to run `notification-hub conversations <command>` with a flag that
// command does not declare (a renamed one, most likely) teaches a command that
// fails. Each command named on a line is checked against that command's flags.
func TestOtherScenariosNameOnlyFlagsOfTheCommandTheyCall(t *testing.T) {
	declared := map[string]map[string]bool{}
	for _, group := range readManifestGroups(t) {
		if group.Name != GroupName {
			continue
		}
		for _, command := range group.Commands {
			flags := map[string]bool{}
			for _, flag := range command.Flags {
				flags[flag.Name] = true
			}
			declared[command.Name] = flags
		}
	}
	require.NotEmpty(t, declared)

	scenarios := filepath.Join("..", "..", "..", "..")
	own := filepath.Join(scenarios, "notification-hub")
	var docs []string
	for _, pattern := range []string{
		filepath.Join(scenarios, "*", "skills", "*", "SKILL.md"),
		filepath.Join(scenarios, "*", "skills", "*", "references", "*.md"),
		filepath.Join(scenarios, "*", "docs", "*.md"),
		filepath.Join(scenarios, "*", "docs", "*", "*.md"),
		filepath.Join(scenarios, "..", "docs", "*", "*.md"),
	} {
		matches, err := filepath.Glob(pattern)
		require.NoError(t, err)
		docs = append(docs, matches...)
	}
	require.NotEmpty(t, docs, "the scan found no documents; its paths are wrong")
	checked := 0
	for _, doc := range docs {
		text, err := os.ReadFile(doc)
		require.NoError(t, err)
		for number, line := range strings.Split(string(text), "\n") {
			// A command's flags run from its name to the next command named on
			// the line, so one line naming several commands checks each against
			// its own flags.
			mention := commandMention
			if strings.HasPrefix(filepath.Clean(doc), filepath.Clean(own)) {
				mention = bareCommandMention
			}
			mentions := mention.FindAllStringSubmatchIndex(line, -1)
			for i, found := range mentions {
				command := line[found[2]:found[3]]
				flags, isCommand := declared[command]
				if !isCommand {
					continue
				}
				end := len(line)
				if i+1 < len(mentions) {
					end = mentions[i+1][0]
				}
				checked++
				for _, match := range flagMention.FindAllStringSubmatch(line[found[0]:end], -1) {
					name := match[1]
					if !flags[name] && !helpWords[name] && !globalFlags[name] {
						t.Errorf("%s:%d runs `conversations %s` with --%s, which that command does not declare", doc, number+1, command, name)
					}
				}
			}
		}
	}
	require.Positive(t, checked, "no document names a conversations command; the scan reads the wrong files")
}
