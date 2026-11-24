package httpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gorilla/mux"
	appruns "test-genie/internal/app/runs"
	sharedruns "test-genie/internal/shared/runs"
)

func TestPersistExperienceCohortReceiptHandlerStoresAndReturnsMissingProvenance(t *testing.T) {
	scenariosRoot := t.TempDir()
	artifactRoot := filepath.Join(t.TempDir(), "demo-artifacts")
	manifestDir := filepath.Join(scenariosRoot, "demo", ".vrooli")
	if err := os.MkdirAll(manifestDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(manifestDir, "service.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(artifactRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := sharedruns.NewIndex(artifactRoot).Append(sharedruns.RunRecord{RunID: "run-1", Scenario: "demo", TargetKind: "scenario", TargetID: "demo", StartedAt: time.Now().UTC(), CompletedAt: time.Now().UTC(), Status: sharedruns.StatusAborted}); err != nil {
		t.Fatal(err)
	}
	svc := appruns.NewService(scenariosRoot, nil, nil, nil).SetArtifactRootResolver(func(string) (string, error) { return artifactRoot, nil })
	server := &Server{runsService: svc}
	body, err := json.Marshal(appruns.ExperienceCohortInput{Cancelled: true, Scenarios: []appruns.ExperienceCohortRunInput{{Scenario: "demo", RunID: "run-1", InvokerStatus: "cancelled"}}})
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/experience-cohort-receipts", bytes.NewReader(body)).WithContext(context.Background())
	recorder := httptest.NewRecorder()
	server.handlePersistExperienceCohortReceipt(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	var stored appruns.StoredExperienceCohortReceipt
	if err := json.NewDecoder(recorder.Body).Decode(&stored); err != nil {
		t.Fatal(err)
	}
	if stored.Receipt.Status != "cancelled" || stored.Artifact.SHA256 == "" || stored.Receipt.Scenarios[0].Verdict != "" {
		t.Fatalf("stored receipt = %+v", stored)
	}
	if len(stored.Receipt.Scenarios[0].Missing) == 0 {
		t.Fatal("missing canonical evidence was not retained")
	}
	get := httptest.NewRequest(http.MethodGet, "/api/v1/experience-cohort-receipts/"+stored.Receipt.ReceiptID, nil)
	get = mux.SetURLVars(get, map[string]string{"receiptID": stored.Receipt.ReceiptID})
	getRecorder := httptest.NewRecorder()
	server.handleGetExperienceCohortReceipt(getRecorder, get)
	if getRecorder.Code != http.StatusOK {
		t.Fatalf("GET status=%d body=%s", getRecorder.Code, getRecorder.Body.String())
	}
	var loaded appruns.StoredExperienceCohortReceipt
	if err := json.NewDecoder(getRecorder.Body).Decode(&loaded); err != nil {
		t.Fatal(err)
	}
	if loaded.Receipt.ReceiptID != stored.Receipt.ReceiptID || loaded.Artifact.SHA256 != stored.Artifact.SHA256 {
		t.Fatalf("GET receipt = %+v", loaded)
	}
}
