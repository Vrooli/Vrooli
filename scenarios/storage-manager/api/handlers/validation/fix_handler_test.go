package validation

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"connectrpc.com/connect"

	"storage-manager/internal/validation"

	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
)

// writeFile lays out path under root (creating parents) for the fix fixtures.
func writeFile(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", rel, err)
	}
}

const brokenRowsFixture = `package repo

import "database/sql"

func list(db *sql.DB) error {
	rows, err := db.Query("SELECT id FROM t")
	if err != nil {
		return err
	}
	for rows.Next() {
	}
	return rows.Err()
}
`

// TestFixRPC_PreviewThenApplyIsIdempotent exercises the real autofix registry
// through the Connect handler over the FixRequest.Path seam: preview reports the
// rows-close candidate without writing, apply writes it once, and a second apply
// is a no-op.
func TestFixRPC_PreviewThenApplyIsIdempotent(t *testing.T) {
	scenarioDir := t.TempDir()
	writeFile(t, scenarioDir, "api/internal/repo/list.go", brokenRowsFixture)

	h := NewConnectHandler(Deps{Validator: stubValidator{}, MaturitySpec: loadRealSpec(t)})
	req := func() *connect.Request[scenariovalidationv1.FixRequest] {
		return connect.NewRequest(&scenariovalidationv1.FixRequest{Path: scenarioDir})
	}

	preview, err := h.PreviewFix(context.Background(), req())
	if err != nil {
		t.Fatalf("PreviewFix error = %v", err)
	}
	if got := len(preview.Msg.GetCandidates()); got != 1 {
		t.Fatalf("preview candidates = %d, want 1", got)
	}
	if preview.Msg.GetApplied() {
		t.Fatalf("preview must not report applied")
	}
	// Preview must not have written anything.
	raw, _ := os.ReadFile(filepath.Join(scenarioDir, "api/internal/repo/list.go"))
	if strings.Contains(string(raw), "defer rows.Close()") {
		t.Fatalf("preview wrote to disk")
	}

	applied, err := h.ApplyFix(context.Background(), req())
	if err != nil {
		t.Fatalf("ApplyFix error = %v", err)
	}
	if !applied.Msg.GetApplied() || len(applied.Msg.GetCandidates()) != 1 {
		t.Fatalf("apply = applied:%v candidates:%d, want applied:true 1", applied.Msg.GetApplied(), len(applied.Msg.GetCandidates()))
	}
	after, _ := os.ReadFile(filepath.Join(scenarioDir, "api/internal/repo/list.go"))
	if !strings.Contains(string(after), "defer rows.Close()") {
		t.Fatalf("apply did not insert the defer")
	}

	second, err := h.ApplyFix(context.Background(), req())
	if err != nil {
		t.Fatalf("second ApplyFix error = %v", err)
	}
	if len(second.Msg.GetCandidates()) != 0 {
		t.Fatalf("second apply candidates = %d, want 0 (idempotent)", len(second.Msg.GetCandidates()))
	}
}

// TestFixRPC_RequiresTarget proves the handler rejects a request with neither a
// scenario nor a path.
func TestFixRPC_RequiresTarget(t *testing.T) {
	h := NewConnectHandler(Deps{Validator: stubValidator{}, MaturitySpec: loadRealSpec(t)})
	_, err := h.PreviewFix(context.Background(), connect.NewRequest(&scenariovalidationv1.FixRequest{}))
	if err == nil {
		t.Fatalf("PreviewFix with no target must error")
	}
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("error code = %v, want InvalidArgument", connect.CodeOf(err))
	}
}

// TestFixRPC_AutofixAvailableStamped proves the assessment marks findings whose
// Code the registry covers as auto-fixable, so the shared AutofixableCount is
// accurate.
func TestFixRPC_AutofixAvailableStamped(t *testing.T) {
	h := NewConnectHandler(Deps{
		Validator: stubValidator{report: validation.Report{
			Scenario: "demo",
			Language: "go",
			Findings: []validation.Finding{
				{Code: "DB_ROWS_NOT_CLOSED", Severity: validation.SeverityError, Title: "rows leak"},
				{Code: "SCHEMA_NOT_IDEMPOTENT", Severity: validation.SeverityError, Title: "not idempotent"},
			},
		}},
		MaturitySpec: loadRealSpec(t),
	})
	resp, err := h.ValidateScenario(context.Background(), connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{Scenario: "demo"}))
	if err != nil {
		t.Fatalf("ValidateScenario error = %v", err)
	}
	covered, uncovered := false, false
	for _, f := range resp.Msg.GetAssessment().GetFindings() {
		switch f.GetCode() {
		case "DB_ROWS_NOT_CLOSED":
			covered = f.GetAutofixAvailable()
		case "SCHEMA_NOT_IDEMPOTENT":
			uncovered = f.GetAutofixAvailable()
		}
	}
	if !covered {
		t.Fatalf("DB_ROWS_NOT_CLOSED must be marked auto-fixable (registry covers it)")
	}
	if uncovered {
		t.Fatalf("SCHEMA_NOT_IDEMPOTENT must NOT be marked auto-fixable (no fixer)")
	}
}
