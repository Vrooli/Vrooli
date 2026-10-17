package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDocsValidateDiagramsCLIHandlesValidInvalidAndUnverified(t *testing.T) {
	cases := []struct {
		name    string
		body    string
		wantErr bool
	}{
		{name: "valid", body: `{"engine":"mermaid@11.13.0"}`},
		{name: "invalid", body: `{"findings":[{"code":"mermaid_invalid","message":"Parse error","line":3}]}`, wantErr: true},
		{name: "unverified", body: `{"unverified":true,"findings":[{"code":"mermaid_unverified","message":"node missing","line":1}]}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodGet && r.URL.Path == "/health" {
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`{"status":"healthy"}`))
					return
				}
				if r.Method != http.MethodPost || r.URL.Path != "/knowledge_observatory.v1.KnowledgeObservatoryService/ValidateMarkdownDiagrams" {
					t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(tc.body))
			}))
			defer server.Close()
			t.Setenv("KNOWLEDGE_OBSERVATORY_API_BASE", server.URL)
			app, err := NewApp()
			if err != nil {
				t.Fatalf("NewApp: %v", err)
			}
			err = app.Run([]string{"docs", "validate-diagrams", "--json"})
			if tc.wantErr && err == nil {
				t.Fatal("expected invalid Mermaid to fail")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected CLI error: %v", err)
			}
		})
	}
}
