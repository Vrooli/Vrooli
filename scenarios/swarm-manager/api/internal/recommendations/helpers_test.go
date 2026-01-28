package recommendations

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"swarm-manager/internal/settings"
)

func TestValidateCreateRequest(t *testing.T) {
	cases := []struct {
		name    string
		req     CreateRequest
		wantErr string
	}{
		{name: "missing scenario", req: CreateRequest{Type: TypeDocs, Description: "desc", Priority: 3}, wantErr: "scenarioName is required"},
		{name: "missing description", req: CreateRequest{Scenario: "demo", Type: TypeDocs, Priority: 3}, wantErr: "description is required"},
		{name: "invalid type", req: CreateRequest{Scenario: "demo", Type: RecommendationType("nope"), Description: "desc", Priority: 3}, wantErr: "invalid recommendation type"},
		{name: "priority too low", req: CreateRequest{Scenario: "demo", Type: TypeDocs, Description: "desc", Priority: 0}, wantErr: "priority must be between 1 and 5"},
		{name: "priority too high", req: CreateRequest{Scenario: "demo", Type: TypeDocs, Description: "desc", Priority: 6}, wantErr: "priority must be between 1 and 5"},
		{name: "valid", req: CreateRequest{Scenario: "demo", Type: TypeDocs, Description: "desc", Priority: 3}, wantErr: ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateCreateRequest(tc.req)
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error %q, got nil", tc.wantErr)
			}
			if err.Error() != tc.wantErr {
				t.Fatalf("expected error %q, got %q", tc.wantErr, err.Error())
			}
		})
	}
}

