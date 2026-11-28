package main

import (
	"testing"

	"github.com/vrooli/cli-core/cliapp"
)

func TestManifestDeclaresTypedCLIProjections(t *testing.T) {
	manifest, err := cliapp.ParseManifest(manifestBytes)
	if err != nil {
		t.Fatalf("parse CLI manifest: %v", err)
	}
	command := func(groupName, commandName string) cliapp.ManifestCommand {
		t.Helper()
		for _, group := range manifest.Groups {
			if group.Name != groupName {
				continue
			}
			for _, command := range group.Commands {
				if command.Name == commandName {
					return command
				}
			}
		}
		t.Fatalf("manifest command %s %s not found", groupName, commandName)
		return cliapp.ManifestCommand{}
	}
	flag := func(groupName, commandName, name string) cliapp.ManifestFlag {
		t.Helper()
		for _, candidate := range command(groupName, commandName).Flags {
			if candidate.Name == name {
				return candidate
			}
		}
		t.Fatalf("manifest flag %s %s --%s not found", groupName, commandName, name)
		return cliapp.ManifestFlag{}
	}
	positional := func(groupName, commandName, name string) cliapp.ManifestPositional {
		t.Helper()
		for _, candidate := range command(groupName, commandName).Positionals {
			if candidate.Name == name {
				return candidate
			}
		}
		t.Fatalf("manifest positional %s %s <%s> not found", groupName, commandName, name)
		return cliapp.ManifestPositional{}
	}
	assertBind := func(g, c, name, field string) {
		t.Helper()
		got := flag(g, c, name)
		if got.Bind == nil || got.Bind.Field != field {
			t.Fatalf("%s %s --%s bind = %#v, want field %q", g, c, name, got.Bind, field)
		}
	}
	assertBind("adoptions", "tokens-sync", "candidate", "candidate_component_id")
	assertBind("adoptions", "tokens-sync", "version", "candidate_version")
	assertBind("workspaces", "assist", "page", "page_id")
	assertBind("review", "accept", "key", "idempotency_key")

	for _, tc := range []struct{ group, command, kind, name string }{
		{"sketch", "get", "positional", "target"},
		{"sketch", "propose", "flag", "intent"},
		{"sketch", "template", "flag", "remap"},
		{"workspaces", "assist", "flag", "state"},
		{"review", "accept", "flag", "candidate"},
		{"review", "acceptance-check", "flag", "candidate"},
	} {
		var localOnly bool
		if tc.kind == "positional" {
			localOnly = positional(tc.group, tc.command, tc.name).LocalOnly
		} else {
			localOnly = flag(tc.group, tc.command, tc.name).LocalOnly
		}
		if !localOnly {
			t.Fatalf("%s %s %s %s must be marked local_only for its typed request adapter", tc.group, tc.command, tc.kind, tc.name)
		}
	}

	foundCapture := false
	for _, exception := range manifest.Exceptions {
		if exception.Command == "preview capture" && exception.Class == "external_delegation" && exception.Reason != "" {
			foundCapture = true
		}
	}
	if !foundCapture {
		t.Fatal("preview capture must be declared as a documented external delegation")
	}
	omittedInventory := map[string]string{}
	for _, omission := range manifest.Omitted {
		if omission.Service == "InventoryService" {
			omittedInventory[omission.Method] = omission.Reason
		}
	}
	for _, method := range []string{"Scan", "ScanScenario"} {
		if omittedInventory[method] == "" {
			t.Fatalf("InventoryService.%s must have an explicit CLI coverage decision", method)
		}
	}
}
