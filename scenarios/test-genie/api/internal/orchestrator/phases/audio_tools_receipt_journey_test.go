package phases

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	"test-genie/internal/orchestrator/browsergrant"
	"test-genie/internal/orchestrator/phases/validationprovider"
	"test-genie/internal/orchestrator/targetruntime"
	"test-genie/internal/orchestrator/workspace"

	"github.com/vrooli/api-core/demand"
	"github.com/vrooli/api-core/provenance"
	"github.com/vrooli/cli-core/cliutil"
	assessmentpkg "github.com/vrooli/maturity-go/assessment"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	scenariovalidationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1/scenariovalidationv1connect"
)

// TestAudioToolsSoakReceiptJourney joins the Test Genie-owned demand lease,
// middleware-verified request provenance, request-scoped browser grant, and
// the real validationprovider Connect client in one owner-level fake journey.
// The HTTP endpoint is a fake Audio Tools owner; no browser or real target is
// started or implied by this test.
func TestAudioToolsSoakReceiptJourney(t *testing.T) {
	const runID = "tg-audio-soak-journey"
	const agentToken = "opaque-agent-token"
	const browserGrant = "opaque-browser-grant-receipt"

	var requestHeaders http.Header
	var requestMu sync.Mutex
	assessment := &commonv1.MaturityAssessment{
		Scenario: "audio-tools", Provider: "audio-tools", Phase: "soak", Version: "1",
		Local:              &commonv1.LocalMaturityAssessment{CurrentLevel: "L1", NextLevel: "L2", Clean: true},
		FindingsBySeverity: map[string]int32{},
	}
	assessment.Presentation = assessmentpkg.BuildPhasePresentation(assessment)
	owner := &audioToolsValidationOwner{
		assessment: assessment,
		observe: func(headers http.Header, request *scenariovalidationv1.ValidateScenarioRequest) {
			requestMu.Lock()
			defer requestMu.Unlock()
			requestHeaders = headers.Clone()
			if request.GetScenario() != "audio-tools" || request.GetPhase() != "soak" {
				t.Errorf("validationprovider request target or phase did not match the Audio Tools soak")
			}
		},
	}
	path, handler := scenariovalidationconnect.NewScenarioValidationServiceHandler(owner)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	prevResolveBaseURL := validationprovider.ResolveBaseURL
	prevNewClient := validationprovider.NewClient
	validationprovider.ResolveBaseURL = func(context.Context, string) (string, error) { return server.URL, nil }
	validationprovider.NewClient = func(timeout time.Duration, baseURL string) validationprovider.Client {
		return scenariovalidationconnect.NewScenarioValidationServiceClient(&http.Client{Timeout: timeout}, baseURL)
	}
	t.Cleanup(func() {
		validationprovider.ResolveBaseURL = prevResolveBaseURL
		validationprovider.NewClient = prevNewClient
	})

	leases := &audioToolsDemandOwner{}
	manager := targetruntime.New("audio-tools", "").
		WithHome(t.TempDir()).
		WithDemandLease(leases, "test-genie:job:"+runID, runID).
		WithURLResolver(audioToolsURLResolver{url: server.URL}).
		WithProbes(func(context.Context, int) bool { return true }, func(int) bool { return true })
	lease, err := manager.EnsureRunning(context.Background(), targetruntime.Needs{API: true}, io.Discard)
	if err != nil {
		t.Fatalf("acquire Test Genie target lease: %v", err)
	}
	issued, err := lease.DemandLeaseReceipt()
	if err != nil {
		t.Fatalf("read owner-issued Test Genie lease receipt: %v", err)
	}
	verified, err := targetruntime.VerifyIssuedDemandLeaseReceipt(issued.ReceiptID, "audio-tools")
	if err != nil || verified != issued {
		t.Fatalf("exact active target lease receipt was not verified")
	}

	initiator := verifiedAudioToolsAgentContext(t, agentToken)
	initiator = browsergrant.AttachRequestReceipt(initiator, []string{browserGrant})
	runCtx := provenance.CarryVerifiedIdentity(context.Background(), initiator)
	runCtx = browsergrant.CarryVerifiedReceipt(runCtx, initiator)

	provider := validationprovider.Provider{Phase: "soak", ProviderScenario: "audio-tools", Timeout: 3 * time.Second}
	client := func(ctx context.Context, env workspace.Environment, _ io.Writer, bound validationprovider.Provider) *validationprovider.Result {
		return validationprovider.Run(ctx, bound, env.ScenarioName, env.ScenarioDir)
	}
	runner := providerRunner(provider, client)
	artifactRoot := t.TempDir()
	report := runner(runCtx, workspace.Environment{
		RunID: runID, ScenarioName: "audio-tools", ScenarioDir: artifactRoot,
		ArtifactRoot: artifactRoot, TestGenieDemandLeaseReceiptID: issued.ReceiptID,
	}, io.Discard)
	if report.Err != nil || report.FailureClassification != "" {
		t.Fatalf("Audio Tools soak validation did not complete successfully")
	}

	requestMu.Lock()
	gotHeaders := requestHeaders.Clone()
	requestMu.Unlock()
	if gotHeaders.Get(cliutil.HeaderAgentIdentityToken) != agentToken ||
		gotHeaders.Get(browsergrant.HeaderBrowserGrantReceipt) != browserGrant ||
		gotHeaders.Get("X-Test-Genie-Lease-Receipt") != issued.ReceiptID {
		t.Fatalf("actual validationprovider request omitted one or more owner-scoped receipts")
	}
	for _, values := range [][]string{
		gotHeaders.Values(cliutil.HeaderAgentIdentityToken),
		gotHeaders.Values(browsergrant.HeaderBrowserGrantReceipt),
		gotHeaders.Values("X-Test-Genie-Lease-Receipt"),
	} {
		if len(values) != 1 {
			t.Fatalf("actual validationprovider request carried a missing or duplicate receipt")
		}
	}
	if gotHeaders.Values(cliutil.HeaderAgentIdentityToken)[0] != agentToken ||
		gotHeaders.Values(browsergrant.HeaderBrowserGrantReceipt)[0] != browserGrant ||
		gotHeaders.Values("X-Test-Genie-Lease-Receipt")[0] != issued.ReceiptID {
		t.Fatalf("actual validationprovider request did not carry the exact single receipts")
	}

	if err := filepath.WalkDir(artifactRoot, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for _, secret := range []string{agentToken, browserGrant, issued.ReceiptID} {
			if strings.Contains(string(data), secret) {
				t.Errorf("request-scoped credential was persisted in Test Genie phase artifacts")
				break
			}
		}
		return nil
	}); err != nil {
		t.Fatalf("inspect Test Genie artifacts: %v", err)
	}
	if err := manager.Cleanup(context.Background(), lease, io.Discard); err != nil {
		t.Fatalf("release Test Genie target lease: %v", err)
	}
	if _, err := targetruntime.VerifyIssuedDemandLeaseReceipt(issued.ReceiptID, "audio-tools"); err == nil {
		t.Fatal("released Test Genie demand lease receipt remained verifiable")
	}
}