func TestFilterRecommendationsSortAndFilters(t *testing.T) {
	items := []Recommendation{
		{ID: "a", Scenario: "Alpha", Type: TypeDocs, Status: StatusPending, Priority: 2, Created: "2024-01-01T00:00:00Z"},
		{ID: "b", Scenario: "Beta", Type: TypeTest, Status: StatusApproved, Priority: 1, Created: "2024-01-02T00:00:00Z"},
		{ID: "c", Scenario: "beta", Type: TypeTest, Status: StatusPending, Priority: 1, Created: "2024-01-01T00:00:00Z"},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/recommendations", nil)
	filtered := filterRecommendations(items, req)
	if len(filtered) != 3 {
		t.Fatalf("expected 3 recommendations, got %d", len(filtered))
	}
	if filtered[0].ID != "b" || filtered[1].ID != "c" || filtered[2].ID != "a" {
		t.Fatalf("unexpected sort order: %v", []string{filtered[0].ID, filtered[1].ID, filtered[2].ID})
	}

	reqStatus := httptest.NewRequest(http.MethodGet, "/api/v1/recommendations?status=pending", nil)
	pending := filterRecommendations(items, reqStatus)
	if len(pending) != 2 {
		t.Fatalf("expected 2 pending items, got %d", len(pending))
	}

	reqScenario := httptest.NewRequest(http.MethodGet, "/api/v1/recommendations?scenario=beta", nil)
	scenario := filterRecommendations(items, reqScenario)
	if len(scenario) != 2 {
		t.Fatalf("expected 2 beta items, got %d", len(scenario))
	}

	reqType := httptest.NewRequest(http.MethodGet, "/api/v1/recommendations?type=test", nil)
	typeFiltered := filterRecommendations(items, reqType)
	if len(typeFiltered) != 2 {
		t.Fatalf("expected 2 test items, got %d", len(typeFiltered))
	}
}

func TestMergeRecommendationsPreservesManualAndStatuses(t *testing.T) {
	existing := []Recommendation{
		{ID: "keep", Scenario: "Alpha", Type: TypeDocs, Status: StatusApproved, Priority: 2, Created: "old", Source: "manual"},
		{ID: "manual", Scenario: "Alpha", Type: TypeTest, Status: StatusPending, Priority: 3, Created: "manual", Source: "manual"},
		{ID: "rejected", Scenario: "Alpha", Type: TypeTest, Status: StatusRejected, Priority: 3, Created: "rej", Source: "generated"},
	}
	generated := []Recommendation{
		{ID: "keep", Scenario: "Alpha", Type: TypeDocs, Status: StatusPending, Priority: 2, Created: "new", Source: "generated"},
		{ID: "fresh", Scenario: "Alpha", Type: TypeFeature, Status: StatusPending, Priority: 1, Created: "newer", Source: "generated"},
	}

	merged := mergeRecommendations(existing, generated)
	if len(merged) != 4 {
		t.Fatalf("expected 4 recommendations, got %d", len(merged))
	}
	for _, rec := range merged {
		if rec.ID == "keep" {
			if rec.Status != StatusApproved {
				t.Fatalf("expected keep status approved, got %s", rec.Status)
			}
			if rec.Created != "old" {
				t.Fatalf("expected keep created preserved, got %s", rec.Created)
			}
			if rec.Source != "manual" {
				t.Fatalf("expected keep source manual, got %s", rec.Source)
			}
		}
	}
	if !containsID(merged, "manual") {
		t.Fatalf("expected manual recommendation to be preserved")
	}
	if !containsID(merged, "rejected") {
		t.Fatalf("expected rejected recommendation to be preserved")
	}
}

func TestApplyYoloMode(t *testing.T) {
	items := []Recommendation{
		{ID: "a", Status: StatusPending, Priority: 2},
		{ID: "b", Status: StatusPending, Priority: 3},
		{ID: "c", Status: StatusApproved, Priority: 4},
	}
	cfg := settings.DefaultSettings()
	cfg.RecommendationMode = "yolo"

	updated := applyYoloMode(items, cfg)
	if updated[0].Status != StatusPending {
		t.Fatalf("expected priority 2 to remain pending")
	}
	if updated[1].Status != StatusApproved {
		t.Fatalf("expected priority 3 to be approved")
	}
	if updated[2].Status != StatusApproved {
		t.Fatalf("expected already approved to stay approved")
	}
}

func TestStableIDDeterministic(t *testing.T) {
	first := stableID("Demo", TypeTest, "token")
	second := stableID("demo", TypeTest, "token")
	third := stableID("demo", TypeTest, "other")

	if first != second {
		t.Fatalf("expected stable id to be case-insensitive")
	}
	if first == third {
		t.Fatalf("expected different token to change id")
	}
}

func TestCountProblemsAndRequirementsSummary(t *testing.T) {
	dir := t.TempDir()
	problemsPath := filepath.Join(dir, "PROBLEMS.md")
	content := []byte("# Problems\n| TD-001 | bug |\n- TODO follow-up\n| NOTE | ignore |\n")
	if err := os.WriteFile(problemsPath, content, 0o644); err != nil {
		t.Fatalf("write problems file: %v", err)
	}

	if count := countProblems(problemsPath); count != 2 {
		t.Fatalf("expected 2 problems, got %d", count)
	}

	reqsPath := filepath.Join(dir, "summary.json")
	payload := []byte(`{"summary":{"completion_rate":65,"pass_rate":90}}`)
	if err := os.WriteFile(reqsPath, payload, 0o644); err != nil {
		t.Fatalf("write summary file: %v", err)
	}

	completion, passRate := loadRequirementsSummary(reqsPath)
	if completion != 65 || passRate != 90 {
		t.Fatalf("expected completion 65 and pass 90, got %d and %d", completion, passRate)
	}
}

func TestNormalizeAndClampPriority(t *testing.T) {
	if clampPriority(0) != 1 {
		t.Fatalf("expected clamp priority min to be 1")
	}
	if clampPriority(6) != 5 {
		t.Fatalf("expected clamp priority max to be 5")
	}
	if clampPriority(3) != 3 {
		t.Fatalf("expected clamp priority 3 to remain 3")
	}
	if normalizeScenarioPriority(0) != 3 {
		t.Fatalf("expected normalizeScenarioPriority default to 3")
	}
	if normalizeScenarioPriority(11) != 5 {
		t.Fatalf("expected normalizeScenarioPriority max to 5")
	}
	if normalizeScenarioPriority(5) != 3 {
		t.Fatalf("expected normalizeScenarioPriority for 5 to be 3")
	}
}

func TestLoadRequirementsSummaryMissing(t *testing.T) {
	completion, passRate := loadRequirementsSummary("/does/not/exist.json")
	if completion != 0 || passRate != 0 {
		t.Fatalf("expected missing summary to return 0,0")
	}
}
