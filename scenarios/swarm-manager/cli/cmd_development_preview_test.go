package main

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"connectrpc.com/connect"
	clitest "github.com/vrooli/cli-core/cliapptest"
	api "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
	apiconnect "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api/apiconnect"
)

type previewService struct {
	apiconnect.UnimplementedTransitionServiceHandler
	request *api.PreviewDevelopmentRequest
}

func previewAPIServer(t *testing.T, s *previewService) {
	t.Helper()
	_, handler := apiconnect.NewTransitionServiceHandler(s)
	clitest.NewAPIServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"status":"healthy"}`))
			return
		}
		handler.ServeHTTP(w, r)
	}))
}

func (s *previewService) PreviewDevelopment(_ context.Context, req *connect.Request[api.PreviewDevelopmentRequest]) (*connect.Response[api.PreviewDevelopmentResponse], error) {
	s.request = req.Msg
	return connect.NewResponse(&api.PreviewDevelopmentResponse{ProposalDigest: "digest", GoalMessage: "DRAFT: repair local streaming", Findings: []*api.DevelopmentReviewFinding{{Code: "budget_required", Detail: "Choose a limit"}}, LaunchBlockers: []string{"Not approved"}}), nil
}

func TestDevelopmentPreviewCLIUsesReadEndpointAndRendersBlockers(t *testing.T) {
	s := &previewService{}
	previewAPIServer(t, s)
	file := filepath.Join(t.TempDir(), "proposal.json")
	if err := os.WriteFile(file, []byte(`{"scenario":"audio-tools","objective":"preserve final tail","outcomes":[{"id":"tail","criterion":"no loss","evidenceSource":"paced trace"}]}`), 0600); err != nil {
		t.Fatal(err)
	}
	app := newAppT(t)
	out := clitest.CaptureStdout(t, func() error { return app.Run([]string{"transitions", "preview-development", "--file", file}) })
	if s.request == nil || s.request.Scenario != "audio-tools" || len(s.request.Outcomes) != 1 {
		t.Fatalf("request: %#v", s.request)
	}
	for _, want := range []string{"launch ready: false", "Nothing approved or started", "budget_required", "Not approved", "DRAFT"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in %s", want, out)
		}
	}
}

func TestDevelopmentPreviewCLIRejectsUnknownFieldsBeforeRequest(t *testing.T) {
	s := &previewService{}
	previewAPIServer(t, s)
	file := filepath.Join(t.TempDir(), "proposal.json")
	if err := os.WriteFile(file, []byte(`{"scenario":"audio-tools","approve":true}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := newAppT(t).Run([]string{"transitions", "preview-development", "--file", file}); err == nil {
		t.Fatal("unknown authorization field accepted")
	}
	if s.request != nil {
		t.Fatal("invalid proposal sent to server")
	}
}

type developmentDecisionService struct {
	apiconnect.UnimplementedDevelopmentServiceHandler
	approved *api.ApproveDevelopmentRequest
}

func (s *developmentDecisionService) ApproveDevelopment(_ context.Context, req *connect.Request[api.ApproveDevelopmentRequest]) (*connect.Response[api.DevelopmentResponse], error) {
	s.approved = req.Msg
	return connect.NewResponse(&api.DevelopmentResponse{WorkItem: req.Msg.Proposal.WorkItem, WorkShape: "contract-development", Version: 8, Digest: req.Msg.ReviewedDigest, Status: "approved", LaunchBlockers: []string{"Owner budget enforcement is not qualified"}}), nil
}

// [REQ:SWM-P0-017]
func TestDevelopmentApprovalCLITransmitsReviewedRevisionWithoutLaunching(t *testing.T) {
	s := &developmentDecisionService{}
	_, handler := apiconnect.NewDevelopmentServiceHandler(s)
	clitest.NewAPIServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"status":"healthy"}`))
			return
		}
		handler.ServeHTTP(w, r)
	}))
	file := filepath.Join(t.TempDir(), "approval.json")
	if err := os.WriteFile(file, []byte(`{"proposal":{"scenario":"example","work_item":"execute/example"},"reviewed_digest":"reviewed-sha","expected_version":"7","reason":"reviewed amendment"}`), 0600); err != nil {
		t.Fatal(err)
	}
	output := clitest.CaptureStdout(t, func() error { return newAppT(t).Run([]string{"development", "approve", "--file", file}) })
	if s.approved == nil || s.approved.ExpectedVersion != 7 || s.approved.ReviewedDigest != "reviewed-sha" || s.approved.Reason != "reviewed amendment" {
		t.Fatalf("approval lost review binding: %v", s.approved)
	}
	for _, want := range []string{"No agent launched", "reviewed-sha", "Owner budget enforcement is not qualified"} {
		if !strings.Contains(output, want) {
			t.Errorf("missing %q in %s", want, output)
		}
	}
}
