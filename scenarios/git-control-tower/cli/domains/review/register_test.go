package review

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	repov1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/repo"
	repoconnect "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/repo/repo_v1connect"
	reviewv1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/review"
	reviewconnect "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/review/review_v1connect"
)

type reviewStartServer struct {
	reviewconnect.UnimplementedReviewServiceHandler
	repoconnect.UnimplementedRepoServiceHandler
	repoID       int64
	startRequest *reviewv1.StartReviewRequest
}

func (s *reviewStartServer) GetActiveRepository(context.Context, *connect.Request[repov1.GetActiveRepositoryRequest]) (*connect.Response[repov1.GetActiveRepositoryResponse], error) {
	return connect.NewResponse(&repov1.GetActiveRepositoryResponse{Repo: &repov1.RepoRecord{Id: s.repoID, Name: "Vrooli"}}), nil
}

func (s *reviewStartServer) Start(_ context.Context, request *connect.Request[reviewv1.StartReviewRequest]) (*connect.Response[reviewv1.StartReviewResponse], error) {
	s.startRequest = request.Msg
	return connect.NewResponse(&reviewv1.StartReviewResponse{JobId: "review-job-1"}), nil
}

func TestRunUsesTypedReviewAdmissionAndActiveRepository(t *testing.T) {
	serverImpl := &reviewStartServer{repoID: 42}
	reviewPath, reviewHandler := reviewconnect.NewReviewServiceHandler(serverImpl)
	repoPath, repoHandler := repoconnect.NewRepoServiceHandler(serverImpl)
	mux := http.NewServeMux()
	mux.Handle(reviewPath, reviewHandler)
	mux.Handle(repoPath, repoHandler)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	previousReview := reviewClientFactory
	previousRepository := repositoryClientFactory
	reviewClientFactory = func(*cliapp.ScenarioApp) reviewconnect.ReviewServiceClient {
		return reviewconnect.NewReviewServiceClient(server.Client(), server.URL)
	}
	repositoryClientFactory = func(*cliapp.ScenarioApp) repoconnect.RepoServiceClient {
		return repoconnect.NewRepoServiceClient(server.Client(), server.URL)
	}
	t.Cleanup(func() {
		reviewClientFactory = previousReview
		repositoryClientFactory = previousRepository
	})

	if err := runRun(nil, []string{"vrooli-onboarding", "--checks=tidiness,tests", "--details=3", "--no-wait", "--json"}); err != nil {
		t.Fatalf("runRun() error = %v", err)
	}
	if serverImpl.startRequest == nil {
		t.Fatal("typed review admission was not called")
	}
	if got := serverImpl.startRequest.GetRepositoryId(); got != 42 {
		t.Fatalf("repository id = %d, want 42", got)
	}
	if got := serverImpl.startRequest.GetScenarioName(); got != "vrooli-onboarding" {
		t.Fatalf("scenario = %q", got)
	}
	if got := serverImpl.startRequest.GetDetails(); got != 3 {
		t.Fatalf("details = %d, want 3", got)
	}
	if got := serverImpl.startRequest.GetChecks(); len(got) != 2 || got[0] != "tidiness" || got[1] != "tests" {
		t.Fatalf("checks = %v", got)
	}
}

