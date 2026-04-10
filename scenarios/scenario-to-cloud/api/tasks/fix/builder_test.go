package fix

import (
	"strings"
	"testing"
	"time"

	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"

	"scenario-to-cloud/domain"
	"scenario-to-cloud/tasks/shared"
)

func TestBuildPromptAndContext_Validation(t *testing.T) {
	valid := validFixInput()

	tests := []struct {
		name  string
		input shared.TaskInput
		want  string
	}{
		{name: "missing request", input: shared.TaskInput{}, want: "request is required"},
		{name: "missing deployment", input: shared.TaskInput{Request: valid.Request}, want: "deployment is required"},
		{name: "missing manifest", input: shared.TaskInput{Request: valid.Request, Deployment: valid.Deployment}, want: "manifest is required"},
	}

	for _, tt := range tests {
		_, err := BuildPromptAndContext(tt.input)
		if err == nil || !strings.Contains(err.Error(), tt.want) {
			t.Fatalf("%s: expected error containing %q, got %v", tt.name, tt.want, err)
		}
	}
}

func TestBuildPromptAndContext_IncludesIterationStateAndSourceFindings(t *testing.T) {
	input := validFixInput()
	findings := "Root cause: missing postgres dependency."
	input.SourceFindings = &findings
	input.Iteration = 2
	input.PreviousIterations = []domain.FixIterationRecord{
		{
			Number:           1,
			StartedAt:        time.Date(2026, 2, 7, 12, 0, 0, 0, time.UTC),
			EndedAt:          time.Date(2026, 2, 7, 12, 5, 0, 0, time.UTC),
			DiagnosisSummary: "postgres missing",
			ChangesSummary:   "added dependency",
			DeployTriggered:  true,
			VerifyResult:     "fail",
			Outcome:          "continue",
		},
	}

	out, err := BuildPromptAndContext(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(out.Prompt, "# Fix Task - Iteration 2/5") {
		t.Fatalf("missing iteration header in prompt: %q", out.Prompt)
	}
	if !strings.Contains(out.Prompt, "<investigation_findings>") {
		t.Fatal("expected source investigation findings block in prompt")
	}

	if !hasAttachment(out.Attachments, "iteration-state") {
		t.Fatal("expected iteration-state attachment")
	}

	vpsAtt := getAttachment(out.Attachments, "vps-connection")
	if vpsAtt == nil {
		t.Fatal("expected vps-connection attachment")
	}
	if vpsAtt.Priority != "high" {
		t.Fatalf("expected elevated vps-connection priority high, got %q", vpsAtt.Priority)
	}
}

func validFixInput() shared.TaskInput {
	errStep := "preflight"
	errMsg := "preflight failed"
	dep := &domain.Deployment{
		ID:           "dep-1",
		Status:       domain.StatusFailed,
		ErrorStep:    &errStep,
		ErrorMessage: &errMsg,
	}
	manifest := &domain.CloudManifest{
		Scenario: domain.ManifestScenario{ID: "landing-page-business-suite"},
		Target: domain.ManifestTarget{
			Type: "vps",
			VPS: &domain.ManifestVPS{
				Host:    "138.197.95.182",
				Port:    22,
				User:    "root",
				KeyPath: "~/.ssh/id_ed25519",
				Workdir: "/root/Vrooli",
			},
		},
		Edge: domain.ManifestEdge{Domain: "vrooli.com"},
	}
	req := &domain.CreateTaskRequest{
		TaskType: domain.TaskTypeFix,
		Focus:    domain.TaskFocus{Harness: true, Subject: true},
		Permissions: domain.FixPermissions{
			Immediate:  true,
			Permanent:  true,
			Prevention: false,
		},
		MaxIterations: 5,
	}
	return shared.TaskInput{
		Deployment: dep,
		Manifest:   manifest,
		Request:    req,
		Iteration:  1,
	}
}

func hasAttachment(atts []*domainpb.ContextAttachment, key string) bool {
	return getAttachment(atts, key) != nil
}

func getAttachment(atts []*domainpb.ContextAttachment, key string) *domainpb.ContextAttachment {
	for _, a := range atts {
		if a != nil && a.Key == key {
			return a
		}
	}
	return nil
}
