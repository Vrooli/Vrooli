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
