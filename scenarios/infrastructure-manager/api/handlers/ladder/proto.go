package ladder

import (
	"strings"

	ladderv1 "github.com/vrooli/vrooli/packages/proto/gen/go/infrastructure-manager/v1/ladder"
	internalcondition "github.com/vrooli/vrooli/scenarios/infrastructure-manager/api/internal/condition"
	"github.com/vrooli/vrooli/scenarios/infrastructure-manager/api/internal/focus"
	internalladder "github.com/vrooli/vrooli/scenarios/infrastructure-manager/api/internal/ladder"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Every projection below resolves through the generated enum value table
// rather than a hand-maintained switch, so adding a token to the vocabulary is
// a proto edit and nothing else. A parallel switch returns the zero value on a
// miss, and an unlabelled enum value is indistinguishable from a deliberate
// UNSPECIFIED.
func protoEnum[T ~int32](values map[string]int32, prefix, token string) T {
	name := prefix + strings.ToUpper(strings.ReplaceAll(token, "-", "_"))
	if value, ok := values[name]; ok {
		return T(value)
	}
	var unspecified T
	return unspecified
}

func protoRung(rung internalladder.Rung) ladderv1.Rung {
	return protoEnum[ladderv1.Rung](ladderv1.Rung_value, "RUNG_", string(rung))
}

func protoObservation(observation internalladder.Observation) ladderv1.Observation {
	return protoEnum[ladderv1.Observation](ladderv1.Observation_value, "OBSERVATION_", string(observation))
}

func protoCellStatus(status string) ladderv1.CellStatus {
	return protoEnum[ladderv1.CellStatus](ladderv1.CellStatus_value, "CELL_STATUS_", status)
}

func protoTrust(trust internalcondition.TrustVerdict) ladderv1.TrustVerdict {
	return protoEnum[ladderv1.TrustVerdict](ladderv1.TrustVerdict_value, "TRUST_VERDICT_", string(trust))
}

func protoBand(band internalcondition.BandVerdict) ladderv1.BandVerdict {
	return protoEnum[ladderv1.BandVerdict](ladderv1.BandVerdict_value, "BAND_VERDICT_", string(band))
}

// protoStage maps the focus domain's cascade stage onto the wire. The focus
// Stage is an ordered int rather than a token, so this is the one place a
// switch is correct: it is the seam between two representations of the same
// documented cascade, not a parallel copy of a vocabulary.
func protoStage(stage focus.Stage) ladderv1.CascadeStage {
	switch stage {
	case focus.StageIntegrity:
		return ladderv1.CascadeStage_CASCADE_STAGE_SENSOR_CHANNEL_INTEGRITY
	case focus.StageSubstrate:
		return ladderv1.CascadeStage_CASCADE_STAGE_HOST_SUBSTRATE
	case focus.StageAvailability:
		return ladderv1.CascadeStage_CASCADE_STAGE_CAPABILITY_AVAILABILITY
	case focus.StageEfficiency:
		return ladderv1.CascadeStage_CASCADE_STAGE_EFFICIENCY
	case focus.StageMeasurement:
		return ladderv1.CascadeStage_CASCADE_STAGE_MEASUREMENT_IMPROVEMENT
	default:
		return ladderv1.CascadeStage_CASCADE_STAGE_UNSPECIFIED
	}
}

func protoCell(cell internalladder.Cell) *ladderv1.LadderCell {
	return &ladderv1.LadderCell{
		DeviceClass:       cell.Key.DeviceClass,
		Rung:              protoRung(cell.Key.Rung),
		HostOs:            cell.Key.HostOS,
		Key:               cell.Key.String(),
		CellRef:           cell.CellRef,
		Question:          cell.Question,
		Status:            protoCellStatus(internalladder.CellStatusToken(cell.Status)),
		StatusSource:      cell.StatusSource,
		Observation:       protoObservation(cell.Observation),
		Reason:            cell.Reason,
		Mechanism:         cell.Mechanism,
		Remediation:       cell.Remediation,
		BlockedBy:         protoRung(cell.BlockedBy),
		Trust:             protoTrust(cell.Trust),
		UnavailableReason: cell.UnavailableReason,
		DeviceCount:       int32(cell.DeviceCount),
		BlindDevices:      int32(cell.BlindDevices),
		BarId:             cell.BarID,
		Graded:            cell.Graded,
		UngradedReason:    cell.UngradedReason,
		Band:              protoBand(cell.Band),
		Provisional:       cell.Provisional,
		Capability:        cell.Capability,
		CapabilityStatus:  cell.CapabilityStatus,
		CapabilityReason:  cell.CapabilityReason,
		ObservedAt:        timestamppb.New(cell.ObservedAt),
		FaultUnit:         cell.FaultUnit,
		FaultCount:        cell.FaultCount,
		FaultCounted:      cell.FaultCounted,
		Severity:          int32(cell.Severity),
		SeverityKnown:     cell.SeverityKnown,
		GapOpenedOn:       cell.GapOpenedOn,
		GapOpenDays:       int32(cell.GapOpenDays),
		GapDated:          cell.GapDated,
	}
}

func protoConfidence(confidence internalladder.Confidence) *ladderv1.Confidence {
	return &ladderv1.Confidence{
		Level:     protoEnum[ladderv1.ConfidenceLevel](ladderv1.ConfidenceLevel_value, "CONFIDENCE_LEVEL_", confidence.Level),
		Rationale: confidence.Rationale,
		Available: confidence.Available,
		Reason:    confidence.Reason,
	}
}

func protoDeviceRung(rung internalladder.DeviceRung) *ladderv1.DeviceRung {
	return &ladderv1.DeviceRung{
		Rung:              protoRung(rung.Rung),
		Observation:       protoObservation(rung.Observation),
		LadderObservation: protoObservation(rung.LadderObservation),
		Reason:            rung.Reason,
		Mechanism:         rung.Mechanism,
		Remediation:       rung.Remediation,
		BlockedBy:         protoRung(rung.BlockedBy),
	}
}

func protoDevice(device internalladder.Device) *ladderv1.Device {
	out := &ladderv1.Device{
		Id: device.ID, Class: device.Class, ParentId: device.ParentID,
		Vendor: device.Vendor, Model: device.Model, Driver: device.Driver,
		SysPath: device.SysPath, Attributes: device.Attributes, Readings: device.Readings,
		Rungs: make([]*ladderv1.DeviceRung, 0, len(device.Rungs)),
	}
	for _, rung := range device.Rungs {
		out.Rungs = append(out.Rungs, protoDeviceRung(rung))
	}
	return out
}

func protoCheckPlatform(coverage internalladder.CheckPlatformCoverage) *ladderv1.CheckPlatformCoverage {
	return &ladderv1.CheckPlatformCoverage{
		HostOs:     coverage.HostOS,
		Applicable: int32(coverage.Applicable),
		Total:      int32(coverage.Total),
		Universal:  int32(coverage.Universal),
		Available:  coverage.Available,
		Reason:     coverage.Reason,
	}
}

func protoSource(source internalladder.SourceState) *ladderv1.SourceState {
	return &ladderv1.SourceState{
		Id:        source.ID,
		Available: source.Available,
		Reason:    source.Reason,
		Trust:     protoTrust(source.Trust),
		CheckedAt: timestamppb.New(source.CheckedAt),
	}
}

func protoFinding(finding focus.RankedFinding) *ladderv1.RankedFinding {
	return &ladderv1.RankedFinding{
		Rank:             int32(finding.Rank),
		Id:               finding.ID,
		Source:           finding.Source,
		CellRef:          finding.CellRef,
		SensorRef:        finding.SensorRef,
		Title:            finding.Title,
		Message:          finding.Message,
		Stage:            protoStage(finding.Stage),
		StageExplanation: finding.RankExplanation,
		Severity:         int32(finding.Severity),
		TrustValid:       finding.TrustValid,
		ExpectedReturn:   finding.ExpectedReturn,
	}
}

func protoLadder(snapshot internalladder.Snapshot) *ladderv1.Ladder {
	out := &ladderv1.Ladder{
		HostOs:            snapshot.HostOS,
		CoverageAvailable: snapshot.CoverageAvailable,
		CoverageReason:    snapshot.CoverageReason,
		ComputedAt:        timestamppb.New(snapshot.ComputedAt),
		Cells:             make([]*ladderv1.LadderCell, 0, len(snapshot.Cells)),
		Sources:           make([]*ladderv1.SourceState, 0, len(snapshot.Sources)),
		Findings:          make([]*ladderv1.RankedFinding, 0, len(snapshot.Findings)),
		CheckPlatforms:    make([]*ladderv1.CheckPlatformCoverage, 0, len(snapshot.CheckPlatforms)),
		Devices:           make([]*ladderv1.Device, 0, len(snapshot.Devices)),
		Confidence:        protoConfidence(snapshot.Confidence),
	}
	for _, device := range snapshot.Devices {
		out.Devices = append(out.Devices, protoDevice(device))
	}
	for _, coverage := range snapshot.CheckPlatforms {
		out.CheckPlatforms = append(out.CheckPlatforms, protoCheckPlatform(coverage))
	}
	for _, cell := range snapshot.Cells {
		out.Cells = append(out.Cells, protoCell(cell))
	}
	for _, source := range snapshot.Sources {
		out.Sources = append(out.Sources, protoSource(source))
	}
	for _, finding := range snapshot.Findings {
		out.Findings = append(out.Findings, protoFinding(finding))
	}
	return out
}
