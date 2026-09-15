package agent_manager

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	vrun "development-toolchain-validator/internal/validation_run"

	"github.com/vrooli/api-core/discovery"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

// TestResolveGoldenRoot pins the path-resolution contract that the golden
// seeding depends on: absolute paths pass through unchanged, relative
// paths become absolute (rooted at the repo), and empty is an error.
func TestResolveGoldenRoot(t *testing.T) {
	if _, err := resolveGoldenRoot(""); err == nil {
		t.Fatalf("empty golden path should error")
	}

	abs := filepath.Join(string(filepath.Separator), "tmp", "golden")
	got, err := resolveGoldenRoot(abs)
	if err != nil {
		t.Fatalf("abs path: %v", err)
	}
	if got != abs {
		t.Fatalf("abs path = %q, want passthrough %q", got, abs)
	}

	// Relative paths resolve against the repo root. The test runs inside
	// the repo, so the cwd fallback locates it; we only assert the result
	// is absolute and ends with the requested relative path.
	rel := filepath.Join(".vrooli", "generated-goldens", "reference-react-vite")
	got, err = resolveGoldenRoot(rel)
	if err != nil {
		t.Fatalf("relative path: %v", err)
	}
	if !filepath.IsAbs(got) {
		t.Fatalf("relative path did not resolve to absolute: %q", got)
	}
	if !strings.HasSuffix(got, rel) {
		t.Fatalf("resolved %q does not end with %q", got, rel)
	}
}

// TestBuildSkillPrompt verifies the agent prompt embeds the skill body
// when present and falls back to a generic description otherwise.
func TestBuildSkillPrompt(t *testing.T) {
	withContent := buildSkillPrompt(vrun.SandboxedRunSpec{
		SkillID:     "progress",
		GoldenSlug:  "reference-react-vite",
		SkillPrompt: "Advance the operational progress log.",
	})
	if !strings.Contains(withContent, "Advance the operational progress log.") {
		t.Fatalf("prompt missing skill body: %q", withContent)
	}
	if !strings.Contains(withContent, "=== BEGIN SKILL: progress ===") {
		t.Fatalf("prompt missing skill delimiter: %q", withContent)
	}
	if !strings.Contains(withContent, "empty diff is a valid") {
		t.Fatalf("prompt missing convergence framing: %q", withContent)
	}

	fallback := buildSkillPrompt(vrun.SandboxedRunSpec{
		SkillID:    "progress",
		GoldenSlug: "reference-react-vite",
	})
	if !strings.Contains(fallback, "progress") || !strings.Contains(fallback, "manifest evaluation") {
		t.Fatalf("fallback prompt unexpected: %q", fallback)
	}
}

// TestSha256Hex pins the three properties the diff_hash fix relies on:
// identical content hashes identically (so converged skills collapse to
// one hash), different content diverges, and the empty diff hashes to a
// stable constant ("ran, produced no changes").
func TestSha256Hex(t *testing.T) {
	const diffA = "diff --git a/x b/x\n+added\n"
	const diffB = "diff --git a/y b/y\n-removed\n"

	if got := sha256Hex(diffA); got != sha256Hex(diffA) {
		t.Fatalf("same content produced different hashes")
	}
	if sha256Hex(diffA) == sha256Hex(diffB) {
		t.Fatalf("different content collided: %q vs %q", diffA, diffB)
	}

	wantEmpty := func() string {
		sum := sha256.Sum256([]byte(""))
		return hex.EncodeToString(sum[:])
	}()
	if got := sha256Hex(""); got != wantEmpty {
		t.Fatalf("empty hash = %q, want %q", got, wantEmpty)
	}
	// Guard the literal so a future hash-algorithm swap is a deliberate
	// decision, not a silent break of "same diff → same record".
	if wantEmpty != "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855" {
		t.Fatalf("empty-string sha256 changed unexpectedly: %q", wantEmpty)
	}
	if len(sha256Hex(diffA)) != 64 {
		t.Fatalf("hash is not 64 hex chars: %q", sha256Hex(diffA))
	}
}

// TestBuildSummaryHashesDiffContent proves the wiring end-to-end: the
// adapter fetches the run diff and stores sha256(content) in DiffHash
// (not the run id, the old bug), and mirrors the changed file paths.
func TestBuildSummaryHashesDiffContent(t *testing.T) {
	const runID = "run-123"
	const content = "diff --git a/api/foo.go b/api/foo.go\n+func foo() {}\n"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/runs/"+runID+"/diff") {
			resp := &apipb.GetRunDiffResponse{Diff: &domainpb.RunDiff{
				RunId:   runID,
				Content: content,
				Files: []*domainpb.FileDiff{
					{Path: "api/foo.go"},
				},
			}}
			out, err := protoJSONMarshal.Marshal(resp)
			if err != nil {
				t.Errorf("marshal diff response: %v", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(out)
			return
		}
		http.Error(w, "unexpected request: "+r.URL.Path, http.StatusNotFound)
	}))
	defer srv.Close()

	c := New(Options{Resolver: discovery.NewStaticResolver(srv.URL)})
	summary, err := c.buildSummary(context.Background(), &domainpb.Run{Id: runID})
	if err != nil {
		t.Fatalf("buildSummary: %v", err)
	}

	if summary.DiffHash != sha256Hex(content) {
		t.Fatalf("DiffHash = %q, want sha256 of content %q", summary.DiffHash, sha256Hex(content))
	}
	if summary.DiffHash == runID {
		t.Fatalf("DiffHash still equals run id (the old bug)")
	}
	if len(summary.DiffPaths) != 1 || summary.DiffPaths[0].Path != "api/foo.go" {
		t.Fatalf("DiffPaths not mirrored from diff files: %+v", summary.DiffPaths)
	}
}

// TestBuildSummaryNoDiffLeavesHashEmpty confirms a run with no retrievable
// diff leaves DiffHash empty rather than fabricating one.
func TestBuildSummaryNoDiffLeavesHashEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return an empty GetRunDiffResponse (no Diff) for any request.
		out, _ := protoJSONMarshal.Marshal(&apipb.GetRunDiffResponse{})
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(out)
	}))
	defer srv.Close()

	c := New(Options{Resolver: discovery.NewStaticResolver(srv.URL)})
	summary, err := c.buildSummary(context.Background(), &domainpb.Run{Id: "run-x"})
	if err != nil {
		t.Fatalf("buildSummary: %v", err)
	}
	if summary.DiffHash != "" {
		t.Fatalf("DiffHash = %q, want empty when no diff present", summary.DiffHash)
	}
}
