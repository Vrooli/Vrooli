package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/vrooli/secrets-manager-native-host/protocol"
)

func TestHTTPAuthorityDrivesBoundHostSession(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer owner-token" || r.Header.Get("X-Secrets-Manager-Native-Host-Token") != "transport-token" {
			http.Error(w, "missing authority headers", http.StatusUnauthorized)
			return
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		switch r.URL.Path {
		case "/api/v1/native-host/enroll":
			if body["extension_id"] != "extension-fixture" {
				http.Error(w, "wrong extension", http.StatusBadRequest)
				return
			}
		case "/api/v1/native-host/unlock":
			if body["grant_id"] != "grant-fixture" || body["field"] != "password" {
				http.Error(w, "wrong grant", http.StatusBadRequest)
				return
			}
		case "/api/v1/native-host/fill":
			_ = json.NewEncoder(w).Encode(map[string]string{"credential": "authority-secret"})
			return
		case "/api/v1/native-host/metadata":
			_ = json.NewEncoder(w).Encode(map[string]any{"items": []map[string]any{{"grant_id": "grant-fixture", "item_id": "item-fixture", "name": "Fixture", "type": "login", "revision": 1}}})
			return
		case "/api/v1/native-host/save", "/api/v1/native-host/update":
			_ = json.NewEncoder(w).Encode(map[string]any{"item_id": "item-fixture", "name": "Fixture", "type": "login", "revision": 1})
			return
		case "/api/v1/native-host/revoke":
		default:
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer server.Close()
	authority := &httpAuthority{baseURL: server.URL, transportToken: "transport-token", client: server.Client(), sessions: make(map[string]httpAuthoritySession)}
	host := protocol.NewHost(authority)

	enroll := nativeProtocolRequest("enroll", "enroll-http")
	enroll.Payload["workspace_id"] = "workspace-fixture"
	enroll.Payload["enrollment_token"] = "owner-token"
	if response := host.Handle(context.Background(), enroll); !response.OK {
		t.Fatalf("enroll = %+v", response)
	}
	unlock := nativeProtocolRequest("unlock", "unlock-http")
	unlock.Payload["workspace_id"] = "workspace-fixture"
	unlock.Payload["grant_id"] = "grant-fixture"
	unlock.Payload["field"] = "password"
	unlock.Payload["session_token"] = "owner-token"
	unlockResponse := host.Handle(context.Background(), unlock)
	if !unlockResponse.OK {
		t.Fatalf("unlock = %+v", unlockResponse)
	}
	fill := nativeProtocolRequest("fill", "fill-http")
	fill.SessionID = unlockResponse.SessionID
	fill.Payload["field"] = "password"
	fillResponse := host.Handle(context.Background(), fill)
	if !fillResponse.OK || fillResponse.Payload["credential"] != "authority-secret" {
		t.Fatalf("fill = %+v", fillResponse)
	}
	revoke := nativeProtocolRequest("revoke", "revoke-http")
	revoke.SessionID = unlockResponse.SessionID
	if response := host.Handle(context.Background(), revoke); !response.OK {
		t.Fatalf("revoke = %+v", response)
	}
}

func TestNativeHostAuthorityConfigurationFailsClosed(t *testing.T) {
	t.Setenv(nativeHostBaseURL, "")
	t.Setenv(nativeHostTransport, "")
	if authority := newHTTPAuthorityFromEnv(); authority != nil {
		t.Fatal("authority was configured without endpoint and transport token")
	}

	t.Setenv(nativeHostBaseURL, "http://127.0.0.1:1")
	t.Setenv(nativeHostTransport, "transport-token")
	if authority := newHTTPAuthorityFromEnv(); authority == nil {
		t.Fatal("authority was not created from complete configuration")
	}
}

func nativeProtocolRequest(operation, nonce string) protocol.Request {
	return protocol.Request{
		Protocol: protocol.ProtocolName, RequestID: "request-" + nonce, Operation: operation,
		ExtensionID: "extension-fixture", Nonce: nonce, ExpiresAt: time.Now().UTC().Add(time.Minute).Unix(),
		Scope:   protocol.Scope{Origin: "https://example.test", TabID: "tab-1", FrameID: "frame-1", DocumentID: "document-1", ItemID: "item-fixture", Revision: 1},
		Payload: map[string]string{},
	}
}
