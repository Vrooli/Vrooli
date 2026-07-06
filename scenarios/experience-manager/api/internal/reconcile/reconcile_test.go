package reconcile

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"experience-manager/internal/spec"
	testdb "experience-manager/internal/testutil/db"

	apidb "github.com/vrooli/api-core/database"
)

type fakeCapturer struct {
	snapshot Snapshot
	err      error
}

func (f fakeCapturer) CaptureAccessibility(context.Context, CaptureTarget) (Snapshot, error) {
	return f.snapshot, f.err
}

func TestDraftCalibrationEmitsExpectedMatrixFailuresOnly(t *testing.T) { // [REQ:EXPERIEN-P0-003]
	report, err := spec.ParseScenario(filepath.Join(repoRoot(t), "scenarios", "business-health"))
	if err != nil {
		t.Fatalf("ParseScenario: %v", err)
	}
	findings := Check{Capturer: fakeCapturer{snapshot: passingSnapshot()}}.Run(context.Background(), report)
	if got := len(findings); got != 8 {
		t.Fatalf("findings = %d, want 8: %+v", got, findings)
	}
	want := map[string]int{
		spec.CodeClaimFailed:   4,
		spec.CodeClaimUnproven: 4,
	}
	for _, finding := range findings {
		want[finding.Code]--
		if finding.Severity == spec.SeverityError {
			t.Fatalf("draft calibration must be advisory, got error: %+v", finding)
		}
	}
	for code, remaining := range want {
		if remaining != 0 {
			t.Fatalf("code %s remaining count = %d", code, remaining)
		}
	}
}

func TestActivePageReconcilesAgainstAccessibilitySnapshot(t *testing.T) { // [REQ:EXPERIEN-P0-003]
	report := activeReport("primary", spec.Binding{TestID: "primary-action"})
	page := report.Spec.Pages["home"]
	page.Elements = append(page.Elements, spec.Element{ID: "summary", Role: "status"})
	page.Claims = append(page.Claims,
		spec.Claim{ID: "summary-first", Type: "reading-order", Tier: "machine", Elements: []string{"summary", "primary"}, States: []string{"default"}},
	)
	page.Bindings.Elements["summary"] = spec.Binding{TestID: "summary"}
	report.Spec.Pages["home"] = page

	findings := Check{Capturer: fakeCapturer{snapshot: passingSnapshot()}}.Run(context.Background(), report)
	if len(findings) != 0 {
		t.Fatalf("expected active page to reconcile green, got %+v", findings)
	}
}

