package delivery

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"landing-page-business-suite-api/internal/delivery"
)

func adminTestDependencies() AdminDependencies {
	return AdminDependencies{
		BundleKey: func() string { return "bundle" },
		SettingsSnapshot: func(context.Context, string) (*delivery.StorageSettingsSnapshot, error) {
			return &delivery.StorageSettingsSnapshot{Bucket: "downloads"}, nil
		},
		SaveSettings: func(_ context.Context, _ string, _ delivery.StorageSettingsUpdate) (*delivery.StorageSettingsSnapshot, error) {
			return &delivery.StorageSettingsSnapshot{Bucket: "updated-downloads"}, nil
		},
		TestConnection: func(context.Context, string) error { return nil },
		DecodeJSON: func(w http.ResponseWriter, r *http.Request, target any) bool {
			if err := json.NewDecoder(r.Body).Decode(target); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return false
			}
			return true
		},
		WriteSuccessData:   func(w http.ResponseWriter, payload any) { _ = json.NewEncoder(w).Encode(payload) },
		WriteSuccessSimple: func(w http.ResponseWriter) { w.WriteHeader(http.StatusOK) },
		WriteError: func(w http.ResponseWriter, status int, message, kind string) {
			w.WriteHeader(status)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": message, "error_type": kind})
		},
	}
}

func TestGetStorage(t *testing.T) {
	response := httptest.NewRecorder()
	GetStorage(adminTestDependencies())(response, httptest.NewRequest(http.MethodGet, "/api/v1/admin/download-storage", nil))
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "downloads") {
		t.Fatalf("response = %d %s", response.Code, response.Body.String())
	}
}

func TestUpdateStorageRejectsInvalidJSON(t *testing.T) {
	response := httptest.NewRecorder()
	UpdateStorage(adminTestDependencies())(response, httptest.NewRequest(http.MethodPut, "/api/v1/admin/download-storage", strings.NewReader("{")))
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", response.Code)
	}
}

func TestTestStorageMapsUnconfiguredStorageToConflict(t *testing.T) {
	deps := adminTestDependencies()
	deps.TestConnection = func(context.Context, string) error { return delivery.ErrStorageNotConfigured }
	response := httptest.NewRecorder()
	TestStorage(deps)(response, httptest.NewRequest(http.MethodPost, "/api/v1/admin/download-storage/test", nil))
	if response.Code != http.StatusConflict {
		t.Fatalf("status = %d", response.Code)
	}
}

func TestTestStorageMapsProviderFailureToValidationError(t *testing.T) {
	deps := adminTestDependencies()
	deps.TestConnection = func(context.Context, string) error { return errors.New("provider unavailable") }
	response := httptest.NewRecorder()
	TestStorage(deps)(response, httptest.NewRequest(http.MethodPost, "/api/v1/admin/download-storage/test", nil))
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", response.Code)
	}
}

func TestTestStorageReportsStructuredDiagnostic(t *testing.T) {
	deps := adminTestDependencies()
	deps.WriteValidation = func(w http.ResponseWriter, status int, payload any) {
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(payload)
	}
	deps.ValidateStorage = func(context.Context, string) (*delivery.StorageValidationResult, error) {
		diagnostic := &delivery.Diagnostic{
			Code:        delivery.CodePutObjectDenied,
			Summary:     "write readiness object: PutObject denied on bucket \"downloads\"",
			Operation:   "PutObject",
			Bucket:      "downloads",
			Region:      "us-east-1",
			Remediation: "Add s3:PutObject for the bucket object ARN to the VrooliDeliveryBucketAccess policy.",
		}
		result := &delivery.StorageValidationResult{Bucket: "downloads", Region: "us-east-1", Diagnostic: diagnostic}
		return result, &delivery.DiagnosticError{Diagnostic: *diagnostic}
	}

	response := httptest.NewRecorder()
	TestStorage(deps)(response, httptest.NewRequest(http.MethodPost, "/api/v1/admin/download-storage/test", nil))

	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusForbidden)
	}
	body := response.Body.String()
	for _, fragment := range []string{"put_object_denied", "write readiness object", "s3:PutObject"} {
		if !strings.Contains(body, fragment) {
			t.Fatalf("body missing %q: %s", fragment, body)
		}
	}
}

