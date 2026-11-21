package depsapproved

import (
	"strings"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
	"scenario-dependency-analyzer/cli/internal/support"
)

func registeredGroups() []cliapp.SubcommandGroup {
	return []cliapp.SubcommandGroup{Register(nil), RegisterInstall(nil)}
}

// An agent discovers how to call a command from its --help. A command whose help
// is only "Usage: <name>" gives it nothing to go on, so every subcommand of the
// dependency groups must document its flags and carry an example.
func TestEverySubcommandHelpDocumentsItsFlagsAndAnExample(t *testing.T) {
	checked := 0
	for _, group := range registeredGroups() {
		for _, cmd := range group.Subcommands {
			name := group.Name + " " + cmd.Name
			checked++
			if cmd.RunCtx != nil {
				// A schema command renders its own Options block; it still needs flags and an example.
				if len(cmd.Args.Flags) == 0 {
					t.Errorf("%s: schema command declares no flags", name)
				}
				if !strings.Contains(strings.ToLower(cmd.HelpText+cmd.LongDescription), "example") {
					t.Errorf("%s: help has no example", name)
				}
				continue
			}
			doc, ok := commandDocs[name]
			if !ok {
				t.Errorf("%s: no entry in commandDocs, so its help is only a usage line", name)
				continue
			}
			if doc.Usage == "" || doc.Example == "" {
				t.Errorf("%s: usage %q and example %q must both be set", name, doc.Usage, doc.Example)
			}
			if !strings.HasPrefix(doc.Example, "deps") {
				t.Errorf("%s: example %q must be a runnable %s command", name, doc.Example, support.AppName)
			}
			body := helpBody(cmd, doc)
			if flags := support.DescribeFlags(cmd.Run); len(flags) == 0 {
				t.Errorf("%s: the command defines no flags to document", name)
			}
			for _, want := range []string{"Options:", "--json", "Example:", support.AppName + " " + doc.Example} {
				if !strings.Contains(body, want) {
					t.Errorf("%s: help lacks %q:\n%s", name, want, body)
				}
			}
		}
	}
	if checked < 20 {
		t.Fatalf("only %d subcommands registered; the groups changed shape", checked)
	}
}

func TestHelpListsTheFlagsACommandActuallyAccepts(t *testing.T) {
	for _, group := range registeredGroups() {
		for _, cmd := range group.Subcommands {
			if group.Name+" "+cmd.Name != "deps-approved approve" {
				continue
			}
			body := helpBody(cmd, commandDocs["deps-approved approve"])
			for _, flag := range []string{"--rationale <value>", "--range <value>", "--range-policy <value>", "--scenarios <value>", "--apply "} {
				if !strings.Contains(body, flag) {
					t.Errorf("approve help lacks %q:\n%s", flag, body)
				}
			}
			return
		}
	}
	t.Fatal("deps-approved approve is not registered")
}

func TestDescribeFlagsDoesNotRunTheCommandBody(t *testing.T) {
	ran := false
	flags := support.DescribeFlags(func(args []string) error {
		fs := support.NewFlagSet("probe")
		fs.String("name", "x", "a name")
		fs.Bool("apply", false, "apply it")
		if err := support.ParseFlags(fs, args); err != nil {
			return err
		}
		ran = true
		return nil
	})
	if ran || len(flags) != 2 || flags[0].Name != "apply" || !flags[0].Bool || flags[1].Name != "name" || flags[1].Default != "x" {
		t.Fatalf("ran=%t flags=%+v", ran, flags)
	}
}

func TestUsageErrorReadsTheSameTextAsHelp(t *testing.T) {
	err := usageError("deps-approved approve")
	if err == nil || !strings.Contains(err.Error(), commandDocs["deps-approved approve"].Usage) {
		t.Fatalf("usage error %v must carry the catalog usage", err)
	}
}
