package scenarioprimitives

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
)

func TestFreshnessManifestProjectsOnlyTheRequestField(t *testing.T) {
	repoRoot := filepath.Clean(filepath.Join("..", "..", ".."))
	raw, err := os.ReadFile(filepath.Join(repoRoot, "cli", "manifest.json"))
	if err != nil {
		t.Fatalf("read root CLI manifest: %v", err)
	}
	manifest, err := cliapp.ParseManifest(raw)
	if err != nil {
		t.Fatalf("parse root CLI manifest: %v", err)
	}
	group := manifest.FindGroup("scenario")
	if group == nil {
		t.Fatal("root CLI manifest is missing the scenario group")
	}
	var command *cliapp.ManifestCommand
	for i := range group.Commands {
		if group.Commands[i].Name == "freshness" {
			command = &group.Commands[i]
			break
		}
	}
	if command == nil {
		t.Fatal("root CLI manifest is missing scenario freshness")
	}

	schema, err := cliapp.ManifestArgs(*command)
	if err != nil {
		t.Fatalf("read freshness argument schema: %v", err)
	}
	request := (&cliv1.GetScenarioFreshnessRequest{}).ProtoReflect().Descriptor()
	resolved := map[string]string{}
	for _, positional := range schema.Positionals {
		if positional.LocalOnly {
			t.Errorf("required scenario identity %q must be projected to the RPC", positional.Name)
			continue
		}
		field, resolveErr := cliapp.ResolveArgField(request, positional.Name, schema)
		if resolveErr != nil {
			t.Errorf("resolve freshness positional %q: %v", positional.Name, resolveErr)
			continue
		}
		resolved[positional.Name] = string(field.Path[len(field.Path)-1].Name())
	}
	for _, flag := range schema.Flags {
		if !flag.LocalOnly {
			field, resolveErr := cliapp.ResolveArgField(request, flag.Name, schema)
			if resolveErr != nil {
				t.Errorf("resolve freshness flag %q: %v", flag.Name, resolveErr)
				continue
			}
			resolved[flag.Name] = string(field.Path[len(field.Path)-1].Name())
		}
	}

	if len(resolved) != 1 {
		t.Errorf("resolved request fields = %v, want exactly {name:name}", resolved)
	}
	if got := resolved["name"]; got != "name" {
		t.Errorf("projected scenario identity maps to %q, want name", got)
	}
	for _, localFlag := range []string{"explain", "inputs", "path"} {
		found := false
		for _, flag := range schema.Flags {
			if flag.Name == localFlag {
				found = true
				if !flag.LocalOnly {
					t.Errorf("%q must remain a CLI flag but must not be projected into GetScenarioFreshnessRequest", localFlag)
				}
			}
		}
		if !found {
			t.Errorf("local CLI flag %q was removed from the freshness command", localFlag)
		}
		if _, projected := resolved[localFlag]; projected {
			t.Errorf("local CLI flag %q was projected into the RPC request", localFlag)
		}
	}
}
