package actions

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"prompt-manager/store"
)

// TestManifestResolverCertaintyMatrix exercises the certainty derivation
// table from the cli-manifest plan §8:
//
//	| Manifest state                                | Action behavior                              |
//	| Missing                                       | Allowed; "unvalidated"                       |
//	| Schema-invalid                                | Rejected; error                              |
//	| run_eligible: false                           | Rejected; error                              |
//	| effect: read + run_eligible: true             | CertaintyCommand; auto-run permitted         |
//	| effect: write + run_eligible: true            | CertaintyCommand; auto-run permitted         |
//	| effect: destructive (any)                     | CertaintyCommand + requires_confirmation     |
func TestManifestResolverCertaintyMatrix(t *testing.T) {
	cases := []struct {
		name                 string
		manifest             string // empty → no manifest file written
		argv                 []string
		wantCertainty        CommandCertainty
		wantEffect           CommandEffect
		wantRunnable         bool
		wantRequiresConfirm  bool
		wantUnvalidated      bool
		wantFailedOwnership  bool
		wantWarningOwnership bool
	}{
		{
			name: "missing manifest → unvalidated owner-only",
			argv: []string{"sample-scenario", "widgets", "list"},
			// manifest not written
			wantCertainty:        CertaintyOwnerOnly,
			wantRunnable:         true,
			wantUnvalidated:      true,
			wantWarningOwnership: true,
		},
		{
			name:          "read + run_eligible → CertaintyCommand",
			manifest:      manifestWith("read", true, nil),
			argv:          []string{"sample-scenario", "widgets", "list"},
			wantCertainty: CertaintyCommand,
			wantEffect:    EffectRead,
			wantRunnable:  true,
		},
		{
			name:          "write + run_eligible → CertaintyCommand",
			manifest:      manifestWith("write", true, nil),
			argv:          []string{"sample-scenario", "widgets", "list"},
			wantCertainty: CertaintyCommand,
			wantEffect:    EffectWrite,
			wantRunnable:  true,
		},
		{
			name:                "destructive defaults to requires_confirmation",
			manifest:            manifestWith("destructive", true, nil),
			argv:                []string{"sample-scenario", "widgets", "list"},
			wantCertainty:       CertaintyCommand,
			wantEffect:          EffectDestructive,
			wantRunnable:        true,
			wantRequiresConfirm: true,
		},
		{
			name:                "explicit requires_confirmation:false overrides destructive default",
			manifest:            manifestWith("destructive", true, ptr(false)),
			argv:                []string{"sample-scenario", "widgets", "list"},
			wantCertainty:       CertaintyCommand,
			wantEffect:          EffectDestructive,
			wantRunnable:        true,
			wantRequiresConfirm: false,
		},
		{
			name:                "run_eligible:false rejects",
			manifest:            manifestWith("read", false, nil),
			argv:                []string{"sample-scenario", "widgets", "list"},
			wantCertainty:       CertaintyNone,
			wantFailedOwnership: true,
		},
		{
			name:                "schema-invalid manifest rejects",
			manifest:            `{ "name": "sample-scenario", "groups": [ { "name": "widgets" } ] }`, // group missing commands[]
			argv:                []string{"sample-scenario", "widgets", "list"},
			wantCertainty:       CertaintyNone,
			wantFailedOwnership: true,
		},
		{
			name:                 "manifest present but command not catalogued → unvalidated owner-only",
			manifest:             manifestWith("read", true, nil),
			argv:                 []string{"sample-scenario", "widgets", "uncatalogued"},
			wantCertainty:        CertaintyOwnerOnly,
			wantRunnable:         true,
			wantUnvalidated:      true,
			wantWarningOwnership: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := newScenarioRepo(t, "sample-scenario", tc.manifest)
			resolver := NewManifestCommandResolver(filepath.Join(repo, "scenarios", "prompt-manager", "store"))
			service := NewService(nil, resolver)

			result := service.Validate(context.Background(), validAction(func(action *store.Action) {
				action.ID = "scenario.sample.widgets.list"
				action.Status = store.StatusActive
				action.Command.Argv = tc.argv
				// Wipe inputs/permissions so unrelated checks don't fire.
				action.Inputs = nil
				// Permissions must align with the manifest's effect, otherwise
				// checkPermissionAlignment flags missing declarations and
				// shadows the certainty-matrix behavior we're asserting.
				action.Permissions = store.ActionPermissions{Destructive: tc.wantEffect == EffectDestructive}
			}))

			if result.Command == nil {
				t.Fatalf("expected command resolution, got nil; checks=%#v", result.Checks)
			}
			if result.Command.Certainty != tc.wantCertainty {
				t.Fatalf("certainty = %s, want %s; message=%s", result.Command.Certainty, tc.wantCertainty, result.Command.Message)
			}
			if tc.wantEffect != "" && result.Command.Effect != tc.wantEffect {
				t.Fatalf("effect = %s, want %s", result.Command.Effect, tc.wantEffect)
			}
			if result.Command.RequiresConfirmation != tc.wantRequiresConfirm {
				t.Fatalf("requires_confirmation = %v, want %v", result.Command.RequiresConfirmation, tc.wantRequiresConfirm)
			}
			if result.RequiresConfirmation != tc.wantRequiresConfirm {
				t.Fatalf("response.requires_confirmation = %v, want %v", result.RequiresConfirmation, tc.wantRequiresConfirm)
			}
			if result.Runnable != tc.wantRunnable {
				t.Fatalf("runnable = %v, want %v; checks=%#v", result.Runnable, tc.wantRunnable, result.Checks)
			}
			if result.Unvalidated != tc.wantUnvalidated {
				t.Fatalf("unvalidated = %v, want %v", result.Unvalidated, tc.wantUnvalidated)
			}
			if tc.wantFailedOwnership && !hasFailedCheck(result, "command_ownership") {
				t.Fatalf("expected failed command_ownership check; checks=%#v", result.Checks)
			}
			if tc.wantWarningOwnership && !hasCheckWithStatus(result, "command_ownership", CheckWarning) {
				t.Fatalf("expected warning command_ownership check; checks=%#v", result.Checks)
			}
		})
	}
}

