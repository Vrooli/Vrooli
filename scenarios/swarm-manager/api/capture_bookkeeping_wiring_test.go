package main

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/backlog"
	"swarm-manager/internal/captures"
	"swarm-manager/internal/transitions"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/identity"
	apiconnect "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api/apiconnect"
)

func TestCaptureIntakeUsesExactBookkeepingGrantAtGlobalRoute(t *testing.T) {
	for _, tc := range []struct {
		name       string
		grant      *identity.OperationGrant
		wantStatus int
		wantCount  int
	}{
		{name: "approved domain target", grant: captureIntakeGrant("swarm-manager:*", time.Now().Add(time.Hour)), wantStatus: http.StatusCreated, wantCount: 1},
		{name: "approved exact target", grant: captureIntakeGrant("swarm-manager:capture/intake", time.Now().Add(time.Hour)), wantStatus: http.StatusCreated, wantCount: 1},
		{name: "wrong target", grant: captureIntakeGrant("swarm-manager:capture/other", time.Now().Add(time.Hour)), wantStatus: http.StatusForbidden},
		{name: "wrong operation", grant: &identity.OperationGrant{ID: "grant", Approver: "owner", Operations: []string{"swarm.accept"}, Targets: []string{"swarm-manager:*"}, ExpiresAt: time.Now().Add(time.Hour)}, wantStatus: http.StatusForbidden},
		{name: "expired grant", grant: captureIntakeGrant("swarm-manager:*", time.Now().Add(-time.Minute)), wantStatus: http.StatusForbidden},
		{name: "missing grant", wantStatus: http.StatusForbidden},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			registry, err := transitions.LoadDir(filepath.Join("..", ".vrooli", "swarm-transitions"))
			if err != nil {
				t.Fatalf("load transition registry: %v", err)
			}
			handler := captures.NewHandler(filepath.Join(root, "cache"), filepath.Join(root, "data"), registry)
			workflow := &captureWorkflowInvoker{}
			handler.SetClassificationWorkflow(workflow)
			router := mux.NewRouter()
			router.Use(backlog.InteractiveGrantMiddleware)
			handler.RegisterRoutes(router)

			body, contentType := captureMultipartBody(t, "lifecycle bookkeeping evidence")
			req := httptest.NewRequest(http.MethodPost, "/api/v1/captures", body)
			req.Header.Set("Content-Type", contentType)
			principal := identity.Principal{Kind: identity.ActorAgent, Subject: "worker", Owner: "owner", Verified: true, Scopes: []string{"swarm-manager:write"}, Grant: tc.grant}
			req = req.WithContext(identity.WithPrincipal(context.Background(), principal))
			response := httptest.NewRecorder()
			router.ServeHTTP(response, req)
			if response.Code != tc.wantStatus {
				t.Fatalf("capture POST status=%d body=%q, want %d", response.Code, response.Body.String(), tc.wantStatus)
			}

			listResponse := httptest.NewRecorder()
			router.ServeHTTP(listResponse, httptest.NewRequest(http.MethodGet, "/api/v1/captures", nil))
			var listed struct {
				Captures []struct {
					Text string `json:"text"`
				} `json:"captures"`
			}
			if err := json.Unmarshal(listResponse.Body.Bytes(), &listed); err != nil {
				t.Fatalf("decode capture list: %v; body=%q", err, listResponse.Body.String())
			}
			if len(listed.Captures) != tc.wantCount {
				t.Fatalf("stored captures=%d, want %d", len(listed.Captures), tc.wantCount)
			}
			if tc.wantCount == 1 && listed.Captures[0].Text != "lifecycle bookkeeping evidence" {
				t.Fatalf("stored capture text=%q", listed.Captures[0].Text)
			}
			if workflow.starts != tc.wantCount {
				t.Fatalf("classification starts=%d, want %d", workflow.starts, tc.wantCount)
			}
		})
	}
}

func TestCaptureGrantDoesNotOpenUnknownOrProposalDecisionRoutes(t *testing.T) {
	grant := captureIntakeGrant("swarm-manager:*", time.Now().Add(time.Hour))
	principal := identity.Principal{Kind: identity.ActorAgent, Subject: "worker", Owner: "owner", Verified: true, Scopes: []string{"swarm-manager:write"}, Grant: grant}
	for _, path := range []string{"/api/v1/captures/extra", apiconnect.BacklogServiceDecideAttemptProcedure} {
		router := mux.NewRouter()
		router.Use(backlog.InteractiveGrantMiddleware)
		router.HandleFunc(path, func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }).Methods(http.MethodPost)
		req := httptest.NewRequest(http.MethodPost, path, nil).WithContext(identity.WithPrincipal(context.Background(), principal))
		response := httptest.NewRecorder()
		router.ServeHTTP(response, req)
		if response.Code != http.StatusForbidden {
			t.Fatalf("POST %s status=%d body=%q, want middleware refusal", path, response.Code, response.Body.String())
		}
	}
}

type captureWorkflowInvoker struct {
	starts int
}

func (s *captureWorkflowInvoker) StartWorkflow(_ context.Context, _ agentmanager.Invocation) (agentmanager.WorkflowStart, error) {
	s.starts++
	return agentmanager.WorkflowStart{ExecutionID: "capture-classification", DefinitionDigest: "sha256:capture-test"}, nil
}

func (*captureWorkflowInvoker) CollectWorkflow(context.Context, string) (agentmanager.InvocationCompletion, error) {
	return agentmanager.InvocationCompletion{}, nil
}

func captureIntakeGrant(target string, expiresAt time.Time) *identity.OperationGrant {
	return &identity.OperationGrant{ID: "approved-bookkeeping", Approver: "owner", Operations: []string{"swarm.bookkeeping"}, Targets: []string{target}, ExpiresAt: expiresAt}
}

func captureMultipartBody(t *testing.T, text string) (*bytes.Buffer, string) {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("text", text); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return &body, writer.FormDataContentType()
}
