package lifecycle

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/lifecycle/retainedruntime"
	"github.com/vrooli/vrooli/internal/scenario"
)

// Compile one harmless host-native fixture, retain only its bytes, and never
// execute it. This keeps native format/architecture checks real and hermetic.
var retainedNativeFixture struct {
	once  sync.Once
	bytes []byte
	err   error
}

func retainedIntegrationNative(t *testing.T) []byte {
	t.Helper()
	retainedNativeFixture.once.Do(func() {
		dir := t.TempDir()
		src := filepath.Join(dir, "main.go")
		output := filepath.Join(dir, "native")
		if e := os.WriteFile(src, []byte("package main\nfunc main() {}\n"), 0600); e != nil {
			retainedNativeFixture.err = e
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		cmd := exec.CommandContext(ctx, filepath.Join(runtime.GOROOT(), "bin", "go"), "build", "-buildvcs=false", "-o", output, src)
		cmd.Env = append(os.Environ(), "GOWORK=off", "GOFLAGS=", "CGO_ENABLED=0", "GOOS="+runtime.GOOS, "GOARCH="+runtime.GOARCH)
		if log, e := cmd.CombinedOutput(); e != nil {
			retainedNativeFixture.err = fmt.Errorf("compile native fixture: %w: %s", e, log)
			return
		}
		retainedNativeFixture.bytes, retainedNativeFixture.err = os.ReadFile(output)
	})
	if retainedNativeFixture.err != nil {
		t.Fatal(retainedNativeFixture.err)
	}
	return retainedNativeFixture.bytes
}

func retainedIntegrationWrite(t *testing.T, path string, data []byte, mode os.FileMode) {
	t.Helper()
	if e := os.MkdirAll(filepath.Dir(path), 0700); e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(path, data, mode); e != nil {
		t.Fatal(e)
	}
}
func retainedIntegrationFixture(t *testing.T, change func(*scenario.ServiceManifest)) (*Runner, scenario.Scenario, retainedruntime.Release) {
	t.Helper()
	root := t.TempDir()
	src := filepath.Join(root, "scenarios", "retained-fixture")
	home := filepath.Join(root, "home")
	bin := filepath.Join(root, "interpreters")
	retainedIntegrationWrite(t, filepath.Join(bin, "node"), retainedIntegrationNative(t), 0755)
	t.Setenv("PATH", bin)
	m := scenario.ServiceManifest{Service: scenario.ServiceMetadata{Name: "retained-fixture"}, Components: map[string]scenario.Component{
		"api":    {Build: scenario.ComponentBuild{Kind: "go_module", Dir: "api", Output: "api/native"}, Run: scenario.ComponentRun{Argv: []string{"{{bin.api}}"}, CWD: "api"}},
		"worker": {Build: scenario.ComponentBuild{Kind: "node_bundle", Dir: "worker", Output: "worker/dist/index.js"}, Run: scenario.ComponentRun{Argv: []string{"node", "dist/index.js"}, CWD: "worker"}},
	}}
	if change != nil {
		change(&m)
	}
	raw, e := json.Marshal(m)
	if e != nil {
		t.Fatal(e)
	}
	retainedIntegrationWrite(t, filepath.Join(src, ".vrooli/service.json"), raw, 0600)
	retainedIntegrationWrite(t, filepath.Join(src, ".vrooli/runtime-release.json"), []byte(`{"version":1,"state_contract":"fixture-state-v1","paths":["api/native","worker"],"interpreters":["node"]}`), 0600)
	retainedIntegrationWrite(t, filepath.Join(src, "api/native"), retainedIntegrationNative(t), 0755)
	retainedIntegrationWrite(t, filepath.Join(src, "worker/package.json"), []byte(`{"dependencies":{}}`), 0644)
	retainedIntegrationWrite(t, filepath.Join(src, "worker/dist/index.js"), []byte("require('./dependency.js')"), 0644)
	retainedIntegrationWrite(t, filepath.Join(src, "worker/dist/dependency.js"), []byte("module.exports='retained-v1'"), 0644)
	retainedIntegrationWrite(t, filepath.Join(src, "data/api/state.db"), []byte("canonical durable state; never snapshot"), 0600)
	r := &Runner{Root: root, Home: home}
	item := scenario.Scenario{Slug: "retained-fixture", Variant: "live", Path: src, ServicePath: filepath.Join(src, ".vrooli/service.json"), Manifest: m}
	release, e := r.runtimeReleaseStore().Capture(context.Background(), retainedruntime.CaptureSpec{Scenario: item.Slug, Variant: item.Variant, SourceDir: src, BuildIdentity: "original-serving-build", SourceRuntimeIdentity: "observed-instance", ServiceManifest: raw, ForbiddenPaths: []string{"data", "logs"}})
	if e != nil {
		t.Fatal(e)
	}
	return r, item, release
}
func TestRetainedRuntimeBinding(t *testing.T) {
	t.Run("Given retained native and Node bytes When binding Then code moves and data origin stays canonical", func(t *testing.T) {
		r, item, release := retainedIntegrationFixture(t, nil)
		bound, e := r.bindRetainedRuntime(item, release)
		if e != nil {
			t.Fatal(e)
		}
		if bound.Path != item.Path || bound.ServicePath != item.ServicePath || bound.SourcePath() != release.RuntimeDir {
			t.Fatalf("canonical/source split lost: %#v", bound)
		}
		if bound.RetainedBuildIdentity != "sha256:"+release.ID || bound.RetainedReleaseID != release.ID || bound.RetainedBuildIdentity == release.BuildIdentity {
			t.Fatalf("build/release identities conflated: %#v", bound)
		}
		if got := bound.Manifest.Components["worker"].Run.Argv[0]; got != release.Interpreters[0].Path {
			t.Fatalf("interpreter not pinned: %q", got)
		}
		if item.Manifest.Components["worker"].Run.Argv[0] != "node" {
			t.Fatal("binding mutated canonical manifest")
		}
		if e = createComponentRuntimeDirectories(bound.Path, bound.Manifest.Components["api"]); e != nil {
			t.Fatal(e)
		}
		if b, e := os.ReadFile(filepath.Join(bound.Path, "data/api/state.db")); e != nil || string(b) != "canonical durable state; never snapshot" {
			t.Fatalf("canonical data changed: %s %v", b, e)
		}
		if _, e = os.Stat(filepath.Join(bound.SourcePath(), "data")); !os.IsNotExist(e) {
			t.Fatalf("data leaked into retained runtime: %v", e)
		}
	})
	t.Run("Given selected release When source artifacts mutate Then selected layout still uses captured bytes", func(t *testing.T) {
		r, item, release := retainedIntegrationFixture(t, nil)
		if _, e := r.runtimeReleaseStore().Select(context.Background(), item.Slug, item.Variant, release.ID, "", "fixture-select"); e != nil {
			t.Fatal(e)
		}
		retainedIntegrationWrite(t, filepath.Join(item.Path, "api/native"), []byte("dirty-native"), 0755)
		retainedIntegrationWrite(t, filepath.Join(item.Path, "worker/dist/dependency.js"), []byte("dirty-module"), 0644)
		bound, selected, e := r.retainedSourceLayout(item)
		if e != nil || !selected {
			t.Fatalf("selected=%v err=%v", selected, e)
		}
		for p, want := range map[string][]byte{"api/native": retainedIntegrationNative(t), "worker/dist/dependency.js": []byte("module.exports='retained-v1'")} {
			b, e := os.ReadFile(filepath.Join(bound.SourcePath(), p))
			if e != nil || !bytes.Equal(b, want) {
				t.Fatalf("retained %s changed: got %d bytes, want %d, error=%v", p, len(b), len(want), e)
			}
		}
	})
	for _, kind := range []string{"state-contract", "launch-policy", "canonical-path", "original-build"} {
		t.Run("Given "+kind+" mismatch When binding Then refuse", func(t *testing.T) {
			r, item, release := retainedIntegrationFixture(t, nil)
			switch kind {
			case "state-contract":
				retainedIntegrationWrite(t, filepath.Join(item.Path, ".vrooli/runtime-release.json"), []byte(`{"version":1,"state_contract":"incompatible-v2"}`), 0600)
			case "launch-policy":
				retainedIntegrationWrite(t, item.ServicePath, []byte(`{"service":{"name":"changed"}}`), 0600)
			case "canonical-path":
				item.Path = filepath.Join(t.TempDir(), "other")
			case "original-build":
				release.BuildIdentity = ""
			}
			if _, e := r.bindRetainedRuntime(item, release); e == nil {
				t.Fatal("incompatible release accepted")
			}
		})
	}
	t.Run("Given matching policies but external output When binding Then refuse uncaptured executable", func(t *testing.T) {
		external := filepath.Join(t.TempDir(), "external-executable")
		retainedIntegrationWrite(t, external, []byte("never execute"), 0755)
		r, item, release := retainedIntegrationFixture(t, func(m *scenario.ServiceManifest) {
			c := m.Components["api"]
			c.Build.Output = external
			m.Components["api"] = c
		})
		if _, e := r.bindRetainedRuntime(item, release); e == nil {
			t.Fatal("external executable accepted")
		}
	})

	for _, kind := range []string{"relative-data", "absolute-data", "relative-logs", "absolute-logs", "develop-hook", "inline-node", "escaping-node-entry"} {
		t.Run("Given "+kind+" When binding Then refuse mutable or uncaptured execution", func(t *testing.T) {
			r, item, release := retainedIntegrationFixture(t, func(m *scenario.ServiceManifest) {
				switch kind {
				case "relative-data":
					c := m.Components["api"]
					c.Run.DataDirs = []string{"data/api"}
					m.Components["api"] = c
				case "absolute-data":
					c := m.Components["api"]
					c.Run.DataDirs = []string{filepath.Join(t.TempDir(), "external-state")}
					m.Components["api"] = c
				case "absolute-logs":
					c := m.Components["api"]
					c.Run.LogDir = filepath.Join(t.TempDir(), "external-logs")
					m.Components["api"] = c
				case "relative-logs":
					c := m.Components["api"]
					c.Run.LogDir = "logs"
					m.Components["api"] = c
				case "develop-hook":
					m.Lifecycle.Develop.Steps = []scenario.PhaseStep{{Name: "uncaptured-hook", Exec: []string{"sh", "mutable-hook.sh"}}}
				case "inline-node":
					c := m.Components["worker"]
					c.Run.Argv = []string{"node", "-e", "inline code"}
					m.Components["worker"] = c
				case "escaping-node-entry":
					c := m.Components["worker"]
					c.Run.Argv = []string{"node", "../../outside.js"}
					m.Components["worker"] = c
				}
			})
			if _, e := r.bindRetainedRuntime(item, release); e == nil {
				t.Fatal("unsafe retained launch accepted")
			}
		})
	}

	t.Run("Given shebang wrapper named node When binding Then refuse non-native interpreter", func(t *testing.T) {
		r, item, previous := retainedIntegrationFixture(t, nil)
		retainedIntegrationWrite(t, previous.Interpreters[0].Path, []byte("#!/bin/sh\necho never-execute\n"), 0755)
		raw, e := os.ReadFile(item.ServicePath)
		if e != nil {
			t.Fatal(e)
		}
		release, e := r.runtimeReleaseStore().Capture(context.Background(), retainedruntime.CaptureSpec{Scenario: item.Slug, Variant: item.Variant, SourceDir: item.Path, BuildIdentity: "observed-build", ServiceManifest: raw, ForbiddenPaths: []string{"data", "logs"}})
		if e != nil {
			t.Fatal(e)
		}
		if _, e = r.bindRetainedRuntime(item, release); e == nil {
			t.Fatal("node wrapper bypassed native interpreter validation")
		}
	})
	t.Run("Given executable shebang output When binding Then refuse unpinned interpreter", func(t *testing.T) {
		r, item, _ := retainedIntegrationFixture(t, nil)
		retainedIntegrationWrite(t, filepath.Join(item.Path, "api/native"), []byte("#!/bin/sh\necho never-execute\n"), 0755)
		raw, e := os.ReadFile(item.ServicePath)
		if e != nil {
			t.Fatal(e)
		}
		release, e := r.runtimeReleaseStore().Capture(context.Background(), retainedruntime.CaptureSpec{Scenario: item.Slug, Variant: item.Variant, SourceDir: item.Path, BuildIdentity: "observed-build", ServiceManifest: raw, ForbiddenPaths: []string{"data", "logs"}})
		if e != nil {
			t.Fatal(e)
		}
		if _, e = r.bindRetainedRuntime(item, release); e == nil {
			t.Fatal("shebang executable bypassed pinned interpreter validation")
		}
	})
	t.Run("Given selected release tampered When resolving Then error without source fallback", func(t *testing.T) {
		r, item, release := retainedIntegrationFixture(t, nil)
		if _, e := r.runtimeReleaseStore().Select(context.Background(), item.Slug, item.Variant, release.ID, "", "select"); e != nil {
			t.Fatal(e)
		}
		retainedIntegrationWrite(t, filepath.Join(release.RuntimeDir, "api/native"), []byte("tampered"), 0755)
		if _, _, e := r.retainedSourceLayout(item); e == nil {
			t.Fatal("tampered release accepted or silently fell back")
		}
	})

	t.Run("Given previously bound descriptor When retained bytes change Then re-resolution refuses", func(t *testing.T) {
		r, item, release := retainedIntegrationFixture(t, nil)
		if _, e := r.runtimeReleaseStore().Select(context.Background(), item.Slug, item.Variant, release.ID, "", "select"); e != nil {
			t.Fatal(e)
		}
		bound, selected, e := r.retainedSourceLayout(item)
		if e != nil || !selected {
			t.Fatalf("initial bind: %v %v", selected, e)
		}
		retainedIntegrationWrite(t, filepath.Join(release.RuntimeDir, "api/native"), []byte("tampered-after-binding"), 0755)
		if _, _, e = r.retainedSourceLayout(bound); e == nil {
			t.Fatal("previously bound descriptor bypassed verification")
		}
	})
	t.Run("Given stale expected selection When changing Then conflict preserves original pointer", func(t *testing.T) {
		r, item, release := retainedIntegrationFixture(t, nil)
		s := r.runtimeReleaseStore()
		if _, e := s.Select(context.Background(), item.Slug, item.Variant, release.ID, "", "first"); e != nil {
			t.Fatal(e)
		}
		if _, e := s.Select(context.Background(), item.Slug, item.Variant, "", "", "stale"); !errors.Is(e, retainedruntime.ErrSelectionConflict) {
			t.Fatalf("want conflict, got %v", e)
		}
		sel, e := s.ReadSelection(item.Slug, item.Variant)
		if e != nil || sel.ReleaseID != release.ID {
			t.Fatalf("pointer changed: %#v %v", sel, e)
		}
	})
	t.Run("Given no durable lifecycle operation When requesting selection Then refuse", func(t *testing.T) {
		r, item, release := retainedIntegrationFixture(t, nil)
		e := r.selectRequestedRuntime(item.Slug, StartOptions{Variant: item.Variant, RetainedReleaseID: release.ID, ExpectedRetainedReleaseID: "source"})
		if e == nil || !strings.Contains(e.Error(), "durable start-operation") {
			t.Fatalf("missing operation accepted: %v", e)
		}
	})
}

func TestRetainedNodeEntryLiteral(t *testing.T) {
	for _, entry := range []string{"/dist/index.js", "${ENTRY}", "{{entry}}"} {
		t.Run("Given an absolute or templated entry When binding Then reject "+entry, func(t *testing.T) {
			r, item, release := retainedIntegrationFixture(t, func(m *scenario.ServiceManifest) {
				c := m.Components["worker"]
				c.Run.Argv[1] = entry
				m.Components["worker"] = c
			})
			if _, err := r.bindRetainedRuntime(item, release); err == nil {
				t.Fatal("unverified Node entry accepted")
			}
		})
	}
}

func TestRetainedDescriptorReturnsToSource(t *testing.T) {
	r, item, release := retainedIntegrationFixture(t, nil)
	s := r.runtimeReleaseStore()
	if _, err := s.Select(context.Background(), item.Slug, item.Variant, release.ID, "", "select"); err != nil {
		t.Fatal(err)
	}
	bound, err := r.withSourceLayout(item)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.Select(context.Background(), item.Slug, item.Variant, "", release.ID, "return-source"); err != nil {
		t.Fatal(err)
	}
	source, err := r.withSourceLayout(bound)
	if err != nil {
		t.Fatal(err)
	}
	if source.SourcePath() != item.Path || source.RetainedReleaseID != "" || source.Manifest.Components["worker"].Run.Argv[0] != "node" {
		t.Fatalf("stale retained descriptor after return to source: %#v", source)
	}
}