// TestManifestResolverCachesPerScenario verifies the manifest is read once
// per scenario across multiple resolutions — prevents O(actions) disk reads
// in hot validation paths.
func TestManifestResolverCachesPerScenario(t *testing.T) {
	repo := newScenarioRepo(t, "cache-scenario", manifestWith("read", true, nil))
	resolver := NewManifestCommandResolver(filepath.Join(repo, "scenarios", "prompt-manager", "store"))

	for i := 0; i < 3; i++ {
		got, err := resolver.ResolveCommand(context.Background(), []string{"cache-scenario", "widgets", "list"})
		if err != nil {
			t.Fatalf("ResolveCommand[%d]: %v", i, err)
		}
		if got.Certainty != CertaintyCommand {
			t.Fatalf("[%d] certainty = %s, want command", i, got.Certainty)
		}
	}
	if resolver.cliManifests == nil {
		t.Fatal("expected cliManifests cache to be initialised")
	}
	cached, ok := resolver.cliManifests.manifests["cache-scenario"]
	if !ok || cached.Manifest == nil {
		t.Fatalf("expected scenario manifest cached; got %#v", cached)
	}
}

func TestManifestResolverResolvesBASScreenshotActionFixturesThroughFlatCatalog(t *testing.T) {
	schemaPath := locateRealSchema(t)
	repoRoot := filepath.Dir(filepath.Dir(filepath.Dir(schemaPath)))
	manifestPath := filepath.Join(repoRoot, "scenarios", "browser-automation-studio", "cli", "manifest.json")
	manifest, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read BAS manifest: %v", err)
	}
	repo := newScenarioRepo(t, "browser-automation-studio", string(manifest))
	resolver := NewManifestCommandResolver(filepath.Join(repo, "scenarios", "prompt-manager", "store"))

	for _, fixture := range []string{"bas.screenshot", "bas.screenshot.mobile"} {
		t.Run(fixture, func(t *testing.T) {
			actionPath := filepath.Join(repoRoot, "scenarios", "prompt-manager", "store", "actions", "packs", "core", fixture, "action.json")
			raw, err := os.ReadFile(actionPath)
			if err != nil {
				t.Fatalf("read action fixture: %v", err)
			}
			var action store.Action
			if err := json.Unmarshal(raw, &action); err != nil {
				t.Fatalf("parse action fixture: %v", err)
			}
			resolution, err := resolver.ResolveCommand(context.Background(), action.Command.Argv)
			if err != nil {
				t.Fatalf("ResolveCommand(%v): %v", action.Command.Argv, err)
			}
			if resolution.Certainty != CertaintyCommand || resolution.Effect != EffectWrite {
				t.Fatalf("unexpected resolution: %#v", resolution)
			}
			if len(resolution.CommandPath) != 1 || resolution.CommandPath[0] != "capture" {
				t.Fatalf("command path = %q, want [capture]", resolution.CommandPath)
			}
		})
	}
}

