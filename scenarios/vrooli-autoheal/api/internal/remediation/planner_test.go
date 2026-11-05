package remediation

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/incidents"
)

func TestGenerateWritesNVIDIARemediationArtifactUnderUserState(t *testing.T) {
	stateRoot := t.TempDir()
	t.Setenv("VROOLI_STATE_ROOT", stateRoot)
	service, err := NewService()
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	service.now = func() time.Time { return time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC) }
	incident := incidents.Incident{
		ID:          "inc_test",
		Fingerprint: "incfp_test",
		EvidenceItems: []incidents.EvidenceItem{{
			Kind: "missing_nvidia_module_package",
			Data: map[string]any{
				"expectedPackage": "linux-modules-nvidia-580-open-6.17.0-23-generic",
				"runningKernel":   "6.17.0-23-generic",
			},
		}},
		RemediationCandidates: []incidents.RemediationCandidate{{
			ID:                "ubuntu-nvidia-kernel-module-mismatch",
			Title:             "Install matching NVIDIA kernel module package",
			Applicability:     "applicable",
			RequiresOperator:  true,
			RequiresPrivilege: true,
			RiskLevel:         "moderate",
			TemplateID:        "ubuntu-nvidia-kernel-module-mismatch",
			PostChecks:        []string{"nvidia-smi"},
		}},
	}

	resp, err := service.Generate(incident, "ubuntu-nvidia-kernel-module-mismatch")
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if resp.Artifact.Path == "" {
		t.Fatal("artifact path is empty")
	}
	if filepath.Clean(resp.Artifact.Path) == filepath.Clean(".") {
		t.Fatalf("artifact path = %q", resp.Artifact.Path)
	}
	if _, err := os.Stat(filepath.Join(resp.Artifact.Path, "remediation.sh")); err != nil {
		t.Fatalf("remediation.sh not written: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(resp.Artifact.Path, "remediation.sh"))
	if err != nil {
		t.Fatalf("read remediation.sh: %v", err)
	}
	if !strings.Contains(string(content), "apt-get -s install") || !strings.Contains(string(content), "Type yes to continue") {
		t.Fatalf("script is missing safety gates:\n%s", string(content))
	}
}

func TestOutcomeUpdatesGeneratedArtifactMetadata(t *testing.T) {
	stateRoot := t.TempDir()
	t.Setenv("VROOLI_STATE_ROOT", stateRoot)
	service, err := NewService()
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	service.now = func() time.Time { return time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC) }
	incident := incidents.Incident{
		ID:          "inc_test",
		Fingerprint: "incfp_test",
		EvidenceItems: []incidents.EvidenceItem{{
			Kind: "missing_nvidia_module_package",
			Data: map[string]any{
				"expectedPackage": "linux-modules-nvidia-580-open-6.17.0-23-generic",
				"runningKernel":   "6.17.0-23-generic",
			},
		}},
		RemediationCandidates: []incidents.RemediationCandidate{{
			ID:            "ubuntu-nvidia-kernel-module-mismatch",
			Title:         "Install matching NVIDIA kernel module package",
			Applicability: "applicable",
			TemplateID:    "ubuntu-nvidia-kernel-module-mismatch",
		}},
	}
	resp, err := service.Generate(incident, "ubuntu-nvidia-kernel-module-mismatch")
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	incident.RemediationArtifacts = []incidents.RemediationArtifact{resp.Artifact}
	outcome, err := service.Outcome(incident, "ubuntu-nvidia-kernel-module-mismatch", OutcomeRequest{
		Status: "verified",
		Note:   "post-checks are clean",
	})
	if err != nil {
		t.Fatalf("Outcome() error = %v", err)
	}
	if outcome.Status != "verified" {
		t.Fatalf("outcome status = %q, want verified", outcome.Status)
	}
	data, err := os.ReadFile(filepath.Join(resp.Artifact.Path, "metadata.json"))
	if err != nil {
		t.Fatalf("read metadata.json: %v", err)
	}
	var metadata map[string]any
	if err := json.Unmarshal(data, &metadata); err != nil {
		t.Fatalf("unmarshal metadata: %v", err)
	}
	if _, ok := metadata["outcome"]; !ok {
		t.Fatalf("metadata missing outcome: %#v", metadata)
	}
}

func TestOutcomeRefusesInvalidStatus(t *testing.T) {
	service, err := NewService()
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	_, err = service.Outcome(incidents.Incident{
		RemediationCandidates: []incidents.RemediationCandidate{{ID: "ubuntu-nvidia-kernel-module-mismatch"}},
	}, "ubuntu-nvidia-kernel-module-mismatch", OutcomeRequest{Status: "maybe"})
	if err == nil {
		t.Fatal("expected invalid outcome status to be refused")
	}
}

func TestGenerateRefusesNotApplicableCandidate(t *testing.T) {
	service, err := NewService()
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	_, err = service.Generate(incidents.Incident{
		ID: "inc_test",
		RemediationCandidates: []incidents.RemediationCandidate{{
			ID:            "ubuntu-nvidia-kernel-module-mismatch",
			Applicability: "not_applicable",
		}},
	}, "ubuntu-nvidia-kernel-module-mismatch")
	if err == nil {
		t.Fatal("expected not_applicable candidate to be refused")
	}
}

func TestGenerateUsesSecondEvidenceKeyedGenerator(t *testing.T) {
	stateRoot := t.TempDir()
	t.Setenv("VROOLI_STATE_ROOT", stateRoot)
	service, err := NewService()
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	incident := incidents.Incident{
		ID:            "inc_runtime",
		EvidenceItems: []incidents.EvidenceItem{{Kind: "runtime_not_callable", Data: map[string]any{"service": "demo-runtime"}}},
		RemediationCandidates: []incidents.RemediationCandidate{{
			ID: "operator-runtime-restart", TemplateID: "operator-runtime-restart", Title: "Review runtime restart", Applicability: "applicable",
		}},
	}
	response, err := service.Generate(incident, "operator-runtime-restart")
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	data, err := os.ReadFile(filepath.Join(response.Artifact.Path, "metadata.json"))
	if err != nil {
		t.Fatalf("read metadata: %v", err)
	}
	if !strings.Contains(string(data), "demo-runtime") {
		t.Fatalf("metadata does not contain generated runtime evidence: %s", data)
	}
}
