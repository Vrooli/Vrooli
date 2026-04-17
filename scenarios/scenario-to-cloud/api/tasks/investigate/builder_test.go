package investigate

import (
	"scenario-to-cloud/domain"
	"scenario-to-cloud/tasks/shared"
	"strings"
	"testing"

	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

func TestBuildPromptAndContext_Validation(t *testing.T) {
	valid := validInvestigateInput(domain.EffortLogs)

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

func TestBuildPromptAndContext_EffortDrivesContextSelection(t *testing.T) {
	include := []string{
		"task-metadata",
		"error-info",
		"safety-rules",
		"diagnostic-checklist",
		"output-format",
		"vps-connection",
		"deployment-manifest",
		"architecture-guide",
	}

	checks := validInvestigateInput(domain.EffortChecks)
	checks.Request.IncludeContexts = include
	checksOut, err := BuildPromptAndContext(checks)
	if err != nil {
		t.Fatalf("checks effort: %v", err)
	}
	if !strings.Contains(checksOut.Prompt, "Mode: Quick checks") {
		t.Fatalf("expected checks mode prompt, got %q", checksOut.Prompt)
	}
	if hasAttachmentKey(checksOut.Attachments, "diagnostic-checklist") {
		t.Fatal("did not expect diagnostic-checklist for checks effort")
	}
	if hasAttachmentKey(checksOut.Attachments, "architecture-guide") {
		t.Fatal("did not expect architecture-guide for checks effort")
	}

	logs := validInvestigateInput(domain.EffortLogs)
	logs.Request.IncludeContexts = include
	logsOut, err := BuildPromptAndContext(logs)
	if err != nil {
		t.Fatalf("logs effort: %v", err)
	}
	if !hasAttachmentKey(logsOut.Attachments, "diagnostic-checklist") {
		t.Fatal("expected diagnostic-checklist for logs effort")
	}
	if hasAttachmentKey(logsOut.Attachments, "architecture-guide") {
		t.Fatal("did not expect architecture-guide for logs effort")
	}

	trace := validInvestigateInput(domain.EffortTrace)
	trace.Request.IncludeContexts = include
	traceOut, err := BuildPromptAndContext(trace)
	if err != nil {
		t.Fatalf("trace effort: %v", err)
	}
	if !hasAttachmentKey(traceOut.Attachments, "diagnostic-checklist") {
		t.Fatal("expected diagnostic-checklist for trace effort")
	}
	if !hasAttachmentKey(traceOut.Attachments, "architecture-guide") {
		t.Fatal("expected architecture-guide for trace effort")
	}
	if !hasAttachmentKey(traceOut.Attachments, "focus-harness") || !hasAttachmentKey(traceOut.Attachments, "focus-subject") {
		t.Fatal("expected both focus attachments when focus.harness and focus.subject are true")
	}

	harness := attachmentByKey(traceOut.Attachments, "focus-harness")
	subject := attachmentByKey(traceOut.Attachments, "focus-subject")
	for key, att := range map[string]*domainpb.ContextAttachment{
		"focus-harness": harness,
		"focus-subject": subject,
	} {
		if att == nil {
			t.Fatalf("missing attachment %q", key)
		}
		if strings.Contains(att.Content, "~/Vrooli/scenarios/") {
			t.Fatalf("%s should not mention legacy ~/Vrooli scenario paths: %q", key, att.Content)
		}
	}
}

func validInvestigateInput(effort domain.InvestigationEffort) shared.TaskInput {
	errStep := "preflight"
	errMsg := "Cannot negotiate ALPN protocol"
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
		TaskType: domain.TaskTypeInvestigate,
		Focus:    domain.TaskFocus{Harness: true, Subject: true},
		Effort:   effort,
		Note:     "user note",
	}
	return shared.TaskInput{
		Deployment: dep,
		Manifest:   manifest,
		Request:    req,
		Iteration:  1,
	}
}

func hasAttachmentKey(atts []*domainpb.ContextAttachment, key string) bool {
	return attachmentByKey(atts, key) != nil
}

func attachmentByKey(atts []*domainpb.ContextAttachment, key string) *domainpb.ContextAttachment {
	for _, a := range atts {
		if a != nil && a.Key == key {
			return a
		}
	}
	return nil
}
