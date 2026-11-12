package retainedruntime

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func put(t *testing.T, p, body string, mode os.FileMode) {
	t.Helper()
	if e := os.MkdirAll(filepath.Dir(p), 0700); e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(p, []byte(body), mode); e != nil {
		t.Fatal(e)
	}
}
func fixture(t *testing.T) (*Store, CaptureSpec) {
	t.Helper()
	root := t.TempDir()
	src := filepath.Join(root, "source")
	put(t, filepath.Join(src, "bin/api"), "native-v1", 0755)
	put(t, filepath.Join(src, "ui/server.js"), "require('./dep.js')", 0644)
	put(t, filepath.Join(src, "ui/dep.js"), "module.exports='v1'", 0644)
	put(t, filepath.Join(src, ".vrooli/runtime-release.json"), `{"version":1,"state_contract":"fixture-v1","paths":["bin","ui"],"interpreters":[]}`, 0600)
	return NewStore(filepath.Join(root, "store")), CaptureSpec{Scenario: "sample", Variant: "main", SourceDir: src, BuildIdentity: "build-1", SourceRuntimeIdentity: "observed-1", ServiceManifest: []byte(`{"lifecycle":{}}`)}
}
func capture(t *testing.T, s *Store, spec CaptureSpec) Release {
	t.Helper()
	r, e := s.Capture(context.Background(), spec)
	if e != nil {
		t.Fatal(e)
	}
	return r
}
func TestRetainsExactNativeAndNodeBytes(t *testing.T) {
	s, spec := fixture(t)
	r := capture(t, s, spec)
	put(t, filepath.Join(spec.SourceDir, "bin/api"), "native-v2", 0755)
	put(t, filepath.Join(spec.SourceDir, "ui/dep.js"), "module.exports='v2'", 0644)
	v, e := s.Verify(context.Background(), spec.Scenario, spec.Variant, r.ID)
	if e != nil {
		t.Fatal(e)
	}
	for p, want := range map[string]string{"bin/api": "native-v1", "ui/dep.js": "module.exports='v1'"} {
		b, e := os.ReadFile(filepath.Join(v.RuntimeDir, p))
		if e != nil || string(b) != want {
			t.Fatalf("%s: %s %v", p, b, e)
		}
	}
	if v.BuildIdentity != "build-1" || v.SourceRuntimeIdentity != "observed-1" || len(v.Paths) != 2 {
		t.Fatal(v)
	}
	r2 := capture(t, s, spec)
	if r2.ID == r.ID {
		t.Fatal("changed runtime reused release ID")
	}
}
func TestSymlinkClosure(t *testing.T) {
	for _, target := range []string{"dep.js", "../../outside", "/tmp/outside", "missing.js"} {
		t.Run(target, func(t *testing.T) {
			s, spec := fixture(t)
			if e := os.Symlink(target, filepath.Join(spec.SourceDir, "ui/link.js")); e != nil {
				t.Fatal(e)
			}
			r, e := s.Capture(context.Background(), spec)
			if target == "dep.js" {
				if e != nil {
					t.Fatal(e)
				}
				if _, e = s.Verify(context.Background(), spec.Scenario, spec.Variant, r.ID); e != nil {
					t.Fatal(e)
				}
			} else if e == nil {
				t.Fatal("accepted incomplete or escaping symlink")
			}
		})
	}
}
func TestRejectsMissingForbiddenAndSpecialPaths(t *testing.T) {
	for _, p := range []string{"missing", "../escape", ".git", "credentials", ".env", ".env.production", ".vrooli/runtime", "data"} {
		t.Run(p, func(t *testing.T) {
			s, spec := fixture(t)
			spec.ForbiddenPaths = []string{"data"}
			if p != "missing" && p != "../escape" {
				put(t, filepath.Join(spec.SourceDir, p), "secret", 0600)
			}
			put(t, filepath.Join(spec.SourceDir, ".vrooli/runtime-release.json"), `{"version":1,"state_contract":"fixture-v1","paths":["`+p+`"]}`, 0600)
			if _, e := s.Capture(context.Background(), spec); e == nil {
				t.Fatal("accepted forbidden closure")
			}
		})
	}
}
func TestRejectsTampering(t *testing.T) {
	for _, change := range []string{"bytes", "mode", "extra", "missing", "manifest", "root-mode"} {
		t.Run(change, func(t *testing.T) {
			s, spec := fixture(t)
			r := capture(t, s, spec)
			p := filepath.Join(r.RuntimeDir, "bin/api")
			switch change {
			case "bytes":
				put(t, p, "tampered", 0755)
			case "mode":
				if e := os.Chmod(p, 0644); e != nil {
					t.Fatal(e)
				}
			case "extra":
				put(t, filepath.Join(r.RuntimeDir, "extra"), "x", 0600)
			case "missing":
				if e := os.Remove(p); e != nil {
					t.Fatal(e)
				}
			case "manifest":
				put(t, filepath.Join(filepath.Dir(r.RuntimeDir), "manifest.json"), "{}", 0600)
			case "root-mode":
				if e := os.Chmod(r.RuntimeDir, 0755); e != nil {
					t.Fatal(e)
				}
			}
			if _, e := s.Verify(context.Background(), spec.Scenario, spec.Variant, r.ID); e == nil {
				t.Fatal("tamper accepted")
			}
		})
	}
}
func TestDetectsConcurrentSourceMutation(t *testing.T) {
	s, spec := fixture(t)
	s.afterCopy = func() { put(t, filepath.Join(spec.SourceDir, "ui/dep.js"), "changed", 0644) }
	if _, e := s.Capture(context.Background(), spec); e == nil {
		t.Fatal("source mutation accepted")
	}
	scope, _ := s.scope(spec.Scenario, spec.Variant)
	entries, e := os.ReadDir(scope)
	if e != nil || len(entries) != 0 {
		t.Fatalf("partial published: %v %v", entries, e)
	}
}
func TestPinsInterpreter(t *testing.T) {
	s, spec := fixture(t)
	bin := filepath.Join(t.TempDir(), "bin")
	put(t, filepath.Join(bin, "synthetic-node"), "interpreter-v1", 0755)
	t.Setenv("PATH", bin)
	put(t, filepath.Join(spec.SourceDir, ".vrooli/runtime-release.json"), `{"version":1,"state_contract":"fixture-v1","paths":["ui"],"interpreters":["synthetic-node"]}`, 0600)
	r := capture(t, s, spec)
	if len(r.Interpreters) != 1 {
		t.Fatal(r)
	}
	put(t, filepath.Join(bin, "synthetic-node"), "interpreter-v2", 0755)
	if _, e := s.Verify(context.Background(), spec.Scenario, spec.Variant, r.ID); e == nil {
		t.Fatal("changed interpreter accepted")
	}
}
func TestSelectionCASDurabilityAndPrivateDirectories(t *testing.T) {
	s, spec := fixture(t)
	ctx := context.Background()
	r := capture(t, s, spec)
	sel, e := s.Select(ctx, spec.Scenario, spec.Variant, r.ID, "", "op-1")
	if e != nil {
		t.Fatal(e)
	}
	read, e := NewStore(s.base).ReadSelection(spec.Scenario, spec.Variant)
	if e != nil || read != sel {
		t.Fatalf("readback %v %v", read, e)
	}
	if _, e = s.Select(ctx, spec.Scenario, spec.Variant, "", "", "stale"); !errors.Is(e, ErrSelectionConflict) {
		t.Fatal(e)
	}
	if _, e = s.Select(ctx, spec.Scenario, "other", r.ID, "", "wrong"); e == nil {
		t.Fatal("cross-variant selection accepted")
	}
	if _, e = s.Select(ctx, spec.Scenario, spec.Variant, "", r.ID, "working"); e != nil {
		t.Fatal(e)
	}
	scope, _ := s.scope(spec.Scenario, spec.Variant)
	for _, p := range []string{s.base, filepath.Dir(scope), scope} {
		st, e := os.Stat(p)
		if e != nil || st.Mode().Perm() != 0700 {
			t.Fatalf("nonprivate %s", p)
		}
	}
}
func TestSelectionRejectsSymlinks(t *testing.T) {
	s, spec := fixture(t)
	r := capture(t, s, spec)
	scope, _ := s.scope(spec.Scenario, spec.Variant)
	target := filepath.Join(t.TempDir(), "pointer")
	put(t, target, `{}`, 0600)
	if e := os.Symlink(target, filepath.Join(scope, "selection.json")); e != nil {
		t.Fatal(e)
	}
	if _, e := s.Select(context.Background(), spec.Scenario, spec.Variant, r.ID, "", "op"); e == nil {
		t.Fatal("symlink pointer accepted")
	}
	if _, e := s.ReadSelection(spec.Scenario, spec.Variant); e == nil {
		t.Fatal("symlink pointer read accepted")
	}
}
func TestIdenticalCaptureIsContentAddressed(t *testing.T) {
	s, spec := fixture(t)
	a := capture(t, s, spec)
	b := capture(t, s, spec)
	if a.ID != b.ID {
		t.Fatal("non-deterministic release")
	}
}

