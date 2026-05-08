package remediation

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"vrooli-autoheal/internal/incidents"

	"github.com/vrooli/api-core/storage"
)

const scenarioID = "vrooli-autoheal"

type Service struct {
	resolver *storage.Resolver
	now      func() time.Time
}

type GenerateResponse struct {
	IncidentID string                         `json:"incidentId"`
	Candidate  incidents.RemediationCandidate `json:"candidate"`
	Artifact   incidents.RemediationArtifact  `json:"artifact"`
	Files      map[string]string              `json:"files"`
	PostChecks []string                       `json:"postChecks"`
}

type OutcomeRequest struct {
	Status string `json:"status"`
	Note   string `json:"note,omitempty"`
}

func NewService() (*Service, error) {
	resolver, err := storage.NewResolver(storage.ResolverConfig{AppID: "vrooli", Profile: storage.ProfileAuto})
	if err != nil {
		return nil, err
	}
	return &Service{resolver: resolver, now: func() time.Time { return time.Now().UTC() }}, nil
}

func (s *Service) Candidates(incident incidents.Incident) []incidents.RemediationCandidate {
	return incident.RemediationCandidates
}

func (s *Service) Generate(incident incidents.Incident, remediationID string) (*GenerateResponse, error) {
	candidate, ok := findCandidate(incident.RemediationCandidates, remediationID)
	if !ok {
		return nil, fmt.Errorf("remediation candidate %q not found", remediationID)
	}
	if candidate.Applicability != "applicable" {
		return nil, fmt.Errorf("remediation candidate %q is %s", remediationID, candidate.Applicability)
	}
	if candidate.ID != "ubuntu-nvidia-kernel-module-mismatch" {
		return nil, fmt.Errorf("remediation candidate %q has no generator", remediationID)
	}
	facts, err := nvidiaFacts(incident)
	if err != nil {
		return nil, err
	}
	rel := filepath.Join("incidents", safePathPart(incident.ID), "remediation", safePathPart(candidate.ID))
	base, err := s.resolver.Path(storage.Options{ScenarioID: scenarioID}, storage.ClassState, rel)
	if err != nil {
		return nil, err
	}
	files := map[string]string{
		"remediation.sh":   filepath.Join(base, "remediation.sh"),
		"metadata.json":    filepath.Join(base, "metadata.json"),
		"README.md":        filepath.Join(base, "README.md"),
		"post-checks.json": filepath.Join(base, "post-checks.json"),
	}
	generatedAt := s.now()
	metadata := map[string]any{
		"generatedAt":           generatedAt.Format(time.RFC3339Nano),
		"sourceIncidentId":      incident.ID,
		"sourceFingerprint":     incident.Fingerprint,
		"templateId":            candidate.TemplateID,
		"templateVersion":       "1",
		"platformApplicability": candidate.Applicability,
		"expectedPackage":       facts.ExpectedPackage,
		"runningKernel":         facts.RunningKernel,
		"commands":              []string{"apt-get update", "apt-get -s install <expected package>", "apt-get install <expected package>"},
		"preflightCommands":     []string{"uname -r", "apt-cache policy <expected package>", "dpkg-query -W <expected package>"},
		"postChecks":            candidate.PostChecks,
		"operatorApproval":      "required",
	}
	metadataJSON, _ := json.MarshalIndent(metadata, "", "  ")
	postChecksJSON, _ := json.MarshalIndent(candidate.PostChecks, "", "  ")
	writes := map[string][]byte{
		files["remediation.sh"]:   []byte(scriptForNVIDIAModule(candidate, facts)),
		files["metadata.json"]:    metadataJSON,
		files["README.md"]:        []byte(readmeForCandidate(incident, candidate, facts)),
		files["post-checks.json"]: postChecksJSON,
	}
	for path, data := range writes {
		perm := storage.DefaultFilePerm
		if strings.HasSuffix(path, ".sh") {
			perm = 0o755
		}
		if err := storage.WriteFileAtomic(path, data, perm); err != nil {
			return nil, err
		}
	}
	return &GenerateResponse{
		IncidentID: incident.ID,
		Candidate:  candidate,
		Artifact: incidents.RemediationArtifact{
			ID:            candidate.ID + "-artifact",
			RemediationID: candidate.ID,
			Path:          base,
			GeneratedAt:   generatedAt,
			Metadata:      metadata,
		},
		Files:      files,
		PostChecks: candidate.PostChecks,
	}, nil
}

func (s *Service) Outcome(incident incidents.Incident, remediationID string, req OutcomeRequest) (incidents.Outcome, error) {
	if _, ok := findCandidate(incident.RemediationCandidates, remediationID); !ok {
		return incidents.Outcome{}, fmt.Errorf("remediation candidate %q not found", remediationID)
	}
	status := strings.TrimSpace(req.Status)
	if !validOutcomeStatus(status) {
		return incidents.Outcome{}, fmt.Errorf("invalid outcome status %q", req.Status)
	}
	outcome := incidents.Outcome{
		RemediationID: remediationID,
		Status:        status,
		Note:          strings.TrimSpace(req.Note),
		ReportedAt:    s.now(),
	}
	_ = updateArtifactMetadataOutcome(incident.RemediationArtifacts, remediationID, outcome)
	return outcome, nil
}