func TestRenderSummaryTriageByReadinessAndDimensions(t *testing.T) {
	resp := &summaryResponse{
		ScenarioName: "git-control-tower",
		Readiness:    "red",
		Dimensions: dimensions{
			Standards: &standardsDimension{
				Available:          true,
				BlockingViolations: 1,
				Warnings:           2,
				TotalViolations:    3,
				TopViolations: []standardsViolationDetail{{
					FilePath:       "Makefile",
					LineNumber:     12,
					Title:          "Missing lifecycle target",
					Severity:       "critical",
					Recommendation: "Add the standard target.",
				}},
			},
			Tests: &testsDimension{
				Available:   true,
				Passed:      false,
				Total:       4,
				FailedCount: 1,
				Failures: []testFailure{{
					Phase:          "smoke",
					Error:          "UI did not start",
					Classification: "environment",
					Remediation:    "Run make start before smoke.",
				}},
			},
			CodeQuality: &codeQualityDimension{
				Available: true,
				Score:     52,
				Stale:     true,
				TopIssues: []codeQualityIssue{{Category: "lint", Count: 2}},
			},
			Visual: &visualDimension{
				Available:       true,
				ScreenshotCount: 0,
			},
			Provenance: &provenanceDimension{
				Available:     true,
				TracedFiles:   8,
				UntracedFiles: []string{"ui/src/App.tsx"},
			},
		},
	}

	var out bytes.Buffer
	if err := renderSummary(&out, resp); err != nil {
		t.Fatalf("renderSummary() failed: %v", err)
	}

	rendered := out.String()
	for _, want := range []string{
		"Scenario: git-control-tower",
		"Readiness: RED (not ready)",
		"Standards: 1 blocking, 2 warnings (3 total)",
		"Makefile:12  Missing lifecycle target (critical)",
		"Tests: 1 of 4 failed",
		"smoke: UI did not start (classification: environment)",
		"Code quality: 52/100 (stale)",
		"Visual: 0 screenshots",
		"Provenance: 8 traced files",
		"Provenance: 1 untraced files",
		"ui/src/App.tsx",
		"git-control-tower review run git-control-tower",
		"Address the issues above before proceeding.",
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("rendered summary missing %q:\n%s", want, rendered)
		}
	}
}

func TestRenderSummaryPassingDimensions(t *testing.T) {
	resp := &summaryResponse{
		ScenarioName: "workspace-sandbox",
		Readiness:    "green",
		Dimensions: dimensions{
			Standards: &standardsDimension{Available: true},
			Tests: &testsDimension{
				Available:   true,
				Passed:      true,
				Total:       5,
				PassedCount: 5,
				LastRun:     "2026-05-01T18:30:00Z",
			},
			CodeQuality: &codeQualityDimension{
				Available: true,
				Score:     91,
				TopIssues: []codeQualityIssue{{Category: "complexity", Count: 1}},
			},
			Visual: &visualDimension{
				Available:       true,
				ScreenshotCount: 3,
				LatestCapture: &visualCaptureMeta{
					CapturedAt: "2026-05-01T18:15:00Z",
					CommitHash: "1234567890abcdef",
				},
			},
			Provenance: &provenanceDimension{
				Available:   true,
				TracedFiles: 12,
			},
		},
	}

	var out bytes.Buffer
	if err := renderSummary(&out, resp); err != nil {
		t.Fatalf("renderSummary() failed: %v", err)
	}

	rendered := out.String()
	for _, want := range []string{
		"Readiness: GREEN (ready)",
		"Standards: 0 violations",
		"Tests: 5/5 passed (last run: 2026-05-01 18:30)",
		"Code quality: 91/100 (complexity: 1)",
		"Visual: 3 screenshots (latest: 2026-05-01 18:15, commit 1234567)",
		"Provenance: 12 traced files",
		"All checks pass; scenario appears ready for commit review.",
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("rendered summary missing %q:\n%s", want, rendered)
		}
	}
	if strings.Contains(rendered, "Needs attention") {
		t.Fatalf("green summary should not render needs-attention triage:\n%s", rendered)
	}
}

func TestReviewFormattingHelpers(t *testing.T) {
	if got := readinessLabel("yellow"); got != "YELLOW (needs attention)" {
		t.Fatalf("unexpected yellow label: %q", got)
	}
	if got := readinessLabel("unknown"); got != "UNKNOWN" {
		t.Fatalf("unexpected unknown label: %q", got)
	}
	if got := formatTimestamp("not-a-time"); got != "not-a-time" {
		t.Fatalf("invalid timestamps should pass through, got %q", got)
	}
	if got := nextStepCommands("green", "git-control-tower"); len(got) != 1 || !strings.Contains(got[0], "All checks pass") {
		t.Fatalf("unexpected green next steps: %#v", got)
	}
	if got := nextStepCommands("unknown", "git-control-tower"); got != nil {
		t.Fatalf("unknown readiness should not suggest next steps: %#v", got)
	}
}
