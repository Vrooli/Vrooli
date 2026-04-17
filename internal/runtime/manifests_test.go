package runtime

import (
	"encoding/json"
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/safeguards"
	"github.com/vrooli/vrooli/internal/tools"
)

// TestToolManifestsReferenceRegisteredHandlers enforces the invariant that
// every tool.json "handler" field under internal/tools/ has a matching entry
// in customToolHandlers. Drift here previously produced installed binaries
// that panicked at package init; keep this test green so the CI gate
// replaces that failure mode.
func TestToolManifestsReferenceRegisteredHandlers(t *testing.T) {
	err := fs.WalkDir(tools.Manifests, ".", func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() || d.Name() != "tool.json" {
			return nil
		}
		data, readErr := fs.ReadFile(tools.Manifests, path)
		if readErr != nil {
			t.Fatalf("read %s: %v", path, readErr)
		}
		var manifest hostreqkit.ToolManifest
		if jsonErr := json.Unmarshal(data, &manifest); jsonErr != nil {
			t.Fatalf("parse %s: %v", path, jsonErr)
		}
		handler := strings.TrimSpace(manifest.Handler)
		if handler == "" {
			return nil
		}
		if _, ok := customToolHandlers[handler]; !ok {
			t.Errorf(
				"tool %q in %s references unknown handler %q; add %q to customToolHandlers in internal/runtime/registry.go or drop the handler field",
				manifest.Name, path, handler, handler,
			)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk tool manifests: %v", err)
	}
}

// TestSafeguardManifestsReferenceRegisteredHandlers is the safeguard analogue
// of TestToolManifestsReferenceRegisteredHandlers.
func TestSafeguardManifestsReferenceRegisteredHandlers(t *testing.T) {
	err := fs.WalkDir(safeguards.Manifests, ".", func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() || d.Name() != "safeguard.json" {
			return nil
		}
		data, readErr := fs.ReadFile(safeguards.Manifests, path)
		if readErr != nil {
			t.Fatalf("read %s: %v", path, readErr)
		}
		var manifest hostreqkit.SafeguardManifest
		if jsonErr := json.Unmarshal(data, &manifest); jsonErr != nil {
			t.Fatalf("parse %s: %v", path, jsonErr)
		}
		handler := strings.TrimSpace(manifest.Handler)
		if handler == "" {
			t.Errorf("safeguard %q in %s has no handler field; safeguards require a handler", manifest.Name, path)
			return nil
		}
		if _, ok := customSafeguardHandlers[handler]; !ok {
			t.Errorf(
				"safeguard %q in %s references unknown handler %q; add %q to customSafeguardHandlers in internal/runtime/registry.go or drop the handler field",
				manifest.Name, path, handler, handler,
			)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk safeguard manifests: %v", err)
	}
}

// TestLoadToolsReportsUnknownHandlerWithoutPanic confirms that a tool.json
// referencing a handler missing from customToolHandlers surfaces as a
// descriptive error from loadTools rather than a panic. This is the
// behavioral guarantee that lets rootcli.Run reach the stale-check/rebuild
// path even when the tree is temporarily inconsistent.
func TestLoadToolsReportsUnknownHandlerWithoutPanic(t *testing.T) {
	fsys := fstest.MapFS{
		"bogus/tool.json": &fstest.MapFile{
			Data: []byte(`{"name":"bogus","handler":"does-not-exist"}`),
		},
	}

	var (
		err       error
		panicked  any
		doLoad    = func() { err = loadTools(&registry{tools: map[string]hostreqkit.Handler{}}, fsys) }
		recovered = func() {
			if r := recover(); r != nil {
				panicked = r
			}
		}
	)

	func() {
		defer recovered()
		doLoad()
	}()

	if panicked != nil {
		t.Fatalf("loadTools panicked: %v", panicked)
	}
	if err == nil {
		t.Fatal("loadTools returned nil error for unknown handler")
	}
	msg := err.Error()
	if !strings.Contains(msg, "bogus") || !strings.Contains(msg, "does-not-exist") {
		t.Fatalf("loadTools error = %q; want mention of tool name and handler key", msg)
	}
}

// TestLoadSafeguardsReportsUnknownHandlerWithoutPanic is the safeguard
// counterpart of TestLoadToolsReportsUnknownHandlerWithoutPanic.
func TestLoadSafeguardsReportsUnknownHandlerWithoutPanic(t *testing.T) {
	fsys := fstest.MapFS{
		"bogus/safeguard.json": &fstest.MapFile{
			Data: []byte(`{"name":"bogus","handler":"does-not-exist"}`),
		},
	}

	var (
		err       error
		panicked  any
		doLoad    = func() { err = loadSafeguards(&registry{safeguards: map[string]hostreqkit.Handler{}}, fsys) }
		recovered = func() {
			if r := recover(); r != nil {
				panicked = r
			}
		}
	)

	func() {
		defer recovered()
		doLoad()
	}()

	if panicked != nil {
		t.Fatalf("loadSafeguards panicked: %v", panicked)
	}
	if err == nil {
		t.Fatal("loadSafeguards returned nil error for unknown handler")
	}
	msg := err.Error()
	if !strings.Contains(msg, "bogus") || !strings.Contains(msg, "does-not-exist") {
		t.Fatalf("loadSafeguards error = %q; want mention of safeguard name and handler key", msg)
	}
}

// TestLoadToolsReportsMalformedManifestWithoutPanic confirms that a
// corrupt tool.json surfaces as an error rather than a panic.
func TestLoadToolsReportsMalformedManifestWithoutPanic(t *testing.T) {
	fsys := fstest.MapFS{
		"corrupt/tool.json": &fstest.MapFile{Data: []byte("{ not json")},
	}

	var (
		err      error
		panicked any
	)
	func() {
		defer func() {
			if r := recover(); r != nil {
				panicked = r
			}
		}()
		err = loadTools(&registry{tools: map[string]hostreqkit.Handler{}}, fsys)
	}()

	if panicked != nil {
		t.Fatalf("loadTools panicked on malformed JSON: %v", panicked)
	}
	if err == nil {
		t.Fatal("loadTools returned nil error for malformed manifest")
	}
	if !strings.Contains(err.Error(), "parse tool manifest") {
		t.Fatalf("loadTools error = %q; want parse-failure message", err.Error())
	}
}
