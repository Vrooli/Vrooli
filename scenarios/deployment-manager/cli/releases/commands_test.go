package releases

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"deployment-manager/cli/cmdutil"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliutil"
	releasesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/releases"
	releasesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/releases/releasesv1connect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type typedOperationService struct {
	releasesconnect.UnimplementedReleasesServiceHandler
	verifyDeep     *bool
	recoverRequest *releasesv1.RecoverReleaseRequest
}

func typedReleaseView() *releasesv1.ReleaseView {
	return &releasesv1.ReleaseView{ReleaseId: "r1", ProfileId: "p1", GitCommitHash: "abc", ReleaseVersion: "1.0.0", Channel: "stable", Status: "published"}
}

func (typedOperationService) List(context.Context, *connect.Request[releasesv1.ListReleasesRequest]) (*connect.Response[releasesv1.ListReleasesResponse], error) {
	return connect.NewResponse(&releasesv1.ListReleasesResponse{Releases: []*releasesv1.ReleaseView{typedReleaseView()}}), nil
}

func (typedOperationService) Get(context.Context, *connect.Request[releasesv1.GetReleaseRequest]) (*connect.Response[releasesv1.GetReleaseResponse], error) {
	return connect.NewResponse(&releasesv1.GetReleaseResponse{Release: typedReleaseView()}), nil
}

func (typedOperationService) Dossier(context.Context, *connect.Request[releasesv1.GetReleaseDossierRequest]) (*connect.Response[releasesv1.GetReleaseDossierResponse], error) {
	return connect.NewResponse(&releasesv1.GetReleaseDossierResponse{Dossier: &releasesv1.ReleaseDossier{
		SchemaVersion: 1,
		Release:       typedReleaseView(),
		Health:        &releasesv1.ReleaseHealth{ReleaseId: "r1", Status: "healthy"},
	}}), nil
}

func (typedOperationService) Start(context.Context, *connect.Request[releasesv1.StartReleaseRequest]) (*connect.Response[releasesv1.StartReleaseResponse], error) {
	return connect.NewResponse(&releasesv1.StartReleaseResponse{OperationId: "release-op-typed-start", ReleaseId: "r1", Status: "queued"}), nil
}

func (s typedOperationService) Reverify(_ context.Context, request *connect.Request[releasesv1.ReverifyReleaseRequest]) (*connect.Response[releasesv1.ReverifyReleaseResponse], error) {
	if s.verifyDeep != nil {
		*s.verifyDeep = request.Msg.GetDeep()
	}
	return connect.NewResponse(&releasesv1.ReverifyReleaseResponse{Release: typedReleaseView()}), nil
}

func (typedOperationService) Reconcile(context.Context, *connect.Request[releasesv1.ReconcileReleaseRequest]) (*connect.Response[releasesv1.ReconcileReleaseResponse], error) {
	return connect.NewResponse(&releasesv1.ReconcileReleaseResponse{Release: typedReleaseView()}), nil
}

func (s typedOperationService) Recover(_ context.Context, request *connect.Request[releasesv1.RecoverReleaseRequest]) (*connect.Response[releasesv1.RecoverReleaseResponse], error) {
	if s.recoverRequest != nil {
		*s.recoverRequest = *request.Msg
	}
	return connect.NewResponse(&releasesv1.RecoverReleaseResponse{ReleaseId: "r1", Receipt: &releasesv1.RecoveryReceipt{
		ReleaseId: "r1", CandidateId: "candidate-1", DestinationRevisionId: "destination-1", DeploymentId: "deployment-1",
		Action: "halt", Outcome: "halted", Health: "stopped", ExternalReceipt: "owner-receipt-1", ObservedAt: timestamppb.New(time.Unix(1, 0).UTC()),
	}}), nil
}

func (typedOperationService) GetOperation(context.Context, *connect.Request[releasesv1.GetReleaseOperationRequest]) (*connect.Response[releasesv1.GetReleaseOperationResponse], error) {
	return connect.NewResponse(&releasesv1.GetReleaseOperationResponse{Operation: &releasesv1.ReleaseOperation{
		OperationId: "release-op-typed",
		ReleaseId:   "r1",
		Status:      releasesv1.ReleaseOperationStatus_RELEASE_OPERATION_STATUS_AMBIGUOUS,
		ActiveStage: "reconcile",
		Error:       "owner uncertain",
		CreatedAt:   timestamppb.New(time.Unix(1, 0).UTC()),
		UpdatedAt:   timestamppb.New(time.Unix(2, 0).UTC()),
	}}), nil
}

func testAPIClient(base string) *cliutil.APIClient {
	return cliutil.NewAPIClient(
		cliutil.NewHTTPClient(cliutil.HTTPClientOptions{BaseOptions: cliutil.APIBaseOptions{DefaultBase: base}}),
		func() cliutil.APIBaseOptions { return cliutil.APIBaseOptions{DefaultBase: base} },
		func() string { return "" },
	)
}

func captureOutput(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w
	fn()
	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("read: %v", err)
	}
	_ = r.Close()
	return buf.String()
}

