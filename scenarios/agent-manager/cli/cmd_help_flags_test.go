package main

import (
	"strings"
	"testing"

	"agent-manager/cli/domains"
)

// SKILL.md §10: `--help` on run create, run list and workflow start must list
// the flags their handlers parse.
func TestHelpListsTheFlagsHandlersParse(t *testing.T) {
	app := &App{}
	help := map[string]string{}
	for _, command := range app.runCommands() {
		help["run "+command.Name] = command.HelpText
	}
	for _, group := range domains.SubcommandGroups(app.dependencies()) {
		if group.Name != "workflow" {
			continue
		}
		for _, command := range group.Subcommands {
			help["workflow "+command.Name] = command.HelpText
		}
	}
	for command, flags := range map[string][]string{
		"run list":       {"--limit", "--status", "--tag-prefix"},
		"run create":     {"--task-id", "--profile-id", "--until"},
		"workflow start": {"--owner", "--input-file", "--idempotency-key"},
	} {
		for _, flag := range flags {
			if !strings.Contains(help[command], flag) {
				t.Errorf("%s help must list %s:\n%s", command, flag, help[command])
			}
		}
	}
}