// newScenarioRepo lays out a temp repo with:
//   - scenarios/<scenario>/.vrooli/service.json (registers CLI target)
//   - scenarios/<scenario>/cli/manifest.json    (if manifestJSON != "")
//   - .vrooli/schemas/cli-manifest.schema.json  (copied from real repo)
//   - scenarios/prompt-manager/store/           (resolver storeDir anchor)
//
// Returns the repo root.
func newScenarioRepo(t *testing.T, scenario, manifestJSON string) string {
	t.Helper()
	repo := t.TempDir()
	writeJSON(t, filepath.Join(repo, "scenarios", scenario, ".vrooli", "service.json"), map[string]any{
		"cli": map[string]any{
			"enabled": true,
			"command": scenario,
			"invoke":  map[string]any{"command": scenario},
		},
	})
	if manifestJSON != "" {
		path := filepath.Join(repo, "scenarios", scenario, "cli", "manifest.json")
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir manifest dir: %v", err)
		}
		if err := os.WriteFile(path, []byte(manifestJSON), 0o644); err != nil {
			t.Fatalf("write manifest: %v", err)
		}
	}
	// Anchor the resolver: the actual store contents are not consulted by the
	// resolver, only the parent path is used to derive repoRoot.
	if err := os.MkdirAll(filepath.Join(repo, "scenarios", "prompt-manager", "store"), 0o755); err != nil {
		t.Fatalf("mkdir store: %v", err)
	}
	// Copy the canonical schema file into the temp repo so SchemaPath's
	// fallback (which lands at <repo>/.vrooli/schemas/<name>) succeeds.
	schemaSrc := locateRealSchema(t)
	schemaDst := filepath.Join(repo, ".vrooli", "schemas", cliManifestSchemaName)
	if err := os.MkdirAll(filepath.Dir(schemaDst), 0o755); err != nil {
		t.Fatalf("mkdir schemas: %v", err)
	}
	data, err := os.ReadFile(schemaSrc)
	if err != nil {
		t.Fatalf("read source schema: %v", err)
	}
	if err := os.WriteFile(schemaDst, data, 0o644); err != nil {
		t.Fatalf("write schema: %v", err)
	}
	return repo
}

func locateRealSchema(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for {
		candidate := filepath.Join(dir, ".vrooli", "schemas", cliManifestSchemaName)
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("could not locate %s starting from working dir", cliManifestSchemaName)
		}
		dir = parent
	}
}

func manifestWith(effect string, runEligible bool, requiresConfirmation *bool) string {
	governance := map[string]any{
		"effect":       effect,
		"run_eligible": runEligible,
	}
	if requiresConfirmation != nil {
		governance["requires_confirmation"] = *requiresConfirmation
	}
	doc := map[string]any{
		"name": "sample-scenario",
		"groups": []map[string]any{{
			"name": "widgets",
			"commands": []map[string]any{{
				"name":       "list",
				"binding":    map[string]any{"kind": "connect-rpc", "service": "WidgetsService", "method": "ListWidgets"},
				"governance": governance,
			}},
		}},
	}
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		panic(err)
	}
	return string(data)
}

func ptr[T any](v T) *T { return &v }
