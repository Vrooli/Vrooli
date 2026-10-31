package transport

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"testing"

	"deployment-manager/dependencies"
	"deployment-manager/deployments"
	"deployment-manager/fitness"
	"deployment-manager/swaps"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/identity"
	releasesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/releases"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestRoutesExposeEveryGeneratedOperatorService(t *testing.T) {
	h := NewHandler(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	routes := Routes(h)
	if got, want := len(routes), 9; got != want {
		t.Fatalf("generated service route count = %d, want %d", got, want)
	}
	for _, route := range routes {
		if route.Path == "" || route.Handler == nil {
			t.Fatalf("invalid generated route: %#v", route)
		}
	}
}

func TestTransportMethodsFailClosedWhenOptionalHandlerIsMissing(t *testing.T) {
	h := NewHandler(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	req := connect.NewRequest(structpb.NewStructValue(&structpb.Struct{Fields: map[string]*structpb.Value{
		"profile_id":      structpb.NewStringValue("profile/1"),
		"deployment_id":   structpb.NewStringValue("deployment/1"),
		"scenario":        structpb.NewStringValue("demo"),
		"from":            structpb.NewStringValue("postgres"),
		"to":              structpb.NewStringValue("sqlite"),
		"id":              structpb.NewStringValue("approval/1"),
		"release_id":      structpb.NewStringValue("release/1"),
		"git_commit_hash": structpb.NewStringValue("abc"),
		"name":            structpb.NewStringValue("task"),
	}}))
	tests := []struct {
		name string
		call func(context.Context, *connect.Request[structpb.Value]) (*connect.Response[structpb.Value], error)
	}{
		{"list telemetry", h.ListTelemetry},
		{"upload telemetry", h.UploadTelemetry},
		{"report migration", h.ReportMigration},
		{"status migration", h.StatusMigration},
		{"get lpbs config", h.GetLPBSConfig},
		{"save lpbs config", h.SaveLPBSConfig},
		{"list releases", h.ListReleases},
		{"get release", h.GetRelease},
		{"get release dossier", h.GetReleaseDossier},
		{"reverify release", h.ReverifyRelease},
		{"start release", h.StartRelease},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := tc.call(context.Background(), req)
			if connect.CodeOf(err) != connect.CodeUnimplemented {
				t.Fatalf("error code = %s, want unimplemented: %v", connect.CodeOf(err), err)
			}
		})
	}
	_, err := (releasesService{h}).GetOperation(context.Background(), connect.NewRequest(&releasesv1.GetReleaseOperationRequest{OperationId: "operation/1"}))
	if connect.CodeOf(err) != connect.CodeUnimplemented {
		t.Fatalf("missing release operation handler error code = %s, want unimplemented: %v", connect.CodeOf(err), err)
	}
}

func TestTransportMutationsFailClosedWithoutAuthorizationBoundary(t *testing.T) {
	h := NewHandler(
		nil, nil, nil, nil,
		nil,
		nil,
		func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(`{"ok":true}`)) },
		nil, nil,
		nil,
		nil, nil,
		nil, nil, nil, nil,
	)
	_, err := h.UploadTelemetry(context.Background(), connect.NewRequest(structpb.NewNullValue()))
	if connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("unconfigured mutation error code = %s, want unauthenticated: %v", connect.CodeOf(err), err)
	}
}

func TestTransportReadsFailClosedWithoutAuthorizationBoundary(t *testing.T) {
	h := NewHandler(
		nil, nil, nil, nil, nil,
		func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(`{"ok":true}`)) },
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
	)
	_, err := h.ListTelemetry(context.Background(), connect.NewRequest(structpb.NewNullValue()))
	if connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("unconfigured read error code = %s, want unauthenticated: %v", connect.CodeOf(err), err)
	}
}

