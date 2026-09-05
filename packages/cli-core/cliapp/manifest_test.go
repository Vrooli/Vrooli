package cliapp

import (
	"strings"
	"testing"
)

const validManifest = `{
  "name": "demo",
  "groups": [
    {
      "name": "notes",
      "description": "Manage notes",
      "commands": [
        {
          "name": "list",
          "description": "List notes",
          "binding": {"kind": "connect-rpc", "service": "NotesService", "method": "ListNotes"},
          "governance": {"effect": "read", "run_eligible": true}
        },
        {
          "name": "create",
          "flags": [
            {"name": "title", "required": true},
            {"name": "body"}
          ],
          "binding": {"kind": "connect-rpc", "service": "NotesService", "method": "CreateNote"},
          "governance": {"effect": "write", "run_eligible": true}
        },
        {
          "name": "get",
          "positionals": [
            {"name": "id", "required": true}
          ],
          "binding": {"kind": "connect-rpc", "service": "NotesService", "method": "GetNote"},
          "governance": {"effect": "read", "run_eligible": true}
        }
      ]
    }
  ]
}`

func okHandler(RunContext) error { return nil }

func TestLoadFromManifestSuccess(t *testing.T) {
	bindings := map[string]func(RunContext) error{
		"NotesService.ListNotes":  okHandler,
		"NotesService.CreateNote": okHandler,
		"NotesService.GetNote":    okHandler,
	}
	group, err := LoadFromManifest([]byte(validManifest), "notes", bindings)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if group.Name != "notes" || !group.NeedsAPI {
		t.Fatalf("unexpected group: %+v", group)
	}
	if len(group.Subcommands) != 3 {
		t.Fatalf("expected 3 subcommands; got %d", len(group.Subcommands))
	}
	create := findCommand(t, group, "create")
	if len(create.Args.Flags) != 2 || create.Args.Flags[0].Name != "title" || !create.Args.Flags[0].Required {
		t.Fatalf("create args malformed: %+v", create.Args)
	}
	get := findCommand(t, group, "get")
	if len(get.Args.Positionals) != 1 || !get.Args.Positionals[0].Required {
		t.Fatalf("get args malformed: %+v", get.Args)
	}
}

func TestLoadFromManifestMissingHandler(t *testing.T) {
	bindings := map[string]func(RunContext) error{
		"NotesService.ListNotes":  okHandler,
		"NotesService.CreateNote": okHandler,
	}
	_, err := LoadFromManifest([]byte(validManifest), "notes", bindings)
	if err == nil || !strings.Contains(err.Error(), "GetNote") {
		t.Fatalf("expected missing-handler error mentioning GetNote; got %v", err)
	}
}

func TestLoadFromManifestUnusedHandler(t *testing.T) {
	bindings := map[string]func(RunContext) error{
		"NotesService.ListNotes":  okHandler,
		"NotesService.CreateNote": okHandler,
		"NotesService.GetNote":    okHandler,
		"NotesService.Phantom":    okHandler,
	}
	_, err := LoadFromManifest([]byte(validManifest), "notes", bindings)
	if err == nil || !strings.Contains(err.Error(), "Phantom") {
		t.Fatalf("expected unused-handler error mentioning Phantom; got %v", err)
	}
}

func TestLoadFromManifestUnknownGroup(t *testing.T) {
	_, err := LoadFromManifest([]byte(validManifest), "missing", map[string]func(RunContext) error{})
	if err == nil || !strings.Contains(err.Error(), "missing") {
		t.Fatalf("expected unknown-group error; got %v", err)
	}
}

func TestLoadFromManifestBuildsLocalBinding(t *testing.T) {
	manifest := []byte(`{"name":"demo","groups":[{"name":"status","flat":true,"commands":[{"name":"status","description":"Show status","binding":{"kind":"local","handler":"status"},"governance":{"effect":"read","run_eligible":true}}]}]}`)
	if _, err := ParseManifest(manifest); err != nil {
		t.Fatalf("ParseManifest() error = %v", err)
	}
	called := false
	group, err := LoadFromManifest(manifest, "status", map[string]func(RunContext) error{
		"status": func(RunContext) error { called = true; return nil },
	})
	if err != nil {
		t.Fatalf("LoadFromManifest() error = %v", err)
	}
	if group.NeedsAPI || len(group.Subcommands) != 1 || group.Subcommands[0].Name != "status" {
		t.Fatalf("local group = %#v, want one API-free status command", group)
	}
	if err := group.Subcommands[0].RunCtx(NewTestRunContext(TestRunContextOptions{Schema: ArgSchema{}})); err != nil {
		t.Fatalf("local command: %v", err)
	}
	if !called {
		t.Fatal("local handler was not invoked")
	}
}

func TestParseManifestAcceptsNestedGroupsAndFindsNestedGroup(t *testing.T) {
	manifest := []byte(`{"name":"demo","groups":[{"name":"runtime","groups":[{"name":"supervisor","commands":[{"name":"status","binding":{"kind":"connect-rpc","service":"RuntimeService","method":"Status"},"governance":{"effect":"read","run_eligible":true}}]}]}]}`)
	parsed, err := ParseManifest(manifest)
	if err != nil {
		t.Fatalf("ParseManifest() error = %v", err)
	}
	group := parsed.FindGroup("supervisor")
	if group == nil || len(group.Commands) != 1 || group.Commands[0].Name != "status" {
		t.Fatalf("nested group = %#v", group)
	}
}

func TestParseManifestRejectsInvalid(t *testing.T) {
	cases := []struct {
		name, manifest, want string
	}{
		{
			name:     "missing name",
			manifest: `{"groups": [{"name":"g","commands":[{"name":"c","binding":{"kind":"connect-rpc","service":"S","method":"M"},"governance":{"effect":"read","run_eligible":true}}]}]}`,
			want:     "name is required",
		},
		{
			name:     "unknown binding kind",
			manifest: `{"name":"d","groups":[{"name":"g","commands":[{"name":"c","binding":{"kind":"rest"},"governance":{"effect":"read","run_eligible":true}}]}]}`,
			want:     "binding.kind",
		},
		{
			name:     "invalid effect",
			manifest: `{"name":"d","groups":[{"name":"g","commands":[{"name":"c","binding":{"kind":"connect-rpc","service":"S","method":"M"},"governance":{"effect":"mutate","run_eligible":true}}]}]}`,
			want:     "governance.effect",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseManifest([]byte(tc.manifest))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("expected error containing %q; got %v", tc.want, err)
			}
		})
	}
}

func findCommand(t *testing.T, g SubcommandGroup, name string) Command {
	t.Helper()
	for _, c := range g.Subcommands {
		if c.Name == name {
			return c
		}
	}
	t.Fatalf("command %q not found in group %q", name, g.Name)
	return Command{}
}
