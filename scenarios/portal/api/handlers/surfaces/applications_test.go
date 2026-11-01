package surfaces

import (
	"bytes"
	"context"
	"image"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/identity"
	wire "github.com/vrooli/vrooli/packages/proto/gen/go/portal/v1/surfaces"
	domain "portal/internal/surfaces"
)

type lifecycleVerifier struct{ principal identity.Principal }

func (v lifecycleVerifier) Verify(context.Context, string) (identity.Principal, error) {
	return v.principal, nil
}

func TestApplicationLifecycleRequiresVerifiedScopedOperator(t *testing.T) {
	calls := 0
	catalog := &domain.Applications{Run: func(context.Context) ([]byte, error) {
		return []byte(`{"success":true,"scenarios":[{"name":"example","status":"running"}]}`), nil
	}, RunLifecycle: func(context.Context, string, string) error { calls++; return nil }}
	principal := identity.Principal{Kind: identity.ActorHuman, Subject: "owner", Verified: true, Scopes: []string{"vrooli:write"}, ExpiresAt: time.Now().Add(time.Hour)}
	h := &applicationLifecycleHandler{catalog: catalog, verifier: lifecycleVerifier{principal}}
	request := func(token string) *connect.Request[wire.ApplicationLifecycleRequest] {
		r := connect.NewRequest(&wire.ApplicationLifecycleRequest{Scenario: "example", Action: wire.ApplicationLifecycleAction_APPLICATION_LIFECYCLE_ACTION_STOP})
		if token != "" {
			r.Header().Set("Authorization", "Bearer "+token)
		}
		return r
	}
	_, err := h.Act(context.Background(), request(""))
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
	principal.Scopes = []string{"vrooli:read"}
	h.verifier = lifecycleVerifier{principal}
	_, err = h.Act(context.Background(), request("token"))
	require.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err))
	principal.Kind = identity.ActorAgent
	principal.Scopes = []string{"vrooli:write"}
	h.verifier = lifecycleVerifier{principal}
	_, err = h.Act(context.Background(), request("token"))
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
	require.Zero(t, calls)
	principal.Kind = identity.ActorHuman
	h.verifier = lifecycleVerifier{principal}
	response, err := h.Act(context.Background(), request("token"))
	require.NoError(t, err)
	require.Equal(t, "unverified", response.Msg.Outcome)
	require.Equal(t, 1, calls)
}

func TestApplicationArtworkEndpointAndMetadata(t *testing.T) {
	root := t.TempDir()
	public := filepath.Join(root, "web-console", "ui", "public", "public")
	require.NoError(t, os.MkdirAll(public, 0700))
	var imageData bytes.Buffer
	require.NoError(t, png.Encode(&imageData, image.NewRGBA(image.Rect(0, 0, 192, 192))))
	require.NoError(t, os.WriteFile(filepath.Join(public, "icon-192.png"), imageData.Bytes(), 0600))
	catalog := &domain.Applications{Branding: &domain.ApplicationBranding{ScenarioRoot: func(name string) (string, error) { return filepath.Join(root, name), nil }}}
	router := mux.NewRouter()
	ApplicationsModule(catalog).Mount(router)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/applications/web-console/icon", nil)
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)
	require.Equal(t, http.StatusOK, res.Code)
	require.Equal(t, "image/png", res.Header().Get("Content-Type"))
	require.Equal(t, imageData.Bytes(), res.Body.Bytes())
	require.Equal(t, "nosniff", res.Header().Get("X-Content-Type-Options"))
	cached := httptest.NewRequest(http.MethodGet, req.URL.String(), nil)
	cached.Header.Set("If-None-Match", res.Header().Get("ETag"))
	cachedRes := httptest.NewRecorder()
	router.ServeHTTP(cachedRes, cached)
	require.Equal(t, http.StatusNotModified, cachedRes.Code)
	require.Empty(t, cachedRes.Body.Bytes())
	missing := httptest.NewRecorder()
	router.ServeHTTP(missing, httptest.NewRequest(http.MethodGet, "/api/v1/applications/missing/icon", nil))
	require.Equal(t, http.StatusNotFound, missing.Code)
	app := applicationProto(domain.Application{Scenario: "web-console", Label: "Aquila", IconPath: "/api/v1/applications/web-console/icon"})
	require.Equal(t, "Aquila", app.Label)
	require.Equal(t, "web-console", app.Scenario)
	require.Equal(t, "unverified", app.TvQualification)
	require.NotEmpty(t, app.IconPath)
	_, _, err := catalog.Branding.Icon(context.Background(), "../web-console")
	require.Error(t, err)
}