func TestConfiguredAuthorizationBlocksRoutedMutations(t *testing.T) {
	h := NewHandler(
		nil, nil, nil, nil, nil,
		nil,
		func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(`{"ok":true}`)) },
		nil, nil,
		nil,
		nil, nil,
		nil, nil, nil, nil,
	).WithAuthorization(func(context.Context) error {
		return errors.New("missing verified principal")
	})

	_, err := h.UploadTelemetry(context.Background(), connect.NewRequest(structpb.NewNullValue()))
	if connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("mutation error code = %s, want unauthenticated", connect.CodeOf(err))
	}
}

func TestConfiguredAuthorizationDistinguishesMissingIdentityFromMissingCapability(t *testing.T) {
	h := NewHandler(
		nil, nil, nil, nil, nil,
		nil,
		func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(`{"ok":true}`)) },
		nil, nil,
		nil,
		nil, nil,
		nil, nil, nil, nil,
	).WithAuthorization(func(ctx context.Context) error {
		if _, ok := identity.PrincipalFromContext(ctx); !ok {
			return errors.New("missing verified principal")
		}
		return errors.New("write capability required")
	})

	verified := identity.WithPrincipal(context.Background(), identity.Principal{Kind: identity.ActorHuman, Subject: "operator", Verified: true})
	_, err := h.UploadTelemetry(verified, connect.NewRequest(structpb.NewNullValue()))
	if connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("verified principal error code = %s, want permission denied", connect.CodeOf(err))
	}
}

func TestTypedReleaseAdapterUsesCanonicalView(t *testing.T) {
	h := (&Handler{releaseList: func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"releases":[{"id":"r1","profile_id":"p1","git_commit_hash":"abc","release_version":"1.2.3","channel":"stable","status":"published","platforms":[{"platform":"linux-x64","status":"published"}]}]}`))
	}}).WithReadAuthorization(func(context.Context) error { return nil })
	response, err := (releasesService{h}).List(context.Background(), connect.NewRequest(&releasesv1.ListReleasesRequest{ProfileId: "p1", Limit: 10}))
	if err != nil {
		t.Fatal(err)
	}
	if len(response.Msg.Releases) != 1 || response.Msg.Releases[0].ReleaseId != "r1" || response.Msg.Releases[0].Platforms[0].Platform != "linux-x64" {
		t.Fatalf("typed release response = %+v", response.Msg)
	}
}

func TestTypedReleaseDossierAdapterUsesCanonicalView(t *testing.T) {
	h := (&Handler{}).WithReleaseDossierHandler(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"schema_version":1,"generated_at":"2026-09-08T00:00:00Z","release":{"id":"r1","profile_id":"p1","git_commit_hash":"abc","release_version":"1.2.3","channel":"stable","status":"published"},"health":{"release_id":"r1","status":"healthy","observed_at":"2026-09-08T00:00:00Z","publication_verified":true,"client_updates_healthy":true},"missing_proof":[]}`))
	}).WithReadAuthorization(func(context.Context) error { return nil })
	response, err := (releasesService{h}).Dossier(context.Background(), connect.NewRequest(&releasesv1.GetReleaseDossierRequest{ReleaseId: "r1"}))
	if err != nil {
		t.Fatal(err)
	}
	if response.Msg.GetDossier().GetRelease().GetReleaseId() != "r1" || response.Msg.GetDossier().GetHealth().GetStatus() != "healthy" {
		t.Fatalf("typed dossier response = %+v", response.Msg)
	}
}

