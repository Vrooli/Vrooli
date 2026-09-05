package knowledgebase

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	kov1 "github.com/vrooli/vrooli/packages/proto/gen/go/knowledge-observatory/v1"
)

func fixture(t *testing.T) (*Service, func(string, string)) {
	t.Helper()
	root := t.TempDir()
	write := func(p, body string) {
		t.Helper()
		full := filepath.Join(root, filepath.FromSlash(p))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return &Service{RepoRoot: root}, write
}

// [REQ:KO-KB-001] [REQ:KO-KB-003]
func TestInspectRevisionApplicabilityAndPagination(t *testing.T) {
	s, write := fixture(t)
	write("docs/guide.md", "# Guide\nαβγ\n[Missing](missing.md)\n")
	write("docs/manifest.json", `{"sections":[{"documents":[{"path":"guide.md","knowledgeStatus":"historical","operatingSystems":["linux","windows"],"verifiedOperatingSystems":["linux"],"machineScope":"test-host","supersededBy":["docs/new.md"]}]}]}`)
	doc, err := s.Inspect(context.Background(), &kov1.InspectDocumentRequest{Path: "docs/guide.md", Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if !doc.Truncated || doc.NextOffset != 10 || doc.Metadata.AsMap()["knowledge_status"] != "historical" {
		t.Fatalf("unexpected evidence: %+v", doc)
	}
	if doc.Metadata.AsMap()["verified_operating_systems"].([]any)[0] != "linux" {
		t.Fatal("verified OS declaration lost")
	}
	next, err := s.Inspect(context.Background(), &kov1.InspectDocumentRequest{Path: doc.Path, Offset: doc.NextOffset, ExpectedSha256: doc.Sha256})
	if err != nil {
		t.Fatal(err)
	}
	if doc.Content+next.Content != "# Guide\nαβγ\n[Missing](missing.md)\n" {
		t.Fatal("pagination lost UTF-8 source content")
	}
	write("docs/guide.md", "changed")
	_, err = s.Inspect(context.Background(), &kov1.InspectDocumentRequest{Path: doc.Path, ExpectedSha256: doc.Sha256})
	if !errors.Is(err, ErrRevision) {
		t.Fatalf("concurrent change was not rejected: %v", err)
	}
}

// [REQ:KO-KB-001]
func TestReviewDuplicatesBacklinksAndBoundedScan(t *testing.T) {
	s, write := fixture(t)
	write("docs/a.md", "# Rule\nsystemctl\n[Missing](gone.md)\n")
	write("docs/b.md", "# Rule\nsystemctl\n[Missing](gone.md)\n")
	write("docs/c.md", "[Rule](a.md)\npath:docs/a.md\n")
	write("docs/plan.html", "<h1>Plan supplement</h1>")
	out, err := s.Review(context.Background(), &kov1.ReviewDocumentsRequest{Paths: []string{"docs/a.md", "docs/plan.html"}, BasePath: "docs", MaxFiles: 10})
	if err != nil {
		t.Fatal(err)
	}
	kinds := map[string]bool{}
	for _, o := range out.Observations {
		kinds[o.Kind] = true
	}
	for _, kind := range []string{"exact_duplicate", "incoming_reference", "missing_reference", "portability_review"} {
		if !kinds[kind] {
			t.Errorf("missing %s", kind)
		}
	}
	if out.Documents[1].Metadata.AsMap()["knowledge_status"] != "supplemental" {
		t.Fatal("HTML falsely authoritative")
	}
	limited, err := s.Review(context.Background(), &kov1.ReviewDocumentsRequest{Paths: []string{"docs/a.md"}, BasePath: "docs", MaxFiles: 1})
	if err != nil {
		t.Fatal(err)
	}
	if !limited.Truncated || limited.FilesChecked != 1 {
		t.Fatal("bounded scan concealed incomplete coverage")
	}
}

// [REQ:KO-KB-001] [REQ:KO-KB-003]
func TestRejectNonportableAndEscapingPaths(t *testing.T) {
	s, write := fixture(t)
	write("docs/a.md", "test")
	for _, p := range []string{"../outside.md", "docs/../../outside.md", "/etc/passwd", "C:/secret.md", `C:\secret.md`, `\\server\share\doc.md`} {
		if _, err := s.Inspect(context.Background(), &kov1.InspectDocumentRequest{Path: p}); err == nil {
			t.Errorf("accepted %q", p)
		}
	}
	outside := filepath.Join(t.TempDir(), "secret.md")
	if err := os.WriteFile(outside, []byte("private"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(s.RepoRoot, "docs", "link.md")); err != nil {
		t.Logf("symlink creation unavailable on this platform: %v", err)
		return
	}
	if _, err := s.Inspect(context.Background(), &kov1.InspectDocumentRequest{Path: "docs/link.md"}); err == nil {
		t.Fatal("symlink escaped root")
	}
}

func TestCanceledReviewAndOversizedSource(t *testing.T) {
	s, write := fixture(t)
	write("docs/a.md", "test")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := s.Review(ctx, &kov1.ReviewDocumentsRequest{Paths: []string{"docs/a.md"}, BasePath: "docs"}); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	write("docs/large.md", string(make([]byte, MaxBytes+1)))
	if _, err := s.Inspect(context.Background(), &kov1.InspectDocumentRequest{Path: "docs/large.md"}); !errors.Is(err, ErrInvalid) {
		t.Fatal(err)
	}
}
