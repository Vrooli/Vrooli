package handlers

import (
	"agent-manager/internal/orchestration"
	"github.com/google/uuid"
	"github.com/vrooli/api-core/effortauthority"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFiniteEffortHTTPExclusiveChannelPreservedBeforeOwner(t *testing.T) {
	for _, name := range []string{"positive", "duplicate", "blank", "human-mix", "native-mix"} {
		t.Run(name, func(t *testing.T) {
			capture := &createIdentityCapture{}
			handler := New(orchestration.HandlerServices{RunService: capture})
			req := httptest.NewRequest(http.MethodPost, "/api/v1/runs", strings.NewReader(`{"task_id":"`+uuid.NewString()+`"}`))
			req.Header.Set(effortauthority.Header, "disposable-signed-proof")
			switch name {
			case "duplicate":
				req.Header.Add(effortauthority.Header, "second-proof")
			case "blank":
				req.Header.Set(effortauthority.Header, " ")
			case "human-mix":
				req.Header.Set("Authorization", "Bearer human-proof")
			case "native-mix":
				req.Header.Set("X-Agent-Identity-Token", "native-proof")
			}
			response := httptest.NewRecorder()
			handler.CreateRun(response, req)
			if name == "positive" {
				if capture.request.EffortProof != "disposable-signed-proof" || capture.request.TaskID == uuid.Nil {
					t.Fatal("actual HTTP adapter dropped effort proof", response.Body.String())
				}
			} else {
				if capture.request.TaskID != uuid.Nil || response.Code != http.StatusUnauthorized {
					t.Fatal("ambiguous finite channel reached native owner", response.Code)
				}
			}
			if strings.Contains(response.Body.String(), "disposable-signed-proof") {
				t.Fatal("proof leaked")
			}
		})
	}
}
