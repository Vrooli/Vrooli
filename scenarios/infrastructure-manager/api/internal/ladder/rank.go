package ladder

import (
	"fmt"

	"github.com/vrooli/api-core/spacedoc"
	internalcondition "github.com/vrooli/vrooli/scenarios/infrastructure-manager/api/internal/condition"
	"github.com/vrooli/vrooli/scenarios/infrastructure-manager/api/internal/focus"
)

// rank orders the ladder's findings by the operating model's cascade: sensor
// channel integrity, then host substrate, then capability availability, then
// efficiency, then measurement improvement. It reuses the focus domain's Stage
// and ranking rather than defining a parallel order, because two ranking
// implementations of one documented cascade will disagree eventually and the
// reader has no way to tell which one is the contract.
//
// Every ranked finding carries the stage explanation, so the output always
// states which cascade stage it applied. A ranking whose order the reader
// cannot see is indistinguishable from a ranking bug.
func rank(snapshot Snapshot) []focus.RankedFinding {
	findings := make([]focus.Finding, 0)

	// Stage 1 — sensor-channel integrity. An unreachable source is the
	// instrument's finding and routes to the instrument's owner. Critically,
	// it produces NO plant-side finding: the plant was never observed, so
	// there is nothing to say about it, and saying anything would be reporting
	// instrument fault as plant fault.
	for _, source := range snapshot.Sources {
		if source.Available {
			continue
		}
		findings = append(findings, focus.Finding{
			ID:             "source-unavailable/" + source.ID,
			Source:         "ladder",
			SensorRef:      source.ID,
			Title:          "source " + source.ID + " could not be read",
			Message:        fmt.Sprintf("%s is unavailable: %s. Every cell it would have graded keeps its authored status; none was downgraded to MISSING, because an owner outage is not a coverage collapse.", source.ID, source.Reason),
			Stage:          focus.StageIntegrity,
			Severity:       severityForStage(focus.StageIntegrity),
			TrustValid:     false,
			ExpectedReturn: "the source becomes readable and its cells grade against their bars",
		})
	}
	if !snapshot.CoverageAvailable {
		findings = append(findings, focus.Finding{
			ID:             "source-unavailable/substrate-space",
			Source:         "ladder",
			SensorRef:      "substrate space document",
			Title:          "the substrate space document could not be read",
			Message:        "the authored cell set is unavailable: " + snapshot.CoverageReason + ". Without it there are no authored statuses to keep, so no ladder cell carries a coverage status at all.",
			Stage:          focus.StageIntegrity,
			Severity:       severityForStage(focus.StageIntegrity),
			TrustValid:     false,
			ExpectedReturn: "the space document is readable and every ladder cell carries an authored status",
		})
	}

	for _, cell := range snapshot.Cells {
		findings = append(findings, cellFindings(cell)...)
	}
	return focus.Rank(findings)
}

// cellFindings turns one graded cell into its findings. A cell contributes to
// at most one plant-side stage, and only when its reading was actually
// believed: an untrusted or unavailable reading never becomes substrate work.
func cellFindings(cell Cell) []focus.Finding {
	out := make([]focus.Finding, 0, 2)

	// Stage 2 — host substrate. Only a trusted reading may raise one.
	if cell.Trust == internalcondition.TrustValid && cell.Band == internalcondition.BandOutOfBand {
		out = append(out, focus.Finding{
			ID:             "substrate-out-of-band/" + cell.CellRef + "/" + cell.Key.String(),
			Source:         "ladder",
			CellRef:        cell.CellRef,
			SensorRef:      cell.Key.String(),
			Title:          fmt.Sprintf("%s is out of band at the %s rung", cell.Key.DeviceClass, cell.Key.Rung),
			Message:        fmt.Sprintf("%d of %d %s could not be graded at the %s rung on %s: %s", cell.BlindDevices, cell.DeviceCount, cell.Key.DeviceClass, cell.Key.Rung, cell.Key.HostOS, cell.Reason),
			Stage:          focus.StageSubstrate,
			Severity:       severityForStage(focus.StageSubstrate),
			TrustValid:     true,
			ExpectedReturn: "every contributing device grades at this rung and the cell returns in band",
		})
	}

	// Stage 3 — capability availability. A capability that resolves nowhere on
	// this host OS is why the cell has no sensor there.
	if cell.CapabilityStatus == "peerless" || cell.CapabilityStatus == "unwired" {
		out = append(out, focus.Finding{
			ID:             "capability-gap/" + cell.Capability + "/" + cell.Key.HostOS,
			Source:         "ladder",
			CellRef:        cell.CellRef,
			SensorRef:      cell.Capability + "/" + cell.Key.HostOS,
			Title:          fmt.Sprintf("capability %s resolves %s on %s", cell.Capability, cell.CapabilityStatus, cell.Key.HostOS),
			Message:        cell.CapabilityReason,
			Stage:          focus.StageAvailability,
			Severity:       severityForStage(focus.StageAvailability),
			TrustValid:     true,
			ExpectedReturn: fmt.Sprintf("an implementation of %s resolves on %s", cell.Capability, cell.Key.HostOS),
		})
	}

	// Stage 5 — measurement improvement. An ungraded cell is a gap in the
	// instrument, ranked last, and it is reported rather than passed over:
	// silently ungraded reads identically to in band.
	//
	// It is not raised for a cell whose bar simply was not evaluated because
	// its reading is untrusted — that is already the integrity finding above,
	// and raising both would double-count one fault under two stages.
	if !cell.Graded && cell.Band != internalcondition.BandNotEvaluated {
		out = append(out, focus.Finding{
			ID:             "ungraded/" + cell.CellRef + "/" + cell.Key.String(),
			Source:         "ladder",
			CellRef:        cell.CellRef,
			SensorRef:      cell.Key.String(),
			Title:          fmt.Sprintf("%s is ungraded", cell.Key.String()),
			Message:        cell.UngradedReason,
			Stage:          focus.StageMeasurement,
			Severity:       severityForStage(focus.StageMeasurement),
			TrustValid:     cell.Trust == internalcondition.TrustValid,
			ExpectedReturn: "a gradeable bar resolves for " + cell.CellRef,
		})
	}
	return out
}

// severityForStage mirrors the focus domain's stage severities so one cascade
// order governs both surfaces.
func severityForStage(stage focus.Stage) int {
	switch stage {
	case focus.StageIntegrity:
		return 10
	case focus.StageSubstrate:
		return 8
	case focus.StageAvailability:
		return 6
	case focus.StageEfficiency:
		return 4
	default:
		return 1
	}
}

// CellStatusToken renders a coverage status for a typed surface, keeping the
// empty status distinguishable from an authored one.
func CellStatusToken(status spacedoc.CellStatus) string {
	if status == "" {
		return "unauthored"
	}
	return string(status)
}
