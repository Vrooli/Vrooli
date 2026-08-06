package fix

import (
	"strings"
	"testing"
	"time"

	"scenario-to-cloud/domain"
	"scenario-to-cloud/tasks/shared"

	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
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

func TestBuildPromptAndContext_UsesDeploymentLocalNativeCLIInSSHExamples(t *testing.T) {
	input := validFixInput()

	out, err := BuildPromptAndContext(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if strings.Contains(out.Prompt, "source resources/") {
		t.Fatalf("prompt should not mention legacy bash setup paths: %q", out.Prompt)
	}
	if strings.Contains(out.Prompt, " cd /root/Vrooli && vrooli ") {
		t.Fatalf("prompt should not emit bare global vrooli commands: %q", out.Prompt)
	}
	if !strings.Contains(out.Prompt, "bash -lc") {
		t.Fatalf("prompt should wrap SSH examples with bash -lc: %q", out.Prompt)
	}
	if !strings.Contains(out.Prompt, "/root/Vrooli/.vrooli/bin/vrooli") {
		t.Fatalf("prompt should reference deployment-local vrooli binary: %q", out.Prompt)
	}
	if !strings.Contains(out.Prompt, "--no-stale-check scenario stop") || !strings.Contains(out.Prompt, "landing-page-business-suite") {
		t.Fatalf("prompt should use native scenario stop command: %q", out.Prompt)
	}
	if !strings.Contains(out.Prompt, "--no-stale-check scenario start") {
		t.Fatalf("prompt should use native scenario start command: %q", out.Prompt)
	}
	if !strings.Contains(out.Prompt, "--no-stale-check scenario status") {
		t.Fatalf("prompt should use native scenario status command: %q", out.Prompt)
	}
}

func TestBuildPromptAndContext_UsesContractNeutralPathGuidance(t *testing.T) {
	out, err := BuildPromptAndContext(validFixInput())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, key := range []string{"focus-harness-fix", "focus-subject-fix"} {
		att := getAttachment(out.Attachments, key)
		if att == nil {
			t.Fatalf("missing attachment %q", key)
		}
		if strings.Contains(att.Content, "~/Vrooli/scenarios/") {
			t.Fatalf("%s should not mention legacy ~/Vrooli scenario paths: %q", key, att.Content)
		}
		if !strings.Contains(att.Content, "repo contract") && !strings.Contains(att.Content, "contract-defined") {
			t.Fatalf("%s should reference contract-backed path guidance: %q", key, att.Content)
		}
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