func TestApplyArtifactUsesVerifiedArtifactChecksum(t *testing.T) {
	deps := adminTestDependencies()
	deps.GetArtifact = func(context.Context, string, int64) (*delivery.Artifact, error) {
		return &delivery.Artifact{ID: 41, BundleKey: "bundle", AppKey: "desktop", Platform: "linux", SHA256: strings.Repeat("a", 64), ReleaseVersion: "1.2.3"}, nil
	}
	var saved ManagedAsset
	deps.UpsertManagedAsset = func(_ context.Context, asset ManagedAsset) (any, error) { saved = asset; return asset, nil }
	body := `{"app_key":"desktop","platform":"linux","artifact_id":41,"checksum":"` + strings.Repeat("a", 64) + `"}`
	response := httptest.NewRecorder()
	ApplyArtifact(deps)(response, httptest.NewRequest(http.MethodPost, "/api/v1/admin/downloads/artifacts/apply", strings.NewReader(body)))
	if response.Code != http.StatusOK || saved.Checksum != strings.Repeat("a", 64) || saved.ArtifactID == nil || *saved.ArtifactID != 41 {
		t.Fatalf("status=%d saved=%+v body=%s", response.Code, saved, response.Body.String())
	}
}

func TestApplyArtifactRejectsChecksumMismatchBeforeUpsert(t *testing.T) {
	deps := adminTestDependencies()
	deps.GetArtifact = func(context.Context, string, int64) (*delivery.Artifact, error) {
		return &delivery.Artifact{ID: 41, BundleKey: "bundle", AppKey: "desktop", Platform: "linux", SHA256: strings.Repeat("a", 64), ReleaseVersion: "1.2.3"}, nil
	}
	upserted := false
	deps.UpsertManagedAsset = func(context.Context, ManagedAsset) (any, error) { upserted = true; return nil, nil }
	body := `{"app_key":"desktop","platform":"linux","artifact_id":41,"checksum":"` + strings.Repeat("b", 64) + `"}`
	response := httptest.NewRecorder()
	ApplyArtifact(deps)(response, httptest.NewRequest(http.MethodPost, "/api/v1/admin/downloads/artifacts/apply", strings.NewReader(body)))
	if response.Code != http.StatusConflict || upserted {
		t.Fatalf("status=%d upserted=%v body=%s", response.Code, upserted, response.Body.String())
	}
}

func TestSetArtifactCurrentPersistsVerifiedArtifactChecksum(t *testing.T) {
	deps := adminTestDependencies()
	deps.GetArtifact = func(context.Context, string, int64) (*delivery.Artifact, error) {
		return &delivery.Artifact{ID: 41, BundleKey: "bundle", AppKey: "desktop", Platform: "linux", SHA256: strings.Repeat("c", 64), ReleaseVersion: "1.2.3"}, nil
	}
	deps.GetManagedAsset = func(string, string, string) (*ManagedAsset, error) { return nil, errors.New("missing") }
	deps.IsAssetNotFound = func(error) bool { return true }
	var saved ManagedAsset
	deps.UpsertManagedAsset = func(_ context.Context, asset ManagedAsset) (any, error) { saved = asset; return asset, nil }
	body := `{"app_key":"desktop","platform":"linux","artifact_id":41}`
	response := httptest.NewRecorder()
	SetArtifactCurrent(deps)(response, httptest.NewRequest(http.MethodPost, "/api/v1/admin/downloads/artifacts/current", strings.NewReader(body)))
	if response.Code != http.StatusOK || saved.Checksum != strings.Repeat("c", 64) {
		t.Fatalf("status=%d saved=%+v body=%s", response.Code, saved, response.Body.String())
	}
}

func TestArtifactAssociationRejectsWrongOwnerOrPlatformBeforeUpsert(t *testing.T) {
	for _, test := range []struct {
		name     string
		artifact delivery.Artifact
		current  bool
		body     string
	}{
		{name: "wrong app", artifact: delivery.Artifact{ID: 41, BundleKey: "bundle", AppKey: "other", Platform: "linux", SHA256: strings.Repeat("a", 64), ReleaseVersion: "1.2.3"}},
		{name: "wrong platform", artifact: delivery.Artifact{ID: 41, BundleKey: "bundle", AppKey: "desktop", Platform: "windows", SHA256: strings.Repeat("a", 64), ReleaseVersion: "1.2.3"}},
		{name: "set current wrong platform", current: true, artifact: delivery.Artifact{ID: 41, BundleKey: "bundle", AppKey: "desktop", Platform: "windows", SHA256: strings.Repeat("a", 64), ReleaseVersion: "1.2.3"}},
		{name: "release version mismatch", artifact: delivery.Artifact{ID: 41, BundleKey: "bundle", AppKey: "desktop", Platform: "linux", SHA256: strings.Repeat("a", 64), ReleaseVersion: "1.2.3"}, body: `{"app_key":"desktop","platform":"linux","artifact_id":41,"release_version":"9.9.9"}`},
		{name: "set current missing release version", current: true, artifact: delivery.Artifact{ID: 41, BundleKey: "bundle", AppKey: "desktop", Platform: "linux", SHA256: strings.Repeat("a", 64)}},
	} {
		t.Run(test.name, func(t *testing.T) {
			deps := adminTestDependencies()
			deps.GetArtifact = func(context.Context, string, int64) (*delivery.Artifact, error) { return &test.artifact, nil }
			upserted, existingRead := false, false
			deps.UpsertManagedAsset = func(context.Context, ManagedAsset) (any, error) { upserted = true; return nil, nil }
			deps.GetManagedAsset = func(string, string, string) (*ManagedAsset, error) {
				existingRead = true
				return nil, errors.New("missing")
			}
			deps.IsAssetNotFound = func(error) bool { return true }
			body := test.body
			if body == "" {
				body = `{"app_key":"desktop","platform":"linux","artifact_id":41}`
			}
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPost, "/api/v1/admin/downloads/artifacts/apply", strings.NewReader(body))
			if test.current {
				SetArtifactCurrent(deps)(recorder, request)
			} else {
				ApplyArtifact(deps)(recorder, request)
			}
			if recorder.Code != http.StatusConflict || upserted || existingRead {
				t.Fatalf("status=%d upserted=%v existingRead=%v body=%s", recorder.Code, upserted, existingRead, recorder.Body.String())
			}
		})
	}
}

