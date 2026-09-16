package deployment

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"connectrpc.com/connect"
	lpbsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1"
	"landing-page-business-suite-api/internal/administration"
	"landing-page-business-suite-api/internal/delivery"
)

func TestReadinessRejectsMalformedJSONBeforeDependencyAccess(t *testing.T) {
	w := httptest.NewRecorder()
	handler := Readiness(Dependencies{
		BundleKey:  func() string { t.Fatal("bundle key should not be read for malformed input"); return "" },
		WriteError: func(w http.ResponseWriter, status int, _, _ string) { w.WriteHeader(status) },
	})

	handler.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/v1/deploy-readiness", strings.NewReader("{")))

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status=%d", w.Code)
	}
}

func TestConnectReadinessUsesSharedWorkflow(t *testing.T) {
	handler := NewConnectHandler(Dependencies{
		BundleKey: func() string { return "bundle" },
		Storage:   readinessStorage{},
	})

	response, err := handler.CheckReadiness(context.Background(), connect.NewRequest(&lpbsv1.CheckDeploymentReadinessRequest{}))
	if err != nil {
		t.Fatalf("CheckReadiness: %v", err)
	}
	if response.Msg.GetReady() {
		t.Fatal("expected missing storage to make readiness false")
	}
	if got := response.Msg.GetGates()[0].GetName(); got != "download_storage" {
		t.Fatalf("gate name=%q, want download_storage", got)
	}
}

func TestReadinessIncludesStripeGateWhenConfigured(t *testing.T) {
	handler := Readiness(Dependencies{
		BundleKey: func() string { return "bundle" },
		Storage:   readinessStorageWithSettings{},
		StripeReadiness: func(context.Context) Gate {
			return Gate{Name: "stripe_commerce", Message: "missing active Stripe credential fields: stripe-test-secret-key"}
		},
		WriteError: func(w http.ResponseWriter, status int, _, _ string) { w.WriteHeader(status) },
	})

	response := CheckReadiness(context.Background(), Dependencies{
		BundleKey: func() string { return "bundle" },
		Storage:   readinessStorageWithSettings{},
		StripeReadiness: func(context.Context) Gate {
			return Gate{Name: "stripe_commerce", Message: "missing active Stripe credential fields: stripe-test-secret-key"}
		},
	}, Request{})
	if response.Ready || len(response.Gates) != 2 || response.Gates[1].Name != "stripe_commerce" {
		t.Fatalf("expected Stripe readiness gate to block with its diagnostic, got %+v", response)
	}
	if !strings.Contains(response.Error, "stripe-test-secret-key") {
		t.Fatalf("expected exact missing field in readiness error, got %q", response.Error)
	}

	// Exercise the HTTP adapter as well so callers receive a non-success status.
	req := httptest.NewRequest(http.MethodPost, "/api/v1/deploy-readiness", strings.NewReader("{}"))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409 for blocked Stripe readiness, got %d", w.Code)
	}
}

func TestReadinessUsesBoundedStorageProbeWhenProvided(t *testing.T) {
	probe := readinessStorageProbe{err: fmt.Errorf("put object denied")}
	handler := NewConnectHandler(Dependencies{
		BundleKey:   func() string { return "bundle" },
		Storage:     readinessStorageWithSettings{},
		TestStorage: probe,
	})

	response, err := handler.CheckReadiness(context.Background(), connect.NewRequest(&lpbsv1.CheckDeploymentReadinessRequest{}))
	if err != nil {
		t.Fatalf("CheckReadiness: %v", err)
	}
	if response.Msg.GetReady() || !strings.Contains(response.Msg.GetGates()[0].GetMessage(), "put object denied") {
		t.Fatalf("probe failure opened readiness: %+v", response.Msg.GetGates())
	}
}

type readinessStorage struct{}

func (readinessStorage) GetSettings(context.Context, string) (*delivery.StorageSettings, error) {
	return nil, nil
}

type readinessStorageWithSettings struct{}

func (readinessStorageWithSettings) GetSettings(context.Context, string) (*delivery.StorageSettings, error) {
	return &delivery.StorageSettings{Bucket: "bucket", SignedURLTTLSeconds: 900}, nil
}

type readinessStorageProbe struct{ err error }

func (probe readinessStorageProbe) TestConnection(context.Context, string) error { return probe.err }

type readinessCatalog struct{ apps map[string]bool }

func (c readinessCatalog) GetApp(_, appKey string) (*delivery.App, error) {
	if c.apps[appKey] {
		return &delivery.App{}, nil
	}
	return nil, nil
}

// readinessRemote is a deterministic stand-in for the stored remote LPBS
// profile. proxyFn lets a test shape each proxied admin response.
type readinessRemote struct {
	profiles []administration.RemoteProfile
	testErr  error
	proxyFn  func(administration.RemoteProfileProxyRequest) (*administration.RemoteProxyResponse, error)
}

func (f readinessRemote) List(context.Context) ([]administration.RemoteProfile, error) {
	return f.profiles, nil
}

func (f readinessRemote) Test(context.Context, int64) (*administration.RemoteProfile, error) {
	if f.testErr != nil {
		return nil, f.testErr
	}
	return &administration.RemoteProfile{}, nil
}

func (f readinessRemote) Proxy(_ context.Context, _ int64, req administration.RemoteProfileProxyRequest) (*administration.RemoteProxyResponse, error) {
	if f.proxyFn != nil {
		return f.proxyFn(req)
	}
	return &administration.RemoteProxyResponse{StatusCode: http.StatusOK, Body: []byte(`{}`)}, nil
}

