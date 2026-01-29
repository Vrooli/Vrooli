package httputil

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/domain"
)

func TestProtoJSONAndStatus(t *testing.T) {
	msg := &domainpb.BacklogItem{
		Name:        "idea-1",
		Title:       "Title",
		Description: "Desc",
		Status:      "backlog",
		Priority:    2,
		Tags:        []string{"alpha"},
		Created:     "2024-01-01T00:00:00Z",
		Updated:     "2024-01-02T00:00:00Z",
		Kind:        "idea",
	}

	w := httptest.NewRecorder()
	if err := ProtoJSON(w, msg); err != nil {
		t.Fatalf("ProtoJSON error: %v", err)
	}

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("expected content-type application/json, got %q", ct)
	}
	body := w.Body.String()
	if !strings.Contains(body, `"name":"idea-1"`) {
		t.Fatalf("expected proto name field, got %s", body)
	}
	if !strings.Contains(body, `"status":"backlog"`) {
		t.Fatalf("expected proto status field, got %s", body)
	}

	w2 := httptest.NewRecorder()
	if err := ProtoJSONWithStatus(w2, http.StatusCreated, msg); err != nil {
		t.Fatalf("ProtoJSONWithStatus error: %v", err)
	}
	if w2.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", w2.Code)
	}
}

func TestDecodeProtoJSONDiscardUnknown(t *testing.T) {
	payload := []byte(`{"name":"idea-2","title":"T","status":"backlog","priority":3,"kind":"idea","unknown_field":"ignored"}`)
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(payload))

	msg := &domainpb.BacklogItem{}
	if err := DecodeProtoJSON(req, msg); err != nil {
		t.Fatalf("DecodeProtoJSON error: %v", err)
	}
	if msg.Name != "idea-2" {
		t.Fatalf("expected name idea-2, got %q", msg.Name)
	}
	if msg.Priority != 3 {
		t.Fatalf("expected priority 3, got %d", msg.Priority)
	}
}
