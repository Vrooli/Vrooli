package scenariocli

import (
	"encoding/json"
	"strings"
	"testing"

	manifest "github.com/vrooli/vrooli/cli"
	"github.com/vrooli/vrooli/internal/cli/commandtree"
)

// Every scenario command is parsed twice: cli/manifest.json validates the raw
// argv before dispatch, and the command tree parses it again inside the
// handler. The command tree is also what renders --help.
//
// When a flag reaches only the command tree, `--help` advertises an option the
// parser rejects as "unknown option", which reads to an operator as a broken
// binary rather than a missing declaration. This has happened three times:
// --variant-dependencies on start, and --force-lifecycle plus
// --lifecycle-override-reason on start. Checking one command at a time only
// finds the one you thought to check, so this walks the whole surface.
func TestEveryCommandTreeFlagIsDeclaredInTheManifest(t *testing.T) {
	declared := manifestScenarioCommandFlags(t)

	// A few commands keep their schema outside the spec table, so walking
	// CommandSpecs alone would skip them silently — `logs` is one, and it is
	// exactly where the most recent instance of this defect was found.
	external := map[string]commandtree.ArgSchema{
		string(CommandLogs): scenarioLogsArgSchema(),
	}

	checked := 0
	for _, spec := range CommandSpecs() {
		if extra, ok := external[spec.Name]; ok {
			spec.Args.Options = append(append([]commandtree.OptionArg(nil), spec.Args.Options...), extra.Options...)
		}
		flags, known := declared[spec.Name]
		if !known {
			// Not every command in the tree is manifest-bound; those are
			// dispatched by other means and are out of scope here.
			continue
		}
		checked++
		for _, option := range spec.Args.Options {
			name := strings.TrimPrefix(option.Name, "--")
			// --json is supplied by the manifest runtime for every command.
			if name == "json" {
				continue
			}
			if !flags[name] {
				t.Errorf("scenario %s: --%s is in the command tree (so --help shows it) but missing from cli/manifest.json, so the parser rejects it as an unknown option", spec.Name, name)
			}
		}
	}

	// A refactor that renamed the group or stopped exporting the specs would
	// make this test vacuously green.
	if _, ok := declared[string(CommandLogs)]; !ok {
		t.Error("scenario logs is not in the manifest lookup; its externally-defined schema would go unchecked")
	}
	if checked < 5 {
		t.Fatalf("only %d scenario commands were compared; the manifest lookup is probably not matching command names any more", checked)
	}
}

// manifestScenarioCommandFlags returns, per scenario command name, the set of
// flag names cli/manifest.json declares for it.
func manifestScenarioCommandFlags(t *testing.T) map[string]map[string]bool {
	t.Helper()
	var doc struct {
		Groups []struct {
			Name     string `json:"name"`
			Commands []struct {
				Name  string `json:"name"`
				Flags []struct {
					Name string `json:"name"`
				} `json:"flags"`
			} `json:"commands"`
		} `json:"groups"`
	}
	if err := json.Unmarshal(manifest.Bytes(), &doc); err != nil {
		t.Fatalf("decode cli/manifest.json: %v", err)
	}
	out := map[string]map[string]bool{}
	for _, group := range doc.Groups {
		if group.Name != "scenario" {
			continue
		}
		for _, command := range group.Commands {
			flags := make(map[string]bool, len(command.Flags))
			for _, f := range command.Flags {
				flags[f.Name] = true
			}
			out[command.Name] = flags
		}
	}
	if len(out) == 0 {
		t.Fatal("cli/manifest.json declares no scenario commands")
	}
	return out
}