func TestActivePagePersistsPerClaimEvidence(t *testing.T) { // [REQ:EXPERIEN-P0-003]
	report := activeReport("primary", spec.Binding{TestID: "primary-action"})
	db := testdb.NewSQLite(t)
	if err := apidb.EnsureSchemas(context.Background(), db, apidb.SchemaProviderFunc(Schema)); err != nil {
		t.Fatalf("EnsureSchemas: %v", err)
	}
	repo := NewSQLiteRepository(db)
	now := time.Date(2026, 7, 5, 16, 30, 0, 0, time.UTC)

	findings := Check{
		Capturer:   fakeCapturer{snapshot: passingSnapshot()},
		Repository: repo,
		Now:        func() time.Time { return now },
	}.Run(context.Background(), report)
	if len(findings) != 0 {
		t.Fatalf("expected active page to reconcile green, got %+v", findings)
	}
	rows, err := repo.ListEvidence(context.Background(), EvidenceFilter{Scenario: "demo", PageID: "home", ClaimID: "primary-present"})
	if err != nil {
		t.Fatalf("ListEvidence: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("evidence rows = %d, want 1: %+v", len(rows), rows)
	}
	if rows[0].Verdict != "passed" || rows[0].CaptureRef != "scenario=demo,path=/" || rows[0].AXNodeJSON == "{}" {
		t.Fatalf("unexpected evidence row: %+v", rows[0])
	}
}

func TestActivePageFailsUnresolvedBinding(t *testing.T) {
	report := activeReport("primary", spec.Binding{TestID: "primary-action"})
	page := report.Spec.Pages["home"]
	page.Elements = append(page.Elements, spec.Element{ID: "missing", Role: "button"})
	page.Claims = append(page.Claims, spec.Claim{
		ID:       "missing-present",
		Type:     "element-present",
		Tier:     "machine",
		Elements: []string{"missing"},
		States:   []string{"default"},
	})
	page.Bindings.Elements["missing"] = spec.Binding{TestID: "not-rendered"}
	report.Spec.Pages["home"] = page

	findings := Check{Capturer: fakeCapturer{snapshot: passingSnapshot()}}.Run(context.Background(), report)
	if !hasCode(findings, spec.CodeBindingUnresolved) || !hasCode(findings, spec.CodeClaimFailed) {
		t.Fatalf("expected binding and claim failures, got %+v", findings)
	}
}

func TestNonDefaultStateClaimReportsSingleCaptureLimitation(t *testing.T) {
	report := activeReport("primary", spec.Binding{TestID: "primary-action"})
	page := report.Spec.Pages["home"]
	page.Claims = []spec.Claim{{
		ID:       "stale-distinct",
		Type:     "state-distinct",
		Tier:     "machine",
		Elements: []string{"primary"},
		States:   []string{"default", "stale"},
	}}
	report.Spec.Pages["home"] = page

	findings := Check{Capturer: fakeCapturer{snapshot: passingSnapshot()}}.Run(context.Background(), report)
	if len(findings) != 1 {
		t.Fatalf("findings = %d, want 1: %+v", len(findings), findings)
	}
	if findings[0].Code != spec.CodeClaimUnverifiable || findings[0].Severity != spec.SeverityWarning {
		t.Fatalf("finding = %+v, want claim_unverifiable warning", findings[0])
	}
	if !strings.Contains(findings[0].Message, "captures only the default state") {
		t.Fatalf("message = %q", findings[0].Message)
	}
}

func TestActivePageWithNoJoinedBindingsSkipsAsUnavailable(t *testing.T) {
	report := activeReport("missing", spec.Binding{TestID: "not-rendered"})
	findings := Check{Capturer: fakeCapturer{snapshot: passingSnapshot()}}.Run(context.Background(), report)
	if len(findings) != 1 || findings[0].Code != spec.CodeCaptureUnavailable || findings[0].Severity != spec.SeverityInfo {
		t.Fatalf("expected capture unavailable info finding, got %+v", findings)
	}
}

func TestCaptureUnavailableIsSkippedInfo(t *testing.T) {
	report := activeReport("primary", spec.Binding{TestID: "primary-action"})
	findings := Check{Capturer: fakeCapturer{err: ErrCaptureUnavailable}}.Run(context.Background(), report)
	if len(findings) != 1 || findings[0].Code != spec.CodeCaptureUnavailable || findings[0].Severity != spec.SeverityInfo {
		t.Fatalf("expected capture unavailable info finding, got %+v", findings)
	}
}

func TestBASCapturerSendsResolvedTargetURL(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/browser_automation_studio.v1.capture.CaptureService/Capture", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			URL                 string `json:"url"`
			InlineAccessibility bool   `json:"inlineAccessibility"`
			InteractionFlowJSON string `json:"interactionFlowJson"`
			WaitFor             struct {
				TimeoutMs string `json:"timeoutMs"`
			} `json:"waitFor"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("Decode request: %v", err)
		}
		if got := req.URL; got != "http://localhost:21233/" {
			t.Fatalf("captured url = %q, want UI URL", got)
		}
		if !req.InlineAccessibility {
			t.Fatal("inlineAccessibility = false, want true")
		}
		if got := req.WaitFor.TimeoutMs; got != "3000" {
			t.Fatalf("waitFor.timeoutMs = %q, want string timeout", got)
		}
		if req.InteractionFlowJSON == "" || !strings.Contains(req.InteractionFlowJSON, `"duration_ms":3000`) {
			t.Fatalf("interactionFlowJson = %q, want settle wait flow", req.InteractionFlowJSON)
		}
		snapshot, err := json.Marshal(passingSnapshot())
		if err != nil {
			t.Fatalf("Marshal snapshot: %v", err)
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"accessibilityJson": string(snapshot)})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	capturer := BASCapturer{
		Resolve: func(context.Context) (string, error) {
			return server.URL, nil
		},
		ResolveTarget: func(_ context.Context, target CaptureTarget) (string, error) {
			return "http://localhost:21233" + target.Route, nil
		},
		HTTPClient: server.Client(),
	}
	if _, err := capturer.CaptureAccessibility(context.Background(), CaptureTarget{
		Scenario: "web-console",
		Route:    "/",
		PageID:   "workspace",
		SettleMs: defaultSettleMs,
	}); err != nil {
		t.Fatalf("CaptureAccessibility: %v", err)
	}
}

func activeReport(elementID string, binding spec.Binding) spec.Report {
	return spec.Report{
		Scenario: "demo",
		Spec: &spec.ScenarioSpec{
			Index: spec.IndexDocument{Pages: []spec.DocumentRef{{ID: "home", Status: "active"}}},
			Pages: map[string]spec.PageDocument{"home": {
				Page:     spec.PageIdentity{ID: "home", Routes: []string{"/"}},
				Elements: []spec.Element{{ID: elementID, Role: "button"}},
				Claims: []spec.Claim{{
					ID:       elementID + "-present",
					Type:     "element-present",
					Tier:     "machine",
					Elements: []string{elementID},
					States:   []string{"default"},
				}},
				Bindings: spec.Bindings{Elements: map[string]spec.Binding{elementID: binding}},
			}},
		},
	}
}

func passingSnapshot() Snapshot {
	return Snapshot{
		Contract: snapshotContract,
		Root: AXNode{Role: "WebArea", Children: []AXNode{
			{Role: "status", DOM: DOMNode{TestID: "summary"}},
			{Role: "button", States: []string{"focusable"}, DOM: DOMNode{TestID: "primary-action"}},
		}},
	}
}

func hasCode(findings []spec.Finding, code string) bool {
	for _, finding := range findings {
		if finding.Code == code {
			return true
		}
	}
	return false
}

func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "VISION.md")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("repo root not found")
		}
		dir = parent
	}
}
