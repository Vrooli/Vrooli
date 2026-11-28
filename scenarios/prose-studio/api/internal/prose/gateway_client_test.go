package prose

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"connectrpc.com/connect"
	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
	inferenceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference/inference_v1connect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
	_ "modernc.org/sqlite"
)

type privacyCaptureInference struct {
	inferenceconnect.UnimplementedInferenceServiceHandler
	request *inferencev1.RunRequest
}

type privateCaptureInference struct {
	inferenceconnect.UnimplementedInferenceServiceHandler
	request *inferencev1.RunRequest
}

func (f *privateCaptureInference) Run(_ context.Context, request *connect.Request[inferencev1.RunRequest]) (*connect.Response[inferencev1.RunResponse], error) {
	f.request = request.Msg
	return connect.NewResponse(&inferencev1.RunResponse{ValueJson: `"synthetic generated output"`, Provider: "fixture-local", Model: "fixture-model"}), nil
}

func TestPrivateEphemeralGenerationUsesLocalGatewayWithoutProseRecords(t *testing.T) {
	fixture := &privateCaptureInference{}
	path, handler := inferenceconnect.NewInferenceServiceHandler(fixture)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	gatewayServer := httptest.NewServer(mux)
	defer gatewayServer.Close()

	db, err := sql.Open("sqlite", "file:prose-private-http-test?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	if err := EnsureSchema(db); err != nil {
		t.Fatal(err)
	}
	s := NewWithGateway(db, HTTPGateway{BaseURL: gatewayServer.URL})
	if _, err := s.CreateProfile(context.Background(), Profile{Key: "private-local", Locality: "local_only", Sampler: Sampler{Kind: samplerDirect, K: 1}, Budget: Budget{MaxOutputTokens: 128}}); err != nil {
		t.Fatal(err)
	}

	const marker = "SYNTHETIC_PRIVATE_HTTP_MARKER"
	out, err := s.Generate(context.Background(), GenerateRequest{ProfileKey: "private-local", Query: marker, PersistenceMode: "private_ephemeral", IncludeCandidates: true})
	if err != nil {
		t.Fatal(err)
	}
	if fixture.request == nil || fixture.request.GetProfile() != sharedv1.Profile_PROFILE_LOCAL_ONLY {
		t.Fatalf("gateway profile = %v, want local_only", fixture.request)
	}
	if fixture.request.GetSource() != marker {
		t.Fatalf("gateway source = %q, want synthetic marker", fixture.request.GetSource())
	}
	if fixture.request.GetPrivacyClass() != sharedv1.PrivacyClass_PRIVACY_CLASS_CONFIDENTIAL || fixture.request.GetPrivacyProvenance() != prosePrivacySource {
		t.Fatalf("gateway privacy contract = %v / %q", fixture.request.GetPrivacyClass(), fixture.request.GetPrivacyProvenance())
	}
	if out.Session.Status != "ephemeral" || out.Session.Query != "" || len(out.Candidates) != 1 {
		t.Fatalf("ephemeral response metadata = %+v, candidates=%d", out.Session, len(out.Candidates))
	}
	for _, table := range []string{"prose_sessions", "prose_rounds", "prose_candidates", "prose_selection_events"} {
		var count int
		if err := db.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 0 {
			t.Fatalf("ephemeral generation persisted %d rows in %s", count, table)
		}
	}
}

func (f *privacyCaptureInference) Run(_ context.Context, request *connect.Request[inferencev1.RunRequest]) (*connect.Response[inferencev1.RunResponse], error) {
	f.request = request.Msg
	return connect.NewResponse(&inferencev1.RunResponse{Error: &inferencev1.InferenceError{
		Code:    inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_UNAVAILABLE,
		Message: "fixture denied provider candidate",
	}}), nil
}

func TestHTTPGatewayCarriesTheSourcedPrivacyContractAndSurfacesRemoteRefusal(t *testing.T) {
	fixture := &privacyCaptureInference{}
	path, handler := inferenceconnect.NewInferenceServiceHandler(fixture)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	defer server.Close()

	_, err := (HTTPGateway{BaseURL: server.URL}).Generate(context.Background(), GatewayRequest{
		Role: "write.default", Instruction: "Write a short response.", Query: "unclassified source",
		Strategy: samplerDirect, K: 1, MaxOutputTokens: 128, Locality: "remote_only",
		PrivacyClass: prosePrivacyClass, PrivacySource: prosePrivacySource,
	})
	if err == nil || !strings.Contains(err.Error(), "fixture denied provider candidate") {
		t.Fatalf("gateway error = %v", err)
	}
	if fixture.request == nil || fixture.request.GetProfile() != sharedv1.Profile_PROFILE_REMOTE_ONLY {
		t.Fatalf("profile request = %v, want remote_only", fixture.request)
	}
	if fixture.request.GetPrivacyClass() != sharedv1.PrivacyClass_PRIVACY_CLASS_CONFIDENTIAL {
		t.Fatalf("privacy class = %s, want the Prose private-editorial-content classification", fixture.request.GetPrivacyClass())
	}
	if fixture.request.GetPrivacyProvenance() != prosePrivacySource {
		t.Fatalf("privacy provenance = %q, want Prose Studio data source", fixture.request.GetPrivacyProvenance())
	}
}

func TestHTTPGatewayRefusesMissingPrivacyProvenanceBeforeDispatch(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	_, err := (HTTPGateway{BaseURL: server.URL}).Generate(context.Background(), GatewayRequest{
		Role: "write.default", Instruction: "Write.", Query: "draft", Strategy: samplerDirect, K: 1,
		MaxOutputTokens: 128, Locality: "remote_only",
	})
	if err == nil || !strings.Contains(err.Error(), "requires privacy provenance") {
		t.Fatalf("missing privacy provenance error = %v", err)
	}
	if called {
		t.Fatal("gateway was called without a sourced privacy classification")
	}
}
