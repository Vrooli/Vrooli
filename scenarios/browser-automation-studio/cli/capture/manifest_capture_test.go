package capture

import (
	"encoding/json"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
)

// TestCaptureArgSchemaCoversManifestFlags asserts that every flag the
// manifest declares for the `capture` command has a matching entry in
// captureArgSchema(). Drift between the two surfaces (manifest names a
// flag the code doesn't accept, or vice versa) is what produced the
// `--captures` / `--out-dir` bug found in the post-migration smoke
// test; this test makes that class of drift fail at test time.
//
// capture is a hand-coded local CommandGroup; its explicit `local` manifest
// binding catalogs its public contract while preserving the purpose-built
// execution path. Connect-RPC domains are manifest-dispatched via
// internal/protodispatch where this parity is guaranteed structurally.
func TestCaptureArgSchemaCoversManifestFlags(t *testing.T) {
	manifest := readBASManifest(t)

	var doc struct {
		Groups []struct {
			Name     string `json:"name"`
			Commands []struct {
				Name  string `json:"name"`
				Flags []struct {
					Name string `json:"name"`
					Bool bool   `json:"bool"`
				} `json:"flags"`
			} `json:"commands"`
		} `json:"groups"`
	}
	if err := json.Unmarshal(manifest, &doc); err != nil {
		t.Fatalf("parse manifest: %v", err)
	}

	var manifestFlags []string
	var manifestBoolFlags []string
	for _, g := range doc.Groups {
		if g.Name != "capture" {
			continue
		}
		for _, c := range g.Commands {
			if c.Name != "capture" {
				continue
			}
			for _, f := range c.Flags {
				manifestFlags = append(manifestFlags, f.Name)
				if f.Bool {
					manifestBoolFlags = append(manifestBoolFlags, f.Name)
				}
			}
		}
	}
	if len(manifestFlags) == 0 {
		t.Fatal("manifest 'capture/capture' has no flags — did the group name change?")
	}

	schema := captureArgSchema()
	codeFlags := make(map[string]cliapp.Flag, len(schema.Flags))
	for _, f := range schema.Flags {
		codeFlags[f.Name] = f
	}

	for _, name := range manifestFlags {
		if _, ok := codeFlags[name]; !ok {
			t.Errorf("manifest declares --%s but captureArgSchema() does not", name)
		}
	}
	for _, name := range manifestBoolFlags {
		if f, ok := codeFlags[name]; ok && !f.Bool {
			t.Errorf("manifest declares --%s as Bool but captureArgSchema() declares it valued", name)
		}
	}
}
