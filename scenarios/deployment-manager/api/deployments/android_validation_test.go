package deployments

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"deployment-manager/profiles"
	"github.com/gorilla/mux"
)

func TestHTTPAndroidRampValidatesImmutableArtifactAndReportsExactDeploymentIdentity(t *testing.T) {
	const ref = "android-debug:" + "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	var created map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer scoped-android-token" {
			t.Errorf("authorization=%q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.Method + " " + r.URL.Path {
		case "GET /api/v1/android/artifacts/metadata":
			if r.URL.Query().Get("immutable_ref") != ref {
				t.Errorf("metadata ref=%q", r.URL.Query().Get("immutable_ref"))
			}
			_, _ = w.Write([]byte(`{"immutable_ref":"` + ref + `","apk_checksum":"sha256:apk","package_name":"com.example.app","version_name":"1.0","version_code":1,"signing":"debug-only"}`))
		case "GET /api/v1/validation/catalog":
			if r.URL.Query().Get("scenario") != "example" {
				t.Errorf("catalog scenario=%q", r.URL.Query().Get("scenario"))
			}
			_, _ = w.Write([]byte(`{"targets":[` +
				`{"descriptor":{"target_id":"android:emulator:local","display_name":"Local emulator","available":true},"kind":"local"},` +
				`{"descriptor":{"target_id":"android-physical-1","display_name":"Moto","available":true,"capabilities":["VALIDATION_TARGET_CAPABILITY_DEVICE_CONTROL","VALIDATION_TARGET_CAPABILITY_SCREEN_RECORDING"]},"kind":"bridge"},` +
				`{"descriptor":{"target_id":"android-a03s","display_name":"Galaxy A03s","available":false,"reason":"device-control transport unavailable"},"kind":"bridge"},` +
				`{"descriptor":{"target_id":"google-cast:tv","display_name":"SmartTV","available":false,"reason":"required screen recording capability is missing"},"kind":"bridge"}]}`))
		case "POST /api/v1/android/matrices":
			if err := json.NewDecoder(r.Body).Decode(&created); err != nil {
				t.Error(err)
			}
			_, _ = w.Write([]byte(`{"run_id":"run-1"}`))
		case "POST /api/v1/validation/matrices/run-1/start":
			_, _ = w.Write([]byte(`{}`))
		case "GET /api/v1/validation/matrices/run-1":
			_, _ = w.Write([]byte(`{"run_id":"run-1","state":"queued","matrix":{"matrix_id":"matrix-1","scenario_name":"example","artifact_digest":"sha256:apk","targets":[{"target_id":"android-physical-1","display_name":"Moto"}]},"selection":{"metadata":{"immutable_ref":"` + ref + `"},"deployment_profile_id":"profile-1","deployment_git_commit":"0123456789012345678901234567890123456789"},"gate":{"passed":true},"release_report":{}}`))
		default:
			t.Errorf("unexpected producer request %s %s", r.Method, r.URL.String())
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	client := &HTTPAndroidRamp{BaseURL: server.URL, Client: server.Client(), Authorize: func(_ context.Context, r *http.Request) error {
		r.Header.Set("Authorization", "Bearer scoped-android-token")
		return nil
	}}
	result, err := client.Start(context.Background(), AndroidValidationRequest{ProfileID: "profile-1", GitCommitHash: "0123456789012345678901234567890123456789", ArtifactRef: ref, Scenario: "example"})
	if err != nil {
		t.Fatal(err)
	}
	if result.RunID != "run-1" || result.State != "queued" || result.ProfileID != "profile-1" || result.TargetID != "android-physical-1" || result.ArtifactDigest != "sha256:apk" {
		t.Fatalf("result=%+v", result)
	}
	selection := created["selection"].(map[string]any)
	if selection["deployment_profile_id"] != "profile-1" || selection["deployment_git_commit"] != "0123456789012345678901234567890123456789" {
		t.Fatalf("deployment identity not bound into matrix: %#v", selection)
	}
	if selection["artifact_digest"] != "sha256:apk" {
		t.Fatalf("artifact digest not bound: %#v", selection)
	}
	if key, ok := selection["idempotency_key"].(string); !ok || !strings.HasPrefix(key, "deployment-manager-android:") {
		t.Fatalf("deployment Android validation lacks an exact idempotency key: %#v", selection)
	}
	targets := selection["targets"].([]any)
	if len(targets) != 1 || targets[0].(map[string]any)["descriptor"].(map[string]any)["target_id"] != "android-physical-1" {
		t.Fatalf("target selection=%#v", targets)
	}
	if created["artifact_ref"] != ref {
		t.Fatalf("immutable ref not sent to guarded matrix API: %#v", created)
	}
}

func TestHTTPAndroidRampRejectsUnavailableExplicitTargetBeforeCreatingMatrix(t *testing.T) {
	const ref = "android-debug:" + "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	createdMatrix := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method + " " + r.URL.Path {
		case "GET /api/v1/android/artifacts/metadata":
			_, _ = w.Write([]byte(`{"immutable_ref":"` + ref + `","apk_checksum":"sha256:apk","package_name":"com.example.app","version_name":"1.0","version_code":1,"signing":"debug-only"}`))
		case "GET /api/v1/validation/catalog":
			_, _ = w.Write([]byte(`{"targets":[{"descriptor":{"target_id":"android-physical-1","display_name":"Moto","available":false,"reason":"device-control cannot reach the phone"},"kind":"bridge"}]}`))
		case "POST /api/v1/android/matrices":
			createdMatrix = true
			t.Error("created a validation matrix for an unavailable target")
			http.Error(w, "unexpected matrix", http.StatusInternalServerError)
		default:
			t.Errorf("unexpected producer request %s %s", r.Method, r.URL.String())
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	client := &HTTPAndroidRamp{BaseURL: server.URL, Client: server.Client(), Authorize: func(_ context.Context, r *http.Request) error {
		r.Header.Set("Authorization", "Bearer scoped-android-token")
		return nil
	}}
	_, err := client.Start(context.Background(), AndroidValidationRequest{
		ProfileID: "profile-1", GitCommitHash: "0123456789012345678901234567890123456789",
		ArtifactRef: ref, Scenario: "example", TargetID: "android-physical-1",
	})
	if err == nil || !strings.Contains(err.Error(), "device-control cannot reach the phone") {
		t.Fatalf("unavailable target error=%v", err)
	}
	if createdMatrix {
		t.Fatal("unavailable target was submitted to scenario-to-android")
	}
}

type androidRampFake struct {
	request AndroidValidationRequest
	result  *AndroidValidationResult
}

func (f *androidRampFake) Start(_ context.Context, request AndroidValidationRequest) (*AndroidValidationResult, error) {
	f.request = request
	return f.result, nil
}
func (f *androidRampFake) Status(context.Context, string) (*AndroidValidationResult, error) {
	return f.result, nil
}

func TestAndroidValidationHandlerRequiresReportReceiptAndBindsProfileScenario(t *testing.T) {
	profileRepo := &orchestratorProfileRepo{profile: &profiles.Profile{ID: "profile-1", Scenario: "example"}}
	ramp := &androidRampFake{result: &AndroidValidationResult{RunID: "run-1", ProfileID: "profile-1", GitCommitHash: "0123456789012345678901234567890123456789", Scenario: "example", ArtifactRef: "android-debug:abc", State: "queued"}}
	handler := NewAndroidValidationHandler(profileRepo, ramp, func(context.Context) error { return nil }, func(context.Context) error { return nil })
	recorder := httptest.NewRecorder()
	newRequest := func() *http.Request {
		return httptest.NewRequest(http.MethodPost, "/api/v1/android/validate", strings.NewReader(`{"profile_id":"profile-1","git_commit_hash":"0123456789012345678901234567890123456789","artifact_ref":"android-debug:abc"}`))
	}
	handler.ServeHTTP(recorder, newRequest())
	if recorder.Code != http.StatusAccepted {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if ramp.request.Scenario != "example" {
		t.Fatalf("scenario was not resolved from the deployment profile: %+v", ramp.request)
	}

	ramp.result.State = "completed"
	recorder = httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/android/validate/{run_id}", handler.Status).Methods(http.MethodGet)
	ramp.result.ReleaseReport = map[string]any{"reported_at": "2026-11-20T00:00:00Z"}
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/android/validate/run-1", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("reported matrix status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	ramp.result.ReleaseReport = map[string]any{"error": "service principal rejected"}
	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/android/validate/run-1", nil))
	if recorder.Code != http.StatusBadGateway {
		t.Fatalf("unreported matrix status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}