func TestArtifactMutationHandlersFailClosedWhenDependenciesAreMissing(t *testing.T) {
	for _, test := range []struct {
		name    string
		current bool
		remove  func(*AdminDependencies)
	}{
		{name: "apply resolver missing", remove: func(d *AdminDependencies) { d.GetArtifact = nil }},
		{name: "apply writer missing", remove: func(d *AdminDependencies) { d.UpsertManagedAsset = nil }},
		{name: "current existing lookup missing", current: true, remove: func(d *AdminDependencies) { d.GetManagedAsset = nil }},
		{name: "current not-found classifier missing", current: true, remove: func(d *AdminDependencies) { d.IsAssetNotFound = nil }},
	} {
		t.Run(test.name, func(t *testing.T) {
			deps := adminTestDependencies()
			deps.GetArtifact = func(context.Context, string, int64) (*delivery.Artifact, error) { return nil, nil }
			deps.UpsertManagedAsset = func(context.Context, ManagedAsset) (any, error) { return nil, nil }
			deps.GetManagedAsset = func(string, string, string) (*ManagedAsset, error) { return nil, errors.New("missing") }
			deps.IsAssetNotFound = func(error) bool { return true }
			test.remove(&deps)
			body := `{"app_key":"desktop","platform":"linux","artifact_id":41}`
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPost, "/api/v1/admin/downloads/artifacts", strings.NewReader(body))
			if test.current {
				SetArtifactCurrent(deps)(recorder, request)
			} else {
				ApplyArtifact(deps)(recorder, request)
			}
			if recorder.Code != http.StatusServiceUnavailable {
				t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
			}
		})
	}
}

func TestSetChannelHaltDryRunDoesNotMutateOwner(t *testing.T) {
	deps := adminTestDependencies()
	called := false
	deps.GetChannelHead = func(string, string, string) (*delivery.ChannelHead, error) {
		return &delivery.ChannelHead{Revision: 4}, nil
	}
	deps.SetChannelHalt = func(context.Context, delivery.ChannelHaltRequest) (*delivery.ChannelHalt, error) {
		called = true
		return nil, errors.New("dry-run must not mutate")
	}
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/admin/download-channels/halt", strings.NewReader(`{"app_key":"demo","variant_key":"default","expected_revision":4,"halted":true,"dry_run":true}`))
	SetChannelHalt(deps)(response, request)
	if response.Code != http.StatusOK || called || !strings.Contains(response.Body.String(), `"outcome":"preview"`) {
		t.Fatalf("status=%d called=%t body=%s", response.Code, called, response.Body.String())
	}
}

func TestRecoverChannelDryRunUsesOwnerValidationWithoutMutation(t *testing.T) {
	deps := adminTestDependencies()
	called := false
	deps.RecoverChannel = func(_ context.Context, request delivery.ChannelRecoveryRequest) (*delivery.ChannelRecovery, error) {
		called = true
		if !request.DryRun || request.Action != "rollback" || request.DataCompatibility != "compatible" {
			t.Fatalf("request = %#v", request)
		}
		return &delivery.ChannelRecovery{AppKey: request.AppKey, VariantKey: request.VariantKey, Revision: request.ExpectedRevision, Action: request.Action, Outcome: "preview", Health: "unknown", DryRun: true}, nil
	}
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/admin/download-channels/recover", strings.NewReader(`{"app_key":"demo","variant_key":"default","expected_revision":4,"action":"rollback","expected_predecessor_revision":3,"data_compatibility":"compatible","dry_run":true}`))
	RecoverChannel(deps)(response, request)
	if response.Code != http.StatusOK || !called || !strings.Contains(response.Body.String(), `"outcome":"preview"`) {
		t.Fatalf("status=%d called=%t body=%s", response.Code, called, response.Body.String())
	}
}