func TestTypedStartReleasePreservesCloudHandoff(t *testing.T) {
	var request map[string]interface{}
	h := (&Handler{releaseStart: func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read start request: %v", err)
		}
		if err := json.Unmarshal(body, &request); err != nil {
			t.Fatalf("decode start request: %v", err)
		}
		_, _ = w.Write([]byte(`{"operation_id":"op-1","release_id":"rel-1","status":"queued"}`))
	}}).WithAuthorization(func(context.Context) error { return nil })
	response, err := (releasesService{h}).Start(context.Background(), connect.NewRequest(&releasesv1.StartReleaseRequest{
		ProfileId: "p1", GitCommitHash: "commit-1", ReleaseVersion: "1.0.0",
		CloudManifestJson: `{"scenario":{"id":"demo"}}`, CloudDeploymentName: "demo-prod",
		CloudBundlePath: "/tmp/demo.tar.gz", CloudBundleSha256: "sha256:bundle", CloudBundleSizeBytes: 42,
		CloudRunPreflight: true,
	}))
	if err != nil {
		t.Fatal(err)
	}
	if response.Msg.GetOperationId() != "op-1" || request["cloud_deployment_name"] != "demo-prod" || request["cloud_bundle_path"] != "/tmp/demo.tar.gz" || request["cloud_run_preflight"] != true {
		t.Fatalf("typed start response/request = %s / %#v", response.Msg, request)
	}
	manifest, ok := request["cloud_manifest"].(map[string]interface{})
	if !ok || manifest["scenario"] == nil {
		t.Fatalf("cloud manifest was not preserved as an object: %#v", request["cloud_manifest"])
	}
}

func TestTypedGetOperationPreservesDurableAmbiguousStanding(t *testing.T) {
	h := (&Handler{releaseOperation: func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"operation_id":"op-1","release_id":"rel-1","profile_id":"p1","status":"ambiguous","active_stage":"reconcile","error":"owner uncertain"}`))
	}}).WithReadAuthorization(func(context.Context) error { return nil })
	response, err := (releasesService{h}).GetOperation(context.Background(), connect.NewRequest(&releasesv1.GetReleaseOperationRequest{OperationId: "op-1"}))
	if err != nil {
		t.Fatal(err)
	}
	operation := response.Msg.GetOperation()
	if operation.GetOperationId() != "op-1" || operation.GetReleaseId() != "rel-1" || operation.GetStatus() != releasesv1.ReleaseOperationStatus_RELEASE_OPERATION_STATUS_AMBIGUOUS || operation.GetActiveStage() != "reconcile" || operation.GetError() != "owner uncertain" {
		t.Fatalf("typed operation response = %+v", operation)
	}
}

func TestTypedReconcileUsesDedicatedReleaseHandler(t *testing.T) {
	h := (&Handler{releaseReconcile: func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("deep"); got != "true" {
			t.Errorf("deep query = %q, want true", got)
		}
		_, _ = w.Write([]byte(`{"release":{"id":"rel-1","profile_id":"p1","git_commit_hash":"abc","release_version":"1.0.0","channel":"stable","status":"ambiguous"}}`))
	}}).WithAuthorization(func(context.Context) error { return nil })
	response, err := (releasesService{h}).Reconcile(context.Background(), connect.NewRequest(&releasesv1.ReconcileReleaseRequest{ReleaseId: "rel-1", Deep: true}))
	if err != nil {
		t.Fatal(err)
	}
	if response.Msg.GetRelease().GetReleaseId() != "rel-1" || response.Msg.GetRelease().GetStatus() != "ambiguous" {
		t.Fatalf("typed reconcile response = %+v", response.Msg)
	}
}

func TestTypedRecoverPreservesOwnerReceipt(t *testing.T) {
	h := (&Handler{releaseRecover: func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"release_id":"rel-1","dry_run":false,"receipt":{"receipt_id":"receipt-1","release_id":"rel-1","candidate_id":"candidate-1","destination_revision_id":"destination-1","deployment_id":"deployment-1","action":"halt","outcome":"halted","health":"stopped","external_receipt":"owner-1","observed_at":"2026-09-09T00:00:00Z"}}`))
	}}).WithAuthorization(func(context.Context) error { return nil })
	response, err := (releasesService{h}).Recover(context.Background(), connect.NewRequest(&releasesv1.RecoverReleaseRequest{ReleaseId: "rel-1", ReviewKey: "review-1", CandidateId: "candidate-1", DestinationRevisionId: "destination-1", Action: "halt", Confirmation: "halt rel-1"}))
	if err != nil {
		t.Fatal(err)
	}
	if response.Msg.GetReleaseId() != "rel-1" || response.Msg.GetReceipt().GetExternalReceipt() != "owner-1" || response.Msg.GetReceipt().GetOutcome() != "halted" {
		t.Fatalf("typed recovery response = %+v", response.Msg)
	}
}

