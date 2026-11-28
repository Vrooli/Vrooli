package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vrooli/api-core/identity"
)

func TestAndroidDeploymentRoutesRequireScopedCapabilities(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	read := requireAndroidCapability("scenario-to-android:read", next)
	write := requireAndroidCapability("scenario-to-android:write", next)
	readOnly := identity.Principal{Kind: identity.ActorService, Subject: "dm", Verified: true, Scopes: []string{"scenario-to-android:read"}}

	for _, tc := range []struct {
		name      string
		handler   http.Handler
		principal *identity.Principal
		want      int
	}{
		{name: "anonymous denied", handler: read, want: http.StatusForbidden},
		{name: "scoped read allowed", handler: read, principal: &readOnly, want: http.StatusNoContent},
		{name: "read scope cannot write", handler: write, principal: &readOnly, want: http.StatusForbidden},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tc.principal != nil {
				req = req.WithContext(identity.WithPrincipal(req.Context(), *tc.principal))
			}
			recorder := httptest.NewRecorder()
			tc.handler.ServeHTTP(recorder, req)
			if recorder.Code != tc.want {
				t.Fatalf("status=%d want %d", recorder.Code, tc.want)
			}
		})
	}
}
