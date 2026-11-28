package incidents

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/api-core/operatordecision"
)

func TestRemediationRequestArrivesAsAPageWithTheScript(t *testing.T) {
	script := filepath.Join(t.TempDir(), "fix-nvidia.sh")
	if err := os.WriteFile(script, []byte("#!/usr/bin/env bash\nset -euo pipefail\nmodprobe nvidia && echo '<loaded>'\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	incident := Incident{
		ID: "inc-42", Fingerprint: "fp-42", Title: "NVIDIA kernel module missing after boot", Severity: SeverityCritical,
		Summary: "The GPU driver did not load after the last reboot, so local models run on the CPU.", Diagnosis: "The nvidia module is not in the running kernel's module list.",
		Confidence: "high", ObservationCount: 7, RemediationDecisionGeneration: 2,
		EvidenceItems:        []EvidenceItem{{Summary: "lsmod has no nvidia entry", Source: "gpu-check", Severity: SeverityCritical}},
		RemediationArtifacts: []RemediationArtifact{{ID: "art-1", RemediationID: "candidate-x", Path: script}},
	}
	candidate := RemediationCandidate{
		ID: "candidate-x", Title: "Load the NVIDIA kernel module", RiskLevel: "moderate", RequiresPrivilege: true, Applicability: "applicable",
		PreflightChecks: []string{"Secure Boot state is known"}, PostChecks: []string{"nvidia-smi lists the GPU"}, RollbackOrFallback: []string{"Unload the module"},
		DecisionPrompt: "Approve loading the module?",
	}
	event := BuildRemediationDecisionEvent(incident, candidate, 3, "digest-42", time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC), time.Date(2026, 10, 18, 12, 0, 0, 0, time.UTC))
	if problems := operatordecision.CoverProblems(remediationDecisionFacts(incident, candidate, time.Date(2026, 10, 18, 12, 0, 0, 0, time.UTC))); len(problems) > 0 {
		t.Errorf("a cold reader cannot parse the cover: %v", problems)
	}
	question, _ := event.Payload["question"].(string)
	if question != "Autoheal found “NVIDIA kernel module missing after boot” and proposes “Load the NVIDIA kernel module” (moderate risk). Run it?" {
		t.Errorf("question = %q", question)
	}
	bigPicture, _ := event.Payload["big_picture"].(string)
	if !strings.Contains(bigPicture, "administrator rights") || !strings.Contains(bigPicture, "12:00 UTC on 18 Oct") || strings.Contains(bigPicture, script) {
		t.Errorf("big picture = %q", bigPicture)
	}
	page, _ := event.Payload["explainer_html"].(string)
	for _, want := range []string{
		"The GPU driver did not load", "7 times", "lsmod has no nvidia entry", // what autoheal saw
		"Administrator rights", "Secure Boot state is known", "nvidia-smi lists the GPU", "Unload the module", // the fix and its checks
		"modprobe nvidia &amp;&amp; echo &#39;&lt;loaded&gt;&#39;", // the script, escaped
		`data-od-answer="approve"`, "If you do not answer:",
	} {
		if !strings.Contains(page, want) {
			t.Errorf("page is missing %q", want)
		}
	}
	details, _ := event.Payload["technical_details"].(string)
	if !strings.Contains(details, script) || !strings.Contains(details, "Approve loading the module?") {
		t.Errorf("technical details = %q", details)
	}
	if len(page) > operatordecision.MaxExplainerBytes {
		t.Fatalf("page is %d bytes", len(page))
	}
}