type nvidiaModuleFacts struct {
	ExpectedPackage string
	RunningKernel   string
}

func nvidiaFacts(incident incidents.Incident) (nvidiaModuleFacts, error) {
	for _, item := range incident.EvidenceItems {
		if item.Kind != "missing_nvidia_module_package" {
			continue
		}
		return nvidiaModuleFacts{
			ExpectedPackage: fmt.Sprintf("%v", item.Data["expectedPackage"]),
			RunningKernel:   fmt.Sprintf("%v", item.Data["runningKernel"]),
		}, nil
	}
	return nvidiaModuleFacts{}, fmt.Errorf("incident does not contain missing NVIDIA module package evidence")
}

func scriptForNVIDIAModule(candidate incidents.RemediationCandidate, facts nvidiaModuleFacts) string {
	return fmt.Sprintf(`#!/usr/bin/env bash
set -euo pipefail

EXPECTED_PACKAGE=%q
EXPECTED_KERNEL=%q

if [[ "${EUID}" -ne 0 ]]; then
  echo "Run this script with sudo." >&2
  exit 1
fi

RUNNING_KERNEL="$(uname -r)"
if [[ "${RUNNING_KERNEL}" != "${EXPECTED_KERNEL}" ]]; then
  echo "Running kernel changed: expected ${EXPECTED_KERNEL}, got ${RUNNING_KERNEL}." >&2
  exit 1
fi

if ! command -v apt-get >/dev/null 2>&1 || ! command -v apt-cache >/dev/null 2>&1; then
  echo "apt-get and apt-cache are required for this remediation." >&2
  exit 1
fi

if dpkg-query -W -f='${Status}' "${EXPECTED_PACKAGE}" 2>/dev/null | grep -q "install ok installed"; then
  echo "${EXPECTED_PACKAGE} is already installed."
  exit 0
fi

if apt-cache policy "${EXPECTED_PACKAGE}" | grep -q "Candidate: (none)"; then
  echo "No apt candidate found for ${EXPECTED_PACKAGE}." >&2
  exit 1
fi

apt-get update
apt-get -s install "${EXPECTED_PACKAGE}"

printf "Install ${EXPECTED_PACKAGE} now? Type yes to continue: "
read -r CONFIRM
if [[ "${CONFIRM}" != "yes" ]]; then
  echo "Aborted without changes."
  exit 1
fi

apt-get install -y "${EXPECTED_PACKAGE}"

echo "Install complete. Reboot when ready, then run these checks:"
echo "  nvidia-smi"
echo "  lsmod | grep '^nvidia'"
echo "  vrooli-autoheal incidents latest --json"
`, facts.ExpectedPackage, facts.RunningKernel)
}

func readmeForCandidate(incident incidents.Incident, candidate incidents.RemediationCandidate, facts nvidiaModuleFacts) string {
	return fmt.Sprintf(`# %s

Incident: %s
Fingerprint: %s
Expected package: %s
Running kernel: %s
Risk: %s

This artifact was generated for operator review. Autoheal does not execute it.

Run with:

`+"```bash"+`
sudo %s
`+"```"+`

Post-checks:

- %s
`, candidate.Title, incident.ID, incident.Fingerprint, facts.ExpectedPackage, facts.RunningKernel, candidate.RiskLevel, "remediation.sh", strings.Join(candidate.PostChecks, "\n- "))
}

func findCandidate(candidates []incidents.RemediationCandidate, id string) (incidents.RemediationCandidate, bool) {
	for _, candidate := range candidates {
		if candidate.ID == id {
			return candidate, true
		}
	}
	return incidents.RemediationCandidate{}, false
}

func validOutcomeStatus(status string) bool {
	switch status {
	case "generated", "operator_ran", "verified", "failed", "abandoned":
		return true
	default:
		return false
	}
}

func updateArtifactMetadataOutcome(artifacts []incidents.RemediationArtifact, remediationID string, outcome incidents.Outcome) error {
	for _, artifact := range artifacts {
		if artifact.RemediationID != remediationID || strings.TrimSpace(artifact.Path) == "" {
			continue
		}
		metadataPath := filepath.Join(artifact.Path, "metadata.json")
		data, err := os.ReadFile(metadataPath)
		if err != nil {
			return err
		}
		var metadata map[string]any
		if err := json.Unmarshal(data, &metadata); err != nil {
			return err
		}
		metadata["outcome"] = outcome
		updated, err := json.MarshalIndent(metadata, "", "  ")
		if err != nil {
			return err
		}
		return storage.WriteFileAtomic(metadataPath, updated, storage.DefaultFilePerm)
	}
	return nil
}

func safePathPart(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "unknown"
	}
	var b strings.Builder
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_', r == '.':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	return b.String()
}