func TestInvokeTranslatesHTTPResponsesAndRequestShape(t *testing.T) {
	var gotMethod, gotPath, gotQuery, gotContentType string
	fn := func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath, gotQuery, gotContentType = r.Method, r.URL.Path, r.URL.RawQuery, r.Header.Get("Content-Type")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"count":2}`))
	}
	h := NewHandler(nil, nil, nil, nil, nil, fn, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil).WithAuthorization(func(context.Context) error { return nil }).WithReadAuthorization(func(context.Context) error { return nil })
	value := structpb.NewStructValue(&structpb.Struct{Fields: map[string]*structpb.Value{"name": structpb.NewStringValue("demo")}})
	response, err := h.ListTelemetry(context.Background(), connect.NewRequest(value))
	if err != nil {
		t.Fatal(err)
	}
	if gotMethod != http.MethodGet || gotPath != "/api/v1/telemetry" || gotContentType != "" {
		t.Fatalf("request = %s %s content-type=%q", gotMethod, gotPath, gotContentType)
	}
	if response.Msg.GetStructValue().Fields["ok"].GetBoolValue() != true || gotQuery != "" {
		t.Fatalf("response/query = %v/%q", response.Msg, gotQuery)
	}

	response, err = h.invoke(context.Background(), http.MethodPost, "/path", value, fn, nil, nil)
	if err != nil || response == nil {
		t.Fatalf("payload invoke error = %v", err)
	}
	if gotMethod != http.MethodPost || gotContentType != "application/json" {
		t.Fatalf("payload request = %s content-type=%q", gotMethod, gotContentType)
	}
}

func TestInvokeMapsHTTPAndDecodeFailures(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
		want   connect.Code
	}{
		{"bad request", http.StatusBadRequest, `{"error":"bad"}`, connect.CodeInvalidArgument},
		{"not found", http.StatusNotFound, `{"error":"missing"}`, connect.CodeNotFound},
		{"conflict", http.StatusConflict, `{"error":"duplicate"}`, connect.CodeAlreadyExists},
		{"unauthorized", http.StatusUnauthorized, `{"error":"auth"}`, connect.CodeUnauthenticated},
		{"forbidden", http.StatusForbidden, `{"error":"forbidden"}`, connect.CodePermissionDenied},
		{"server error", http.StatusInternalServerError, `{"error":"broken"}`, connect.CodeInternal},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := NewHandler(nil, nil, nil, nil, nil, func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil).WithReadAuthorization(func(context.Context) error { return nil })
			_, err := h.ListTelemetry(context.Background(), connect.NewRequest(structpb.NewNullValue()))
			if connect.CodeOf(err) != tc.want {
				t.Fatalf("error code = %s, want %s: %v", connect.CodeOf(err), tc.want, err)
			}
		})
	}

	h := NewHandler(nil, nil, nil, nil, nil, func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("not-json")) }, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil).WithReadAuthorization(func(context.Context) error { return nil })
	_, err := h.ListTelemetry(context.Background(), connect.NewRequest(structpb.NewNullValue()))
	if connect.CodeOf(err) != connect.CodeInternal {
		t.Fatalf("decode error code = %s, want internal", connect.CodeOf(err))
	}
}

func TestFieldAndStatusHelpers(t *testing.T) {
	value := structpb.NewStructValue(&structpb.Struct{Fields: map[string]*structpb.Value{"name": structpb.NewStringValue("value")}})
	if got := stringField(value, "name"); got != "value" {
		t.Fatalf("stringField = %q", got)
	}
	if got := stringFieldDefault(value, "missing", "fallback"); got != "fallback" {
		t.Fatalf("stringFieldDefault = %q", got)
	}
	if got := stringFieldDefault(nil, "missing", "fallback"); got != "fallback" {
		t.Fatalf("nil stringFieldDefault = %q", got)
	}
	for status, want := range map[int]connect.Code{
		http.StatusBadRequest:   connect.CodeInvalidArgument,
		http.StatusNotFound:     connect.CodeNotFound,
		http.StatusConflict:     connect.CodeAlreadyExists,
		http.StatusUnauthorized: connect.CodeUnauthenticated,
		http.StatusForbidden:    connect.CodePermissionDenied,
		http.StatusOK:           connect.CodeInternal,
	} {
		if got := codeForStatus(status); got != want {
			t.Errorf("codeForStatus(%d) = %s, want %s", status, got, want)
		}
	}
}

func TestDomainAdaptersTranslateRealHandlerResponses(t *testing.T) {
	logger := func(string, map[string]interface{}) {}
	h := NewHandler(
		dependencies.NewHandler(logger),
		fitness.NewHandler(logger),
		deployments.NewHandler(logger),
		nil,
		swaps.NewHandler(nil, logger),
		func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(`{"status":"ok"}`)) },
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
	).WithAuthorization(func(context.Context) error { return nil }).WithReadAuthorization(func(context.Context) error { return nil })
	call := func(fields map[string]string) *connect.Request[structpb.Value] {
		values := make(map[string]*structpb.Value, len(fields))
		for key, value := range fields {
			values[key] = structpb.NewStringValue(value)
		}
		return connect.NewRequest(structpb.NewStructValue(&structpb.Struct{Fields: values}))
	}
	if response, err := h.Score(context.Background(), call(map[string]string{"scenario": "demo"})); err != nil || response == nil {
		t.Fatalf("Score() = %v, %v", response, err)
	}
	if _, err := h.Analyze(context.Background(), call(map[string]string{})); connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("Analyze() error code = %s, want invalid argument: %v", connect.CodeOf(err), err)
	}
	if _, err := h.Deploy(context.Background(), call(map[string]string{"profile_id": "bad"})); connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("Deploy() error code = %s, want not found: %v", connect.CodeOf(err), err)
	}
	if _, err := h.Status(context.Background(), call(map[string]string{"deployment_id": "missing"})); connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("Status() error code = %s, want not found: %v", connect.CodeOf(err), err)
	}
	if response, err := h.AnalyzeSwaps(context.Background(), call(map[string]string{"from": "postgres", "to": "sqlite"})); err != nil || response == nil {
		t.Fatalf("AnalyzeSwaps() = %v, %v", response, err)
	}
	if response, err := h.CascadeSwaps(context.Background(), call(map[string]string{"from": "postgres", "to": "sqlite"})); err != nil || response == nil {
		t.Fatalf("CascadeSwaps() = %v, %v", response, err)
	}
	if response, err := h.ListTelemetry(context.Background(), call(nil)); err != nil || response == nil {
		t.Fatalf("ListTelemetry() = %v, %v", response, err)
	}
	if _, err := h.ListSwaps(context.Background(), call(nil)); connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("ListSwaps() error code = %s, want invalid argument: %v", connect.CodeOf(err), err)
	}
	if _, err := h.AnalyzeSwaps(context.Background(), call(map[string]string{"from": "", "to": ""})); connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("AnalyzeSwaps(empty) error code = %s, want invalid argument: %v", connect.CodeOf(err), err)
	}
	if _, err := h.CascadeSwaps(context.Background(), call(map[string]string{"from": "", "to": ""})); connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("CascadeSwaps(empty) error code = %s, want invalid argument: %v", connect.CodeOf(err), err)
	}
	if _, err := h.ApplySwaps(context.Background(), call(nil)); connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("ApplySwaps() error code = %s, want invalid argument: %v", connect.CodeOf(err), err)
	}
	if _, err := h.ApplySwapsToProfile(context.Background(), call(nil)); connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("ApplySwapsToProfile() error code = %s, want invalid argument: %v", connect.CodeOf(err), err)
	}
}
