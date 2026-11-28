package investigation

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"scenario-to-cloud/domain"
	"scenario-to-cloud/tasks/shared"

	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

func TestBuildPromptAndContext_DefaultsAndNote(t *testing.T) {
	dep := validDeploymentForContext()

	prompt, atts, err := buildPromptAndContext(dep, false, "operator note", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(prompt, "Investigate deployment failure for landing-page-business-suite") {
		t.Fatalf("unexpected prompt: %q", prompt)
	}
	if !strings.Contains(prompt, "Mode: report-only") {
		t.Fatalf("expected report-only mode in prompt: %q", prompt)
	}

	required := []string{
		"task-metadata",
		"error-info",
		"safety-rules",
		"diagnostic-checklist",
		"output-format",
		"deployment-manifest",
		"vps-connection",
		"architecture-guide",
		"user-note",
	}
	for _, key := range required {
		if !hasInvestigationAttachment(atts, key) {
			t.Fatalf("expected attachment key %q", key)
		}
	}
}

func TestBuildPromptAndContext_ValidationErrors(t *testing.T) {
	invalidManifest := &domain.Deployment{Manifest: []byte("{")}
	if _, _, err := buildPromptAndContext(invalidManifest, false, "", nil); err == nil {
		t.Fatal("expected manifest parse error")
	}

	m := domain.CloudManifest{
		Scenario: domain.ManifestScenario{ID: "landing-page-business-suite"},
		Target:   domain.ManifestTarget{Type: "vps"},
	}
	manifestJSON, _ := json.Marshal(m)
	noVPS := &domain.Deployment{Manifest: manifestJSON}
	if _, _, err := buildPromptAndContext(noVPS, false, "", nil); err == nil || !strings.Contains(err.Error(), "deployment has no VPS target") {
		t.Fatalf("expected missing vps error, got %v", err)
	}
}

func TestBuildPromptAndContext_UsesSharedOutputFormatForBothModes(t *testing.T) {
	for _, autoFix := range []bool{false, true} {
		t.Run(map[bool]string{false: "report-only", true: "auto-fix"}[autoFix], func(t *testing.T) {
			_, atts, err := buildPromptAndContext(validDeploymentForContext(), autoFix, "", []string{"output-format"})
			if err != nil {
				t.Fatalf("buildPromptAndContext: %v", err)
			}

			var got *domainpb.ContextAttachment
			for _, att := range atts {
				if att != nil && att.Key == "output-format" {
					got = att
					break
				}
			}
			want := shared.BuildOutputFormatAttachment(autoFix)
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("output-format attachment differs from shared owner:\ngot:  %+v\nwant: %+v", got, want)
			}
			if strings.Contains(got.Content, "If you applied fixes, list what you did.") != autoFix {
				t.Fatalf("auto-fix guidance presence does not match mode: %q", got.Content)
			}
		})
	}
}

func TestExtractErrorSummary(t *testing.T) {
	msg := "first line\nsecond line"
	if got := extractErrorSummary(&domain.Deployment{ErrorMessage: &msg}); got != "first line" {
		t.Fatalf("expected first line, got %q", got)
	}

	msg = strings.Repeat("x", 120)
	if got := extractErrorSummary(&domain.Deployment{ErrorMessage: &msg}); !strings.HasSuffix(got, "...") {
		t.Fatalf("expected truncated summary, got %q", got)
	}
}

func TestBuildDiagnosticChecklistAttachment_UsesContractNeutralScenarioGuidance(t *testing.T) {
	att := buildDiagnosticChecklistAttachment()
	if att == nil {
		t.Fatal("expected attachment")
	}
	if strings.Contains(att.Content, "<workdir>/scenarios/<scenario>/.vrooli/service.json") {
		t.Fatalf("diagnostic checklist should not hard-code scenario path layout: %q", att.Content)
	}
	if !strings.Contains(att.Content, "contract-defined scenario root") {
		t.Fatalf("diagnostic checklist should reference contract-defined scenario root: %q", att.Content)
	}
}

func validDeploymentForContext() *domain.Deployment {
	errStep := "preflight"
	errMsg := "Cannot negotiate ALPN protocol"

	m := domain.CloudManifest{
		Scenario: domain.ManifestScenario{ID: "landing-page-business-suite"},
		Target: domain.ManifestTarget{
			Type: "vps",
			VPS: &domain.ManifestVPS{
				Host:    "138.197.95.182",
				Port:    22,
				User:    "root",
				Workdir: "/root/Vrooli",
			},
		},
		Edge: domain.ManifestEdge{Domain: "vrooli.com"},
	}
	manifestJSON, _ := json.Marshal(m)

	return &domain.Deployment{
		ID:           "dep-1",
		Status:       domain.StatusFailed,
		Manifest:     manifestJSON,
		ErrorStep:    &errStep,
		ErrorMessage: &errMsg,
	}
}

func hasInvestigationAttachment(atts []*domainpb.ContextAttachment, key string) bool {
	for _, a := range atts {
		if a != nil && a.Key == key {
			return true
		}
	}
	return false
}
