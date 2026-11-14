package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/nodereach"
)

func TestDesktopProxyForwardsTypedConnectRequestThroughResolvedScenario(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/vrooli.device_control.v1.desktop.DesktopSessionService/Signal" {
			t.Fatalf("path=%q", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		if string(body) != `{"payload":"offer"}` {
			t.Fatalf("body=%q", body)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, `{"accepted":true}`)
	}))
	defer upstream.Close()
	proxy := newDesktopProxy(func(context.Context) (string, error) { return upstream.URL, nil })
	record := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/vrooli.device_control.v1.desktop.DesktopSessionService/Signal", strings.NewReader(`{"payload":"offer"}`))
	request.Header.Set("Content-Type", "application/json")
	proxy.ServeHTTP(record, request)
	if record.Code != http.StatusOK || !strings.Contains(record.Body.String(), "accepted") {
		t.Fatalf("status=%d body=%q", record.Code, record.Body.String())
	}
}

func TestDesktopTargetProxyUsesBridgeTargetRouteAndServerSideAuth(t *testing.T) {
	bridge := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/targets/node-1/scenarios/device-control/vrooli.device_control.v1.desktop.DesktopSessionService/GetReadiness" {
			t.Fatalf("path=%q", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer owner-token" {
			t.Fatalf("authorization=%q", r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"state":6}`)
	}))
	defer bridge.Close()
	client := nodereach.New(nodereach.Config{BridgeURL: bridge.URL, Token: "owner-token"})
	proxy := newDesktopTargetProxy(func(context.Context) (*nodereach.Client, error) { return client, nil })
	record := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/desktop/node-1/vrooli.device_control.v1.desktop.DesktopSessionService/GetReadiness", strings.NewReader(`{"surface":{}}`))
	request.Header.Set("Content-Type", "application/json")
	request = mux.SetURLVars(request, map[string]string{"nodeID": "node-1"})
	proxy.ServeHTTP(record, request)
	if record.Code != http.StatusOK || !strings.Contains(record.Body.String(), `"state":6`) {
		t.Fatalf("status=%d body=%q", record.Code, record.Body.String())
	}
}

func TestBridgeControlProxyKeepsInteractiveRPCOnBridge(t *testing.T) {
	bridge := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/vrooli.vrooli_bridge.v1.interactive.InteractiveDesktopService/Signal" {
			t.Fatalf("path=%q", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer owner-token" {
			t.Fatalf("authorization=%q", r.Header.Get("Authorization"))
		}
		_, _ = io.WriteString(w, `{"accepted":true}`)
	}))
	defer bridge.Close()
	client := nodereach.New(nodereach.Config{BridgeURL: bridge.URL, Token: "owner-token"})
	proxy := newBridgeControlProxy(func(context.Context) (*nodereach.Client, error) { return client, nil })
	record := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/desktop-bridge/vrooli.vrooli_bridge.v1.interactive.InteractiveDesktopService/Signal", strings.NewReader(`{"requestId":"r1"}`))
	request.Header.Set("Content-Type", "application/json")
	proxy.ServeHTTP(record, request)
	if record.Code != http.StatusOK || !strings.Contains(record.Body.String(), "accepted") {
		t.Fatalf("status=%d body=%q", record.Code, record.Body.String())
	}
}
