package validationmatrix

import (
	"testing"

	"github.com/stretchr/testify/require"
	domainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain"
)

func TestEvaluateReleaseGateBlocksMissingRequiredCoverage(t *testing.T) {
	gate := EvaluateReleaseGate(&domainv1.ValidationMatrix{MatrixId: "matrix-1"})
	require.False(t, gate.GetPassed())
	require.Equal(t, domainv1.ValidationDisposition_VALIDATION_DISPOSITION_NOT_RUN, gate.GetDisposition())
}

func TestEvaluateReleaseGateRequiresRedactedChecksummedEvidence(t *testing.T) {
	cell := passingCell()
	cell.Evidence[0].Redacted = false
	gate := EvaluateReleaseGate(&domainv1.ValidationMatrix{MatrixId: "matrix-1", Cells: []*domainv1.ValidationCell{cell}})
	require.False(t, gate.GetPassed())
	require.Equal(t, []string{"cell-1"}, gate.GetFailedCellIds())
}

func TestEvaluateReleaseGatePassesCompleteRequiredApplicableCells(t *testing.T) {
	cell := passingCell()
	gate := EvaluateReleaseGate(&domainv1.ValidationMatrix{MatrixId: "matrix-1", Cells: []*domainv1.ValidationCell{cell}})
	require.True(t, gate.GetPassed())
	require.Equal(t, domainv1.ValidationDisposition_VALIDATION_DISPOSITION_PASS, gate.GetDisposition())
	require.Equal(t, int32(1), gate.GetPassingCellCount())
}

func TestEvaluateReleaseGateBlocksOmittedSelectedCell(t *testing.T) {
	cell := passingCell()
	cell.JourneyId = "journey-1"
	cell.TargetId = "target-1"
	cell.EnvironmentProfile = domainv1.ValidationEnvironmentProfile_VALIDATION_ENVIRONMENT_PROFILE_NORMAL
	matrix := &domainv1.ValidationMatrix{
		MatrixId: "matrix-1",
		Journeys: []*domainv1.JourneyCatalogItem{
			{JourneyId: "journey-1", Required: true},
			{JourneyId: "journey-2", Required: true},
		},
		Targets: []*domainv1.ValidationTargetDescriptor{{TargetId: "target-1"}},
		EnvironmentProfiles: []domainv1.ValidationEnvironmentProfile{
			domainv1.ValidationEnvironmentProfile_VALIDATION_ENVIRONMENT_PROFILE_NORMAL,
		},
		Cells: []*domainv1.ValidationCell{cell},
	}

	gate := EvaluateReleaseGate(matrix)
	require.False(t, gate.GetPassed())
	require.Contains(t, gate.GetMissingCellIds(), "journey=journey-2 profile=VALIDATION_ENVIRONMENT_PROFILE_NORMAL")
}

func TestEvaluateReleaseGateAllowsAnyCapableTargetToSatisfyChapter(t *testing.T) {
	journey := &domainv1.JourneyCatalogItem{JourneyId: "offline", Required: true, RequiredCapabilities: []domainv1.ValidationTargetCapability{domainv1.ValidationTargetCapability_VALIDATION_TARGET_CAPABILITY_OFFLINE_NETWORK}}
	profile := domainv1.ValidationEnvironmentProfile_VALIDATION_ENVIRONMENT_PROFILE_NORMAL
	emulator := &domainv1.ValidationTargetDescriptor{TargetId: "emulator", Available: true, Capabilities: []domainv1.ValidationTargetCapability{domainv1.ValidationTargetCapability_VALIDATION_TARGET_CAPABILITY_OFFLINE_NETWORK}}
	phone := &domainv1.ValidationTargetDescriptor{TargetId: "galaxy-a03s", Available: true}
	cell := passingStructuredCell("offline", "emulator", profile)
	matrix := &domainv1.ValidationMatrix{MatrixId: "matrix-1", Journeys: []*domainv1.JourneyCatalogItem{journey}, Targets: []*domainv1.ValidationTargetDescriptor{emulator, phone}, EnvironmentProfiles: []domainv1.ValidationEnvironmentProfile{profile}, Cells: []*domainv1.ValidationCell{cell}}

	gate := EvaluateReleaseGate(matrix)
	require.True(t, gate.GetPassed())
	require.Equal(t, "emulator", gate.GetSatisfyingTargetIds()["journey=offline profile=VALIDATION_ENVIRONMENT_PROFILE_NORMAL"])
}

func TestEvaluateReleaseGateFailsWhenNoTargetAdvertisesRequiredCapability(t *testing.T) {
	journey := &domainv1.JourneyCatalogItem{JourneyId: "offline", Required: true, RequiredCapabilities: []domainv1.ValidationTargetCapability{domainv1.ValidationTargetCapability_VALIDATION_TARGET_CAPABILITY_OFFLINE_NETWORK}}
	profile := domainv1.ValidationEnvironmentProfile_VALIDATION_ENVIRONMENT_PROFILE_NORMAL
	matrix := &domainv1.ValidationMatrix{MatrixId: "matrix-1", Journeys: []*domainv1.JourneyCatalogItem{journey}, Targets: []*domainv1.ValidationTargetDescriptor{{TargetId: "galaxy-a03s", Available: true}}, EnvironmentProfiles: []domainv1.ValidationEnvironmentProfile{profile}}

	gate := EvaluateReleaseGate(matrix)
	require.False(t, gate.GetPassed())
	require.Contains(t, gate.GetMissingCellIds(), "journey=offline profile=VALIDATION_ENVIRONMENT_PROFILE_NORMAL")
}

func passingStructuredCell(journeyID, targetID string, profile domainv1.ValidationEnvironmentProfile) *domainv1.ValidationCell {
	return &domainv1.ValidationCell{
		CellId: "cell-" + targetID, JourneyId: journeyID, TargetId: targetID, EnvironmentProfile: profile, Required: true, Applicable: true,
		Disposition: domainv1.ValidationDisposition_VALIDATION_DISPOSITION_PASS,
		Evidence: []*domainv1.LayeredEvidence{
			{EvidenceId: "workflow", Uri: "captures/workflow.json", Sha256: "sha256:workflow", Redacted: true, Kind: domainv1.LayeredEvidence_KIND_BAS_WORKFLOW},
			{EvidenceId: "runtime", Uri: "captures/runtime.json", Sha256: "sha256:runtime", Redacted: true, Kind: domainv1.LayeredEvidence_KIND_DESKTOP_RUNTIME},
			{EvidenceId: "target", Uri: "captures/target.json", Sha256: "sha256:target", Redacted: true, Kind: domainv1.LayeredEvidence_KIND_TARGET},
			{EvidenceId: "machine", Uri: "captures/machine.json", Sha256: "sha256:machine", Redacted: true, Kind: domainv1.LayeredEvidence_KIND_MACHINE_ASSERTION},
		},
	}
}

func passingCell() *domainv1.ValidationCell {
	return &domainv1.ValidationCell{
		CellId: "cell-1", Required: true, Applicable: true,
		Disposition: domainv1.ValidationDisposition_VALIDATION_DISPOSITION_PASS,
		Evidence:    []*domainv1.LayeredEvidence{{EvidenceId: "evidence-1", Uri: "captures/evidence.json", Sha256: "sha256:one", Redacted: true}},
	}
}
