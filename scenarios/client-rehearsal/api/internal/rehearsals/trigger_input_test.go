package rehearsals

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestNormalizeTriggerInputBoundsAndSanitizes(t *testing.T) {
	input := map[string]any{
		"body":         map[string]any{"name": "Synthetic user", "items": []any{map[string]any{"count": 2}}},
		"access_token": "synthetic-secret-marker",
	}
	got, replacements, err := normalizeTriggerInput(input)
	if err != nil {
		t.Fatal(err)
	}
	if replacements != 1 {
		t.Fatalf("credential replacements = %d, want 1", replacements)
	}
	if got["access_token"] != "synthetic://credential-replaced" {
		t.Fatalf("secret-shaped field was not replaced: %#v", got["access_token"])
	}
	if input["access_token"] != "synthetic-secret-marker" {
		t.Fatal("normalization mutated the caller's map")
	}

	tooLarge := map[string]any{"payload": strings.Repeat("x", maxTriggerInputBytes)}
	if _, _, err := normalizeTriggerInput(tooLarge); err == nil {
		t.Fatal("expected oversized input rejection")
	}
	tooDeep := any("leaf")
	for i := 0; i < maxTriggerInputDepth; i++ {
		tooDeep = map[string]any{"nested": tooDeep}
	}
	if _, _, err := normalizeTriggerInput(map[string]any{"payload": tooDeep}); err == nil {
		t.Fatal("expected excessive nesting rejection")
	}
	if _, _, err := normalizeTriggerInput(map[string]any{"__proto__": "unsafe"}); err == nil {
		t.Fatal("expected unsafe object key rejection")
	}
	if _, _, err := normalizeTriggerInput(nil); err != nil {
		t.Fatalf("omitted input must remain valid: %v", err)
	}
}

// [REQ:CR-P0-001]
func TestCreateAndRunUsesSuppliedWebhookInputAndDoesNotPersistIt(t *testing.T) {
	tests := []struct {
		name, workflowFile, profileFile string
		input                           map[string]any
		operations                      []string
		noOperations                    []string
	}{
		{
			name:         "article webhook",
			workflowFile: "public-workflows/1110.json",
			profileFile:  "public-workflow-1110-profile.json",
			input:        map[string]any{"body": map[string]any{"type": float64(2), "data": map[string]any{"options": []any{map[string]any{"value": "article.synthetic.invalid/story"}}}}, "access_token": "synthetic-secret-marker"},
			operations:   []string{"GET /story", "notion:databasePage.create"},
		},
		{
			name:         "attendance webhook",
			workflowFile: "public-workflows/16904.json",
			profileFile:  "public-workflow-16904-profile.json",
			input:        map[string]any{"body": map[string]any{"name": "Synthetic Attendee", "email": "new-attendee@synthetic.invalid", "role": "Participant", "eventId": "synthetic-event-1", "day": "Day 1"}},
			operations:   []string{"googleSheets:sheet.read", "googleSheets:sheet.append", "gmail:message.send"},
		},
		{
			name:         "attendance webhook missing name",
			workflowFile: "public-workflows/16904.json",
			profileFile:  "public-workflow-16904-profile.json",
			input:        map[string]any{"body": map[string]any{"email": "missing-name@synthetic.invalid"}},
			noOperations: []string{"googleSheets:sheet.read", "googleSheets:sheet.append", "gmail:message.send"},
		},
		{
			name:         "attendance webhook malformed email",
			workflowFile: "public-workflows/16904.json",
			profileFile:  "public-workflow-16904-profile.json",
			input:        map[string]any{"body": map[string]any{"name": "Synthetic Invalid Email", "email": "invalid-address"}},
			noOperations: []string{"googleSheets:sheet.read", "googleSheets:sheet.append", "gmail:message.send"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMemoryRepository()
			now := time.Date(2026, 11, 14, 12, 0, 0, 0, time.UTC)
			svc := NewService(repo, SyntheticEngine{}, func() time.Time { return now })
			profile := readFixture(t, tt.profileFile)
			got, err := svc.CreateAndRun(context.Background(), CreateInput{
				Title:               "Synthetic webhook with per-run input",
				WorkflowJSON:        readFixture(t, tt.workflowFile),
				TriggerInput:        tt.input,
				SimulatorProfile:    "synthetic-public-profile",
				SimulatorSourceJSON: profile,
			})
			if err != nil {
				t.Fatal(err)
			}
			if got.State != StateNeedsReview {
				t.Fatalf("state = %s, want needs_review; fault=%s trace=%#v", got.State, got.FaultSummary, got.Trace)
			}
			for _, operation := range tt.operations {
				if !traceHasSuccessfulOperation(got.Trace, operation) {
					t.Errorf("expected successful synthetic operation %q in trace: %#v", operation, got.Trace)
				}
			}
			for _, operation := range tt.noOperations {
				if traceHasSuccessfulOperation(got.Trace, operation) {
					t.Errorf("unexpected successful synthetic operation %q in trace: %#v", operation, got.Trace)
				}
			}
			if tt.name == "article webhook" && got.CredentialReplacements != 1 {
				t.Errorf("credential replacements = %d, want one trigger-input replacement", got.CredentialReplacements)
			}
			stored, err := repo.Get(context.Background(), got.ID)
			if err != nil {
				t.Fatal(err)
			}
			encoded, err := json.Marshal(stored)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(encoded), "synthetic-secret-marker") {
				t.Fatal("raw trigger input was persisted into rehearsal evidence")
			}
			if strings.Contains(string(encoded), "synthetic://credential-replaced") {
				t.Fatal("transient trigger input was persisted into rehearsal evidence")
			}
		})
	}
}

func TestCreateAndRunRejectsTriggerInputForTaskScopedExecution(t *testing.T) {
	workflow := `{"name":"manual synthetic","nodes":[{"name":"Manual","type":"n8n-nodes-base.manualTrigger","parameters":{}}],"connections":{}}`
	svc := NewService(newMemoryRepository(), SyntheticEngine{}, func() time.Time { return time.Unix(1, 0) })
	input := CreateInput{Title: "Task scoped synthetic", WorkflowJSON: workflow, TriggerInput: map[string]any{"body": map[string]any{"id": "synthetic-1"}}, taskSandbox: &WorkspaceSandboxTaskRequest{TaskID: "task-1", Name: "trigger-input-test"}}
	_, err := svc.CreateAndRun(context.Background(), input)
	if err == nil || !strings.Contains(err.Error(), "trigger_input") {
		t.Fatalf("task-scoped caller input error = %v, want trigger_input rejection", err)
	}
}