func TestRequiresExplicitStateContract(t *testing.T) {
	s, spec := fixture(t)
	put(t, filepath.Join(spec.SourceDir, ".vrooli/runtime-release.json"), `{"version":1,"paths":["bin"],"interpreters":[]}`, 0600)
	if _, e := s.Capture(context.Background(), spec); e == nil {
		t.Fatal("missing compatibility contract accepted")
	}
}
func TestRejectsForbiddenDescendant(t *testing.T) {
	s, spec := fixture(t)
	put(t, filepath.Join(spec.SourceDir, "ui/.env"), "never-retain", 0600)
	if _, e := s.Capture(context.Background(), spec); e == nil {
		t.Fatal("secret descendant accepted")
	}
}
func TestRejectsSymlinkAncestor(t *testing.T) {
	s, spec := fixture(t)
	external := t.TempDir()
	put(t, filepath.Join(external, "out.js"), "outside", 0644)
	if e := os.Symlink(external, filepath.Join(spec.SourceDir, "alias")); e != nil {
		t.Fatal(e)
	}
	put(t, filepath.Join(spec.SourceDir, ".vrooli/runtime-release.json"), `{"version":1,"state_contract":"fixture-v1","paths":["alias/out.js"]}`, 0600)
	if _, e := s.Capture(context.Background(), spec); e == nil {
		t.Fatal("symlink ancestor accepted")
	}
}
func TestCanceledCaptureDoesNotPublish(t *testing.T) {
	s, spec := fixture(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, e := s.Capture(ctx, spec); !errors.Is(e, context.Canceled) {
		t.Fatal(e)
	}
}

func TestServiceManifestIsOnlyPersistedAsNormalizedDigest(t *testing.T) {
	s, spec := fixture(t)
	spec.ServiceManifest = []byte(`{"private":"do-not-persist-this-value","lifecycle":{}}`)
	first := capture(t, s, spec)
	spec.ServiceManifest = []byte("{\n  \"lifecycle\": {}, \"private\": \"do-not-persist-this-value\"\n}")
	second := capture(t, s, spec)
	if first.ID != second.ID || first.ServiceManifestSHA256 != second.ServiceManifestSHA256 {
		t.Fatal("semantically equivalent manifest changed content identity")
	}
	path := filepath.Join(filepath.Dir(first.RuntimeDir), "manifest.json")
	raw, e := os.ReadFile(path)
	if e != nil {
		t.Fatal(e)
	}
	if strings.Contains(string(raw), "do-not-persist-this-value") || strings.Contains(string(raw), `"service_manifest":`) {
		t.Fatal("raw configuration persisted")
	}
	if first.ServiceManifestSHA256 != digest([]byte(`{"lifecycle":{},"private":"do-not-persist-this-value"}`)) {
		t.Fatal("wrong canonical digest")
	}
}
