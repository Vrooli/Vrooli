package services

import (
	"context"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSubmitFixToSwarmManagerReturnsBacklogIdentity(t *testing.T) {
	var itemPart string
	var manifestPart string
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/backlog" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

		_, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if err != nil {
			t.Fatalf("failed to parse content type: %v", err)
		}
		reader := multipart.NewReader(r.Body, params["boundary"])
		for {
			part, err := reader.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Fatalf("failed to read multipart part: %v", err)
			}
			data, err := io.ReadAll(part)
			if err != nil {
				t.Fatalf("failed to read part body: %v", err)
			}
			switch part.FormName() {
			case "item":
				itemPart = string(data)
			case "files_manifest":
				manifestPart = string(data)
			}
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"item":{"name":"fix-abc","title":"Fix ABC","kind":"fix","status":"backlog"}}`))
	}

	server := httptest.NewServer(http.HandlerFunc(handler))
	defer server.Close()

	service := NewAppService(nil)
	service.scenarioURL = func(context.Context, string) (string, error) {
		return server.URL, nil
	}

	result, err := service.submitFixToSwarmManager(context.Background(), map[string]interface{}{
		"name": "fix-abc", "title": "Fix ABC", "kind": "fix",
	}, []swarmEvidenceFile{{Path: "evidence/report.json", Content: []byte(`{"ok":true}`), ContentType: "application/json"}})
	if err != nil {
		t.Fatalf("submitFixToSwarmManager returned error: %v", err)
	}
	if result == nil || result.Kind != "fix" || result.Name != "fix-abc" {
		t.Fatalf("expected fix identity, got %#v", result)
	}
	if !strings.Contains(itemPart, `"name":"fix-abc"`) {
		t.Fatalf("item part was not populated correctly: %s", itemPart)
	}
	if !strings.Contains(manifestPart, `"path":"evidence/report.json"`) {
		t.Fatalf("manifest part was not populated correctly: %s", manifestPart)
	}
}
