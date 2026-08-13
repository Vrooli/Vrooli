package session

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
	identityv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/identity"
	identityconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/identity/identity_v1connect"
)

type refreshIdentity struct {
	identityconnect.UnimplementedIdentityServiceHandler
}

func (refreshIdentity) Refresh(_ context.Context, _ *connect.Request[identityv1.RefreshRequest]) (*connect.Response[identityv1.RefreshResponse], error) {
	return connect.NewResponse(&identityv1.RefreshResponse{Token: "new-access", RefreshToken: "new-refresh"}), nil
}

func TestClientRefreshesOnceAndReplaysOriginalRequest(t *testing.T) {
	var originalCalls, refreshCalls int
	var authorization []string
	var paths []string
	_, refreshHandler := identityconnect.NewIdentityServiceHandler(refreshIdentity{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		authorization = append(authorization, r.Header.Get("Authorization"))
		if strings.HasSuffix(r.URL.Path, "IdentityService/Refresh") {
			refreshCalls++
			refreshHandler.ServeHTTP(w, r)
			return
		}
		originalCalls++
		if originalCalls == 1 {
			w.WriteHeader(http.StatusUnauthorized)
			io.WriteString(w, `{"code":"unauthenticated","message":"expired"}`)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"ok":true}`)
	}))
	defer srv.Close()

	core, err := cliapp.NewStandardScenarioApp(cliapp.StandardScenarioOptions{
		Name: "bridge-test", Version: "0.0.0", DefaultAPIBase: srv.URL, AllowAnonymous: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	core.ConfigFile = &cliutil.ConfigFile{Path: filepath.Join(t.TempDir(), "config.json")}
	core.Config.APIBase = srv.URL
	core.Config.Token = "old-access"
	core.Config.RefreshToken = "old-refresh"
	transportClient, _ := NewConnectHTTPClient(core)
	transportClient.(*client).exchange = func(context.Context) (string, string, error) {
		return "", "", errors.New("local exchange unavailable")
	}
	req, err := http.NewRequest(http.MethodPost, srv.URL+"/api/v1/example", bytes.NewBufferString(`{"value":"body"}`))
	if err != nil {
		t.Fatal(err)
	}
	resp, err := transportClient.Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200 (original=%d refresh=%d auth=%v paths=%v)", resp.StatusCode, originalCalls, refreshCalls, authorization, paths)
	}
	if originalCalls != 2 || refreshCalls != 1 {
		t.Fatalf("calls original=%d refresh=%d, want 2/1", originalCalls, refreshCalls)
	}
	if core.Config.Token != "new-access" || core.Config.RefreshToken != "new-refresh" {
		t.Fatalf("session not replaced: %#v", core.Config)
	}
	if authorization[0] != "Bearer old-access" || authorization[2] != "Bearer new-access" {
		t.Fatalf("authorization sequence = %v", authorization)
	}
}

func TestClientRecoversStaleSavedTokenThroughLocalBinding(t *testing.T) {
	var originalCalls int
	var authorization []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		originalCalls++
		authorization = append(authorization, r.Header.Get("Authorization"))
		if originalCalls == 1 {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = io.WriteString(w, `{"code":"unauthenticated","message":"expired"}`)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"ok":true}`)
	}))
	defer srv.Close()

	core, err := cliapp.NewStandardScenarioApp(cliapp.StandardScenarioOptions{
		Name: "bridge-test", Version: "0.0.0", DefaultAPIBase: srv.URL, AllowAnonymous: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	core.ConfigFile = &cliutil.ConfigFile{Path: filepath.Join(t.TempDir(), "config.json")}
	core.Config.APIBase = srv.URL
	core.Config.Token = "stale-access"
	core.Config.RefreshToken = "stale-refresh"
	transportClient, _ := NewConnectHTTPClient(core)
	transport := transportClient.(*client)
	transport.exchange = func(context.Context) (string, string, error) {
		return "local-access", "local-refresh", nil
	}
	req, err := http.NewRequest(http.MethodPost, srv.URL+"/api/v1/example", bytes.NewBufferString(`{"value":"body"}`))
	if err != nil {
		t.Fatal(err)
	}
	resp, err := transportClient.Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if originalCalls != 2 {
		t.Fatalf("original calls = %d, want 2", originalCalls)
	}
	if got, want := authorization, []string{"Bearer stale-access", "Bearer local-access"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("authorization sequence = %v, want %v", got, want)
	}
	if core.Config.Token != "local-access" || core.Config.RefreshToken != "local-refresh" {
		t.Fatalf("local session not persisted: %#v", core.Config)
	}
}