type audioToolsValidationOwner struct {
	scenariovalidationconnect.UnimplementedScenarioValidationServiceHandler
	assessment *commonv1.MaturityAssessment
	observe    func(http.Header, *scenariovalidationv1.ValidateScenarioRequest)
}

func (o *audioToolsValidationOwner) ValidateScenario(_ context.Context, request *connect.Request[scenariovalidationv1.ValidateScenarioRequest]) (*connect.Response[scenariovalidationv1.ValidateScenarioResponse], error) {
	if o.observe != nil {
		o.observe(request.Header(), request.Msg)
	}
	return connect.NewResponse(&scenariovalidationv1.ValidateScenarioResponse{
		Scenario: "audio-tools", Status: scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_PASSED,
		Assessment: o.assessment,
	}), nil
}

type audioToolsDemandOwner struct{}

func (*audioToolsDemandOwner) Acquire(_ context.Context, request demand.AcquireRequest) (demand.Lease, error) {
	return demand.Lease{
		LeaseID: request.LeaseID, Scenario: request.Scenario, ConsumerID: request.ConsumerID,
		RequestID: request.RequestID, Kind: request.Kind, Status: "active", ExpiresAt: time.Now().Add(5 * time.Minute),
	}, nil
}

func (*audioToolsDemandOwner) Renew(_ context.Context, leaseID string, _ time.Duration) (demand.Lease, error) {
	return demand.Lease{LeaseID: leaseID, Status: "active", ExpiresAt: time.Now().Add(5 * time.Minute)}, nil
}

func (*audioToolsDemandOwner) Release(_ context.Context, leaseID, _ string) (demand.Lease, error) {
	return demand.Lease{LeaseID: leaseID, Status: "released"}, nil
}

type audioToolsURLResolver struct{ url string }

func (r audioToolsURLResolver) ResolveScenarioURL(_ context.Context, scenario, portKey string) (string, error) {
	if scenario != "audio-tools" || portKey != "API_PORT" {
		return "", os.ErrNotExist
	}
	return r.url, nil
}

func verifiedAudioToolsAgentContext(t *testing.T, token string) context.Context {
	t.Helper()
	var got context.Context
	handler := provenance.Middleware(provenance.VerifierFunc(func(value string) (*cliutil.VerifyResult, error) {
		if value != token {
			return nil, os.ErrPermission
		}
		return &cliutil.VerifyResult{Valid: true, Claims: &cliutil.VerifiedClaims{RunID: "agent-run"}}, nil
	}))(http.HandlerFunc(func(_ http.ResponseWriter, request *http.Request) { got = request.Context() }))
	request := httptest.NewRequest(http.MethodPost, "/test-genie/run", nil)
	request.Header.Set(cliutil.HeaderAgentIdentityToken, token)
	handler.ServeHTTP(httptest.NewRecorder(), request)
	if got == nil || !provenance.FromContext(got).IsVerifiedAgent() {
		t.Fatal("Test Genie request provenance was not verified")
	}
	return got
}