func remoteReadyDeps(remote readinessRemote) Dependencies {
	return Dependencies{
		BundleKey:      func() string { return "bundle" },
		Storage:        readinessStorageWithSettings{},
		Catalog:        readinessCatalog{apps: map[string]bool{"web-console": true}},
		RemoteProfiles: remote,
	}
}

func remoteProxyByPath(storageStatus, appStatus int) func(administration.RemoteProfileProxyRequest) (*administration.RemoteProxyResponse, error) {
	return func(req administration.RemoteProfileProxyRequest) (*administration.RemoteProxyResponse, error) {
		switch req.Path {
		case "/admin/download-storage/test":
			return &administration.RemoteProxyResponse{StatusCode: storageStatus}, nil
		case "/admin/download-apps":
			body := []byte(`{"apps":[{"app_key":"web-console","name":"Aquila"}]}`)
			if appStatus != http.StatusOK {
				body = []byte(`{"apps":[]}`)
			}
			return &administration.RemoteProxyResponse{StatusCode: appStatus, Body: body}, nil
		default:
			return &administration.RemoteProxyResponse{StatusCode: http.StatusOK, Body: []byte(`{}`)}, nil
		}
	}
}

func TestCheckReadinessProvesRemoteStorageAndApp(t *testing.T) {
	ctx := context.Background()

	t.Run("remote storage permission failure blocks readiness", func(t *testing.T) {
		deps := remoteReadyDeps(readinessRemote{
			profiles: []administration.RemoteProfile{{ID: 7, Tag: "prod"}},
			proxyFn:  remoteProxyByPath(http.StatusForbidden, http.StatusOK),
		})
		response := CheckReadiness(ctx, deps, Request{AppKey: "web-console", RemoteProfile: "prod"})
		if response.Ready {
			t.Fatalf("expected readiness false, got %+v", response)
		}
		if !strings.Contains(response.Error, "remote_download_storage") {
			t.Fatalf("expected remote_download_storage to be the first failure, got %q", response.Error)
		}
	})

	t.Run("missing remote app key blocks readiness", func(t *testing.T) {
		deps := remoteReadyDeps(readinessRemote{
			profiles: []administration.RemoteProfile{{ID: 7, Tag: "prod"}},
			proxyFn:  remoteProxyByPath(http.StatusOK, http.StatusOK),
		})
		deps.RemoteProfiles = readinessRemote{
			profiles: []administration.RemoteProfile{{ID: 7, Tag: "prod"}},
			proxyFn: func(req administration.RemoteProfileProxyRequest) (*administration.RemoteProxyResponse, error) {
				if req.Path == "/admin/download-apps" {
					return &administration.RemoteProxyResponse{StatusCode: http.StatusOK, Body: []byte(`{"apps":[]}`)}, nil
				}
				return &administration.RemoteProxyResponse{StatusCode: http.StatusOK, Body: []byte(`{}`)}, nil
			},
		}
		response := CheckReadiness(ctx, deps, Request{AppKey: "web-console", RemoteProfile: "prod"})
		if response.Ready {
			t.Fatalf("expected readiness false, got %+v", response)
		}
		if !strings.Contains(response.Error, "remote_app_key") {
			t.Fatalf("expected remote_app_key failure, got %q", response.Error)
		}
	})

	t.Run("inactive remote session blocks readiness before remote probes", func(t *testing.T) {
		deps := remoteReadyDeps(readinessRemote{
			profiles: []administration.RemoteProfile{{ID: 7, Tag: "prod"}},
			testErr:  fmt.Errorf("session expired"),
			proxyFn:  remoteProxyByPath(http.StatusOK, http.StatusOK),
		})
		response := CheckReadiness(ctx, deps, Request{AppKey: "web-console", RemoteProfile: "prod"})
		if response.Ready {
			t.Fatalf("expected readiness false, got %+v", response)
		}
		if !strings.Contains(response.Error, "remote_profile_session") {
			t.Fatalf("expected remote_profile_session failure, got %q", response.Error)
		}
	})

	t.Run("ready remote target passes every remote gate", func(t *testing.T) {
		deps := remoteReadyDeps(readinessRemote{
			profiles: []administration.RemoteProfile{{ID: 7, Tag: "prod"}},
			proxyFn:  remoteProxyByPath(http.StatusOK, http.StatusOK),
		})
		response := CheckReadiness(ctx, deps, Request{AppKey: "web-console", RemoteProfile: "prod"})
		if !response.Ready {
			t.Fatalf("expected readiness true, got %+v", response)
		}
		names := map[string]bool{}
		for _, gate := range response.Gates {
			names[gate.Name] = true
		}
		for _, want := range []string{"remote_profile_session", "remote_download_storage", "remote_app_key"} {
			if !names[want] {
				t.Fatalf("missing gate %q in %+v", want, response.Gates)
			}
		}
	})
}

func TestCheckReadinessServiceProfileUsesAllRemoteGatesWithoutSession(t *testing.T) {
	deps := remoteReadyDeps(readinessRemote{
		profiles: []administration.RemoteProfile{{ID: 7, Tag: "prod-service", AuthMode: administration.RemoteProfileAuthModeService, RemoteServiceSecretConfigured: true, HasSession: false}},
		proxyFn:  remoteProxyByPath(http.StatusOK, http.StatusOK),
	})
	response := CheckReadiness(context.Background(), deps, Request{AppKey: "web-console", RemoteProfile: "prod-service"})
	if !response.Ready {
		t.Fatalf("service profile should be ready without a session: %+v", response)
	}
	for _, gate := range response.Gates {
		if strings.HasPrefix(gate.Name, "remote_") && !gate.Ready {
			t.Fatalf("remote gate %s failed: %s", gate.Name, gate.Message)
		}
	}
}