func TestRunRequiresSubcommand(t *testing.T) {
	if err := New(nil).Run(nil); err == nil {
		t.Fatalf("expected error when subcommand missing")
	}
}

func TestUnknownSubcommand(t *testing.T) {
	if err := New(nil).Run([]string{"frobnicate"}); err == nil ||
		!strings.Contains(err.Error(), "unknown releases subcommand") {
		t.Fatalf("expected unknown subcommand error, got %v", err)
	}
}

func TestListRequiresProfileID(t *testing.T) {
	if err := New(nil).Run([]string{"list"}); err == nil ||
		!strings.Contains(err.Error(), "profile ID is required") {
		t.Fatalf("expected profile ID required error, got %v", err)
	}
}

func TestHealthRendersReceiptBackedStanding(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/releases/r1/health" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		_, _ = io.WriteString(w, `{"status":"attention","publication_verified":false,"client_updates_healthy":false,"recovery_standing":"" ,"alerts":[{"code":"missing_publication_receipt","severity":"high","message":"receipt missing","next_action":"reconcile"}]}`)
	}))
	defer srv.Close()
	out := captureOutput(t, func() {
		if err := New(testAPIClient(srv.URL)).Run([]string{"health", "r1"}); err != nil {
			t.Fatalf("health: %v", err)
		}
	})
	if !strings.Contains(out, "attention") || !strings.Contains(out, "missing_publication_receipt") || !strings.Contains(out, "reconcile") {
		t.Fatalf("health output=%q", out)
	}
}

func TestStartRequiresFlags(t *testing.T) {
	cmd := New(nil)
	if err := cmd.Run([]string{"start"}); err == nil ||
		!strings.Contains(err.Error(), "profile ID is required") {
		t.Fatalf("expected profile ID required error, got %v", err)
	}
	if err := cmd.Run([]string{"start", "p1"}); err == nil ||
		!strings.Contains(err.Error(), "--commit is required") {
		t.Fatalf("expected commit required error, got %v", err)
	}
	if err := cmd.Run([]string{"start", "p1", "--commit", "abc"}); err == nil ||
		!strings.Contains(err.Error(), "--version is required") {
		t.Fatalf("expected version required error, got %v", err)
	}
	base := []string{"start", "p1", "--commit", "abc", "--version", "1.0.0"}
	for _, tc := range []struct {
		name string
		flag string
		want string
	}{
		{name: "artifact", flag: "--artifact-digest", want: "--artifact-digest is required"},
		{name: "candidate", flag: "--candidate-id", want: "--candidate-id is required"},
		{name: "destination", flag: "--destination-revision-id", want: "--destination-revision-id is required"},
		{name: "review", flag: "--readiness-review-key", want: "--readiness-review-key is required"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			args := append([]string(nil), base...)
			for _, required := range []string{"--artifact-digest sha256:artifact", "--candidate-id candidate-1", "--destination-revision-id destination-1", "--readiness-review-key review-1", "--authorization-epoch 1"} {
				if strings.HasPrefix(required, tc.flag+" ") {
					continue
				}
				args = append(args, strings.Fields(required)...)
			}
			if err := cmd.Run(args); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("expected %s, got %v", tc.want, err)
			}
		})
	}
	if err := cmd.Run(append(base, "--artifact-digest", "sha256:artifact", "--candidate-id", "candidate-1", "--destination-revision-id", "destination-1", "--readiness-review-key", "review-1")); err == nil || !strings.Contains(err.Error(), "--authorization-epoch") {
		t.Fatalf("expected authorization epoch required error, got %v", err)
	}
}

func TestOperationUsesTypedConnectServiceWhenConfigured(t *testing.T) {
	_, handler := releasesconnect.NewReleasesServiceHandler(typedOperationService{})
	srv := httptest.NewServer(handler)
	defer srv.Close()
	client := releasesconnect.NewReleasesServiceClient(srv.Client(), srv.URL)

	cmdutil.SetGlobalFormat("text")
	defer cmdutil.SetGlobalFormat("json")
	out := captureOutput(t, func() {
		if err := NewWithOperationClient(nil, client).Run([]string{"operation", "release-op-typed"}); err != nil {
			t.Fatalf("typed operation: %v", err)
		}
	})
	for _, expected := range []string{"Release operation release-op-typed status: ambiguous", "active phase: reconcile", "action: reconcile", "error: owner uncertain"} {
		if !strings.Contains(out, expected) {
			t.Fatalf("expected %q in typed operation output, got: %s", expected, out)
		}
	}
}

