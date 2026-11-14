package suppressions

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestParseMarker(t *testing.T) {
	cases := []struct {
		name    string
		line    string
		wantOK  bool
		id      string
		reason  string
		expires string
	}{
		{"go full", `// arch:allow cycle reason="legacy hub, tracked in #123" expires="until:2026-12-31"`, true, "cycle", "legacy hub, tracked in #123", "until:2026-12-31"},
		{"go no expires", `	// arch:allow god_domain reason="composition root by design"`, true, "god_domain", "composition root by design", ""},
		{"python hash", `# arch:allow coupling_smell reason="ok"`, true, "coupling_smell", "ok", ""},
		{"block comment", `/* arch:allow convergence_drift reason="planned" */`, true, "convergence_drift", "planned", ""},
		{"no marker", `// just a comment`, false, "", "", ""},
		{"id only (malformed)", `// arch:allow cycle`, true, "cycle", "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m, ok := ParseMarker(tc.line)
			if ok != tc.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tc.wantOK)
			}
			if !ok {
				return
			}
			if m.ID != tc.id || m.Reason != tc.reason || m.Expires != tc.expires {
				t.Fatalf("got %+v, want id=%q reason=%q expires=%q", m, tc.id, tc.reason, tc.expires)
			}
		})
	}
}

func TestMarkerValidate(t *testing.T) {
	if (Marker{ID: "cycle", Reason: "x"}).Validate() != true {
		t.Fatal("valid marker should validate")
	}
	if (Marker{ID: "cycle"}).Validate() != false {
		t.Fatal("marker without reason must be invalid")
	}
	if (Marker{Reason: "x"}).Validate() != false {
		t.Fatal("marker without id must be invalid")
	}
}

func TestMarkerIsActive(t *testing.T) {
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	if !(Marker{}).IsActive(now) {
		t.Fatal("no expiry → active")
	}
	if !(Marker{Expires: "until:2026-06-30"}).IsActive(now) {
		t.Fatal("future until → active")
	}
	if (Marker{Expires: "until:2026-05-01"}).IsActive(now) {
		t.Fatal("past until → inactive")
	}
	if !(Marker{Expires: "after:branch-merges"}).IsActive(now) {
		t.Fatal("non-until expiry is advisory → active")
	}
}

func TestFileScanner(t *testing.T) {
	dir := t.TempDir()
	write := func(rel, content string) {
		full := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("api/internal/graph/service.go", "package graph\n// arch:allow cycle reason=\"known\"\nfunc X(){}\n")
	write("node_modules/pkg/index.js", "// arch:allow cycle reason=\"should be skipped\"\n")
	write("docs/readme.md", "// arch:allow cycle reason=\"non-source skipped\"\n")

	markers, err := NewFileScanner().Scan(context.Background(), dir)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if len(markers) != 1 {
		t.Fatalf("expected 1 marker (node_modules + non-source skipped), got %d: %+v", len(markers), markers)
	}
	m := markers[0]
	if m.File != "api/internal/graph/service.go" || m.Line != 2 || m.ID != "cycle" {
		t.Fatalf("unexpected marker %+v", m)
	}
}

func TestFileWriter_InsertsAndIsIdempotent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "x.go")
	if err := os.WriteFile(path, []byte("package x\n\nfunc Y() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	w := NewFileWriter()
	m := Marker{ID: "god_domain", Reason: "intentional orchestration root"}
	if err := w.WriteMarker(path, 3, m); err != nil {
		t.Fatalf("write: %v", err)
	}
	data, _ := os.ReadFile(path)
	got := string(data)
	if want := `// arch:allow god_domain reason="intentional orchestration root"`; !contains(got, want) {
		t.Fatalf("marker not written; file=\n%s", got)
	}
	// Idempotent: second write of the same id is a no-op.
	if err := w.WriteMarker(path, 3, m); err != nil {
		t.Fatalf("second write: %v", err)
	}
	data2, _ := os.ReadFile(path)
	if countOccur(string(data2), "arch:allow god_domain") != 1 {
		t.Fatalf("marker written twice; file=\n%s", string(data2))
	}
}

func TestProvider_FiltersInactiveAndInvalid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a.go")
	content := "package a\n" +
		"// arch:allow cycle reason=\"valid active\"\n" +
		"// arch:allow stale reason=\"expired\" expires=\"until:2000-01-01\"\n" +
		"// arch:allow malformed\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	p := NewProvider(fixedLocator{dir}, NewFileScanner(), fixedClock{time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)})
	got, err := p.Active(context.Background(), "demo")
	if err != nil {
		t.Fatalf("active: %v", err)
	}
	if len(got) != 1 || got[0].ID != "cycle" {
		t.Fatalf("expected only the valid active marker, got %+v", got)
	}
}

type fixedLocator struct{ dir string }

func (l fixedLocator) Locate(string) (string, error) { return l.dir, nil }

type fixedClock struct{ t time.Time }

func (c fixedClock) Now() time.Time { return c.t }

func contains(s, sub string) bool { return len(s) >= len(sub) && indexOf(s, sub) >= 0 }
func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func countOccur(s, sub string) int {
	n, i := 0, 0
	for {
		j := indexOf(s[i:], sub)
		if j < 0 {
			return n
		}
		n++
		i += j + len(sub)
	}
}
