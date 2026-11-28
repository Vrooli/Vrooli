package delivery

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAuthorizeRequiresAppBeforeDependencyCalls(t *testing.T) {
	called := false
	handler := Authorize(Dependencies{
		UserEmail:  func(context.Context) string { called = true; return "user@example.com" },
		WriteError: func(w http.ResponseWriter, status int, message, kind string) { w.WriteHeader(status) },
	})
	w := httptest.NewRecorder()
	handler(w, httptest.NewRequest(http.MethodGet, "/api/v1/downloads?platform=linux", nil))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
	if called {
		t.Fatal("identity lookup called before required app validation")
	}
}

func TestAuthorizeManagedAssetWithoutArtifactIDFailsClosed(t *testing.T) {
	resolved, recorded := false, false
	var response map[string]any
	handler := Authorize(Dependencies{
		UserEmail: func(context.Context) string { return "member@example.com" },
		Authorize: func(context.Context, string, string, string) (Authorization, error) {
			payload := map[string]string{"artifact_url": "https://stale.example/public.zip"}
			return Authorization{Payload: payload, Managed: true, ArtifactID: 0, SetURL: func(url string) { payload["artifact_url"] = url }}, nil
		},
		ResolveManaged: func(context.Context, int64) (string, bool, error) { resolved = true; return "", false, nil },
		ManagedError:   func(error) (string, string) { return "", "" },
		WriteJSON:      func(_ http.ResponseWriter, value any) { response = value.(map[string]any) },
		WriteError:     func(w http.ResponseWriter, status int, _, _ string) { w.WriteHeader(status) },
		Log:            func(string, map[string]any) {},
		RecordEvent:    func(context.Context, string, string, string) error { recorded = true; return nil },
	})
	w := httptest.NewRecorder()
	handler(w, httptest.NewRequest(http.MethodGet, "/api/v1/downloads?app=desktop&platform=linux", nil))
	if w.Code != http.StatusServiceUnavailable || resolved || recorded || response != nil {
		t.Fatalf("status=%d resolved=%v recorded=%v response=%v", w.Code, resolved, recorded, response)
	}
}

func TestAuthorizeManagedAssetResolvesFreshURLBeforeRecordingSuccess(t *testing.T) {
	requestCtx := context.WithValue(context.Background(), struct{}{}, "request-context")
	var resolveCtx context.Context
	var resolvedID int64
	var eventCount int
	var response map[string]string
	assetURL := "https://catalog.invalid/stale.zip"
	handler := Authorize(Dependencies{
		UserEmail: func(context.Context) string { return "member@example.com" },
		Authorize: func(context.Context, string, string, string) (Authorization, error) {
			payload := map[string]string{"artifact_url": assetURL}
			return Authorization{Payload: payload, Managed: true, ArtifactID: 91, SetURL: func(url string) { payload["artifact_url"] = url }}, nil
		},
		ResolveManaged: func(ctx context.Context, artifactID int64) (string, bool, error) {
			resolveCtx = ctx
			resolvedID = artifactID
			return "https://storage.example/signed-object?token=one-time", true, nil
		},
		RecordEvent: func(context.Context, string, string, string) error {
			eventCount++
			return nil
		},
		WriteJSON:  func(_ http.ResponseWriter, value any) { response = value.(map[string]string) },
		WriteError: func(w http.ResponseWriter, status int, _, _ string) { w.WriteHeader(status) },
		Log:        func(string, map[string]any) {},
	})
	w := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/downloads?app=desktop&platform=linux", nil).WithContext(requestCtx)
	handler(w, request)
	if w.Code != http.StatusOK || response["artifact_url"] != "https://storage.example/signed-object?token=one-time" {
		t.Fatalf("status=%d response=%v", w.Code, response)
	}
	if resolvedID != 91 || resolveCtx != requestCtx || eventCount != 1 {
		t.Fatalf("resolvedID=%d contextPreserved=%v successEvents=%d", resolvedID, resolveCtx == requestCtx, eventCount)
	}
}

func TestAuthorizeManagedAssetWithoutResolverOrURLSetterFailsClosed(t *testing.T) {
	for _, test := range []struct {
		name   string
		setURL func(string)
	}{
		{name: "resolver missing", setURL: func(string) {}},
		{name: "url setter missing", setURL: nil},
	} {
		t.Run(test.name, func(t *testing.T) {
			resolved, recorded, serialized := false, false, false
			var resolver func(context.Context, int64) (string, bool, error)
			if test.setURL == nil {
				resolver = func(context.Context, int64) (string, bool, error) {
					resolved = true
					return "https://fresh.example/object", true, nil
				}
			}
			handler := Authorize(Dependencies{
				UserEmail: func(context.Context) string { return "member@example.com" },
				Authorize: func(context.Context, string, string, string) (Authorization, error) {
					return Authorization{Payload: map[string]string{"artifact_url": "https://stale.example/public.zip"}, Managed: true, ArtifactID: 91, SetURL: test.setURL}, nil
				},
				ResolveManaged: resolver,
				RecordEvent:    func(context.Context, string, string, string) error { recorded = true; return nil },
				WriteJSON:      func(http.ResponseWriter, any) { serialized = true },
				WriteError:     func(w http.ResponseWriter, status int, _, _ string) { w.WriteHeader(status) },
				Log:            func(string, map[string]any) {},
			})
			w := httptest.NewRecorder()
			handler(w, httptest.NewRequest(http.MethodGet, "/api/v1/downloads?app=desktop&platform=linux", nil))
			if w.Code != http.StatusServiceUnavailable || resolved || recorded || serialized {
				t.Fatalf("status=%d resolved=%v recorded=%v serialized=%v", w.Code, resolved, recorded, serialized)
			}
		})
	}
}