func TestStartUsesTypedConnectServiceWhenConfigured(t *testing.T) {
	_, handler := releasesconnect.NewReleasesServiceHandler(typedOperationService{})
	srv := httptest.NewServer(handler)
	defer srv.Close()
	client := releasesconnect.NewReleasesServiceClient(srv.Client(), srv.URL)

	cmdutil.SetGlobalFormat("text")
	defer cmdutil.SetGlobalFormat("json")
	out := captureOutput(t, func() {
		if err := NewWithOperationClient(nil, client).Run([]string{"start", "p1", "--commit", "abc", "--version", "1.0.0", "--artifact-digest", "sha256:artifact", "--candidate-id", "candidate-1", "--destination-revision-id", "destination-1", "--readiness-review-key", "review-1", "--authorization-epoch", "1"}); err != nil {
			t.Fatalf("typed start: %v", err)
		}
	})
	for _, expected := range []string{"Release operation release-op-typed-start status: queued", "deployment-manager releases operation release-op-typed-start"} {
		if !strings.Contains(out, expected) {
			t.Fatalf("expected %q in typed start output, got: %s", expected, out)
		}
	}
}

func TestReadCommandsUseTypedConnectServiceWhenConfigured(t *testing.T) {
	_, handler := releasesconnect.NewReleasesServiceHandler(typedOperationService{})
	srv := httptest.NewServer(handler)
	defer srv.Close()
	client := releasesconnect.NewReleasesServiceClient(srv.Client(), srv.URL)

	cmdutil.SetGlobalFormat("text")
	defer cmdutil.SetGlobalFormat("json")
	for _, tc := range []struct {
		args     []string
		expected []string
	}{
		{[]string{"list", "p1"}, []string{"r1", "published"}},
		{[]string{"get", "r1"}, []string{"Release: r1", "Status: published"}},
		{[]string{"dossier", "r1"}, []string{"Dossier: r1", "Health: healthy"}},
	} {
		out := captureOutput(t, func() {
			if err := NewWithOperationClient(nil, client).Run(tc.args); err != nil {
				t.Fatalf("typed %s: %v", tc.args[0], err)
			}
		})
		for _, expected := range tc.expected {
			if !strings.Contains(out, expected) {
				t.Fatalf("typed %s expected %q in output, got: %s", tc.args[0], expected, out)
			}
		}
	}
}

func TestVerifyUsesTypedConnectServiceWhenConfigured(t *testing.T) {
	var deep bool
	_, handler := releasesconnect.NewReleasesServiceHandler(typedOperationService{verifyDeep: &deep})
	srv := httptest.NewServer(handler)
	defer srv.Close()
	client := releasesconnect.NewReleasesServiceClient(srv.Client(), srv.URL)

	cmdutil.SetGlobalFormat("text")
	defer cmdutil.SetGlobalFormat("json")
	out := captureOutput(t, func() {
		if err := NewWithOperationClient(nil, client).Run([]string{"verify", "r1", "--deep"}); err != nil {
			t.Fatalf("typed verify: %v", err)
		}
	})
	if !deep {
		t.Fatal("typed verify did not preserve --deep")
	}
	for _, expected := range []string{"Release: r1", "Status: published", "Channel: stable"} {
		if !strings.Contains(out, expected) {
			t.Fatalf("expected %q in typed verify output, got: %s", expected, out)
		}
	}
}

func TestRecoveryAndReconcileUseTypedConnectServiceWhenConfigured(t *testing.T) {
	var recoverRequest releasesv1.RecoverReleaseRequest
	_, handler := releasesconnect.NewReleasesServiceHandler(typedOperationService{recoverRequest: &recoverRequest})
	srv := httptest.NewServer(handler)
	defer srv.Close()
	client := releasesconnect.NewReleasesServiceClient(srv.Client(), srv.URL)

	cmdutil.SetGlobalFormat("text")
	defer cmdutil.SetGlobalFormat("json")
	recoverOut := captureOutput(t, func() {
		if err := NewWithOperationClient(nil, client).Run([]string{"recover", "r1", "--review-key", "review-1", "--candidate-id", "candidate-1", "--destination-revision-id", "destination-1", "--confirmation", "halt r1"}); err != nil {
			t.Fatalf("typed recover: %v", err)
		}
	})
	for _, expected := range []string{"Action: halt", "Outcome: halted", "Receipt: owner-receipt-1"} {
		if !strings.Contains(recoverOut, expected) {
			t.Fatalf("typed recover expected %q in output, got: %s", expected, recoverOut)
		}
	}
	if recoverRequest.GetReleaseId() != "r1" || recoverRequest.GetReviewKey() != "review-1" || recoverRequest.GetCandidateId() != "candidate-1" || recoverRequest.GetDestinationRevisionId() != "destination-1" || recoverRequest.GetConfirmation() != "halt r1" {
		t.Fatalf("typed recovery identity = %v", &recoverRequest)
	}
	reconcileOut := captureOutput(t, func() {
		if err := NewWithOperationClient(nil, client).Run([]string{"reconcile", "r1"}); err != nil {
			t.Fatalf("typed reconcile: %v", err)
		}
	})
	if !strings.Contains(reconcileOut, "Release: r1") || !strings.Contains(reconcileOut, "Status: published") {
		t.Fatalf("typed reconcile output = %s", reconcileOut)
	}
}

func TestRecoverRequiresExactFlags(t *testing.T) {
	if err := New(nil).Run([]string{"recover", "r1"}); err == nil || !strings.Contains(err.Error(), "review-key") {
		t.Fatalf("expected recovery identity error, got %v", err)
	}
}
