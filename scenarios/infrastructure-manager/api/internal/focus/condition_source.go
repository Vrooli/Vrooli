package focus

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/vrooli/api-core/spacedoc"
	internalcondition "github.com/vrooli/vrooli/scenarios/infrastructure-manager/api/internal/condition"
)

// ConditionReader is the read side of the condition domain that focus needs.
// Keeping it an interface here means focus depends on the question ("what do
// the readings say") rather than on how condition assembles them.
type ConditionReader interface {
	ReadAll(context.Context) internalcondition.Snapshot
}

// ConditionSource turns live readings into ranked findings. Without it the
// ranked surface could only ever report coverage gaps: the two inner-tier
// sources — a reading outside its band, and a reading that cannot be believed
// — had no join, so an out-of-band substrate severity was invisible while a
// documentation gap ranked first.
type ConditionSource struct{ Condition ConditionReader }

// stageForProjection places a finding on the cascade ladder. Substrate is the
// inner tier and outranks availability, which is why an unreadable machine
// check must not queue behind a coverage gap.
func stageForProjection(projection string) Stage {
	switch projection {
	case string(spacedoc.ProjectionSubstrate):
		return StageSubstrate
	case string(spacedoc.ProjectionValidationCost), string(spacedoc.ProjectionCapacity), string(spacedoc.ProjectionHeadroom):
		return StageEfficiency
	default:
		return StageAvailability
	}
}

func projectionOf(cellRef string) string {
	if index := strings.Index(cellRef, "/"); index > 0 {
		return cellRef[:index]
	}
	return cellRef
}

func (s ConditionSource) Read(ctx context.Context) ([]Finding, []GapSource, error) {
	if s.Condition == nil {
		return nil, []GapSource{
			{ID: "out-of-band", Label: "out-of-band readings", Available: false, Reason: "condition source join is not configured"},
			{ID: "untrusted", Label: "untrusted readings", Available: false, Reason: "condition trust source join is not configured"},
		}, nil
	}
	snapshot := s.Condition.ReadAll(ctx)

	// A source that could not be read makes both of its finding classes
	// unavailable. Reporting zero out-of-band readings from an unread source
	// would be a clean sheet earned by not looking.
	unavailable := make([]string, 0)
	for _, source := range snapshot.Sources {
		if !source.Available {
			unavailable = append(unavailable, fmt.Sprintf("%s: %s", source.Source, source.Reason))
		}
	}

	grouped := make(map[groupKey][]Finding)
	order := make([]groupKey, 0)
	outOfBand, untrusted := 0, 0
	add := func(key groupKey, finding Finding) {
		if _, seen := grouped[key]; !seen {
			order = append(order, key)
		}
		grouped[key] = append(grouped[key], finding)
	}
	for _, reading := range snapshot.Readings {
		projection := projectionOf(reading.CellRef)
		switch {
		case reading.BandVerdict == internalcondition.BandOutOfBand:
			outOfBand++
			add(groupKey{source: "out-of-band", cellRef: reading.CellRef}, Finding{
				ID:      "condition/out-of-band/" + reading.CellRef + "/" + reading.ID,
				Source:  "out-of-band",
				CellRef: reading.CellRef,
				// The sensor, not the cell, is what a fix has to move.
				SensorRef:      reading.ID,
				Title:          fmt.Sprintf("%s is out of band", reading.CellRef),
				Message:        describeExcursion(reading),
				Stage:          stageForProjection(projection),
				Severity:       severityForStage(stageForProjection(projection)),
				TrustValid:     true,
				ExpectedReturn: "IN_BAND",
			})
		case isUntrusted(reading.Trust):
			untrusted++
			// An untrusted reading is a fault in the measuring channel, so it
			// ranks at integrity and routes to the instrument's owner rather
			// than becoming plant work.
			add(groupKey{source: "untrusted", cellRef: reading.CellRef, verdict: string(reading.Trust)}, Finding{
				ID:             "condition/untrusted/" + reading.CellRef + "/" + reading.ID,
				Source:         "untrusted",
				CellRef:        reading.CellRef,
				SensorRef:      reading.ID,
				Title:          untrustedTitle(reading),
				Message:        describeUntrusted(reading),
				Stage:          StageIntegrity,
				Severity:       10,
				TrustValid:     false,
				ExpectedReturn: "VALID",
			})
		}
	}

	findings := make([]Finding, 0, len(order))
	for _, key := range order {
		findings = append(findings, collapse(key, grouped[key])...)
	}

	outOfBandSource := GapSource{ID: "out-of-band", Label: "out-of-band readings", Available: true, FindingCount: outOfBand}
	untrustedSource := GapSource{ID: "untrusted", Label: "untrusted readings", Available: true, FindingCount: untrusted}
	if len(unavailable) > 0 {
		reason := "some condition sources are unreadable: " + strings.Join(unavailable, "; ")
		outOfBandSource.Available, outOfBandSource.Reason = false, reason
		untrustedSource.Available, untrustedSource.Reason = false, reason
	}
	return findings, []GapSource{outOfBandSource, untrustedSource}, nil
}

// groupKey identifies findings that say the same thing about the same cell.
type groupKey struct {
	source  string
	cellRef string
	verdict string
}

// floodThreshold is the point at which repeating one finding per sensor stops
// informing and starts burying. ISA-18.2 alarm hygiene is explicit that a
// repeated identical alarm carries no extra information — and the instrument
// producing its own flood while reporting on alarm floods would be absurd.
const floodThreshold = 3

// collapse emits one finding per contributing sensor while a group is small
// enough to stay readable, and a single counted finding once it floods. The
// aggregate names its worst contributors so it stays actionable, and always
// reports the full count so nothing is silently dropped.
func collapse(key groupKey, group []Finding) []Finding {
	if len(group) <= floodThreshold {
		return group
	}
	sensors := make([]string, 0, len(group))
	for _, finding := range group {
		sensors = append(sensors, finding.SensorRef)
	}
	sort.Strings(sensors)
	shown := sensors
	if len(shown) > floodThreshold {
		shown = shown[:floodThreshold]
	}
	title := fmt.Sprintf("%d readings on %s are out of band", len(group), key.cellRef)
	if key.source == "untrusted" {
		title = fmt.Sprintf("%d checks on %s carry verdict %s", len(group), key.cellRef, key.verdict)
		if key.verdict == string(internalcondition.TrustSaturated) {
			title = fmt.Sprintf("%d checks on %s are stuck in a non-normal state", len(group), key.cellRef)
		}
	}
	return []Finding{{
		ID:      fmt.Sprintf("condition/%s/%s/aggregate", key.source, key.cellRef),
		Source:  key.source,
		CellRef: key.cellRef,
		// The aggregate has no single sensor to re-read, so efficacy is
		// measured against the cell rather than against one check.
		SensorRef:      key.cellRef,
		Title:          title,
		Message:        fmt.Sprintf("%d contributing sensors, including %s. Read them all with `infrastructure-manager condition status --cell %s`.", len(group), strings.Join(shown, ", "), key.cellRef),
		Stage:          group[0].Stage,
		Severity:       group[0].Severity,
		TrustValid:     group[0].TrustValid,
		ExpectedReturn: group[0].ExpectedReturn,
	}}
}

func isUntrusted(verdict internalcondition.TrustVerdict) bool {
	switch verdict {
	case internalcondition.TrustValid, internalcondition.TrustShelved:
		// A shelved check is deliberately stopped, not a broken sensor.
		return false
	default:
		return true
	}
}

func describeExcursion(reading internalcondition.Observation) string {
	bound := "its bar"
	switch {
	case reading.Band.Max != nil && reading.Value > *reading.Band.Max:
		bound = fmt.Sprintf("a maximum of %g", *reading.Band.Max)
	case reading.Band.Min != nil && reading.Value < *reading.Band.Min:
		bound = fmt.Sprintf("a minimum of %g", *reading.Band.Min)
	}
	message := fmt.Sprintf("%s read %g %s against %s (sensor %s).", reading.CellRef, reading.Value, reading.Unit, bound, reading.ID)
	if reading.Band.Provisional {
		message += " The bar is provisional and awaiting operator ratification."
	}
	return message
}

// untrustedTitle distinguishes a sensor that could not be read from one that
// is stuck. A saturated check is not unbelievable — it is pinned in a
// non-normal state, which is real information about the plant. Calling that
// "cannot be believed" would tell a reader to doubt the very signal that is
// reporting a fault.
func untrustedTitle(reading internalcondition.Observation) string {
	if reading.Trust == internalcondition.TrustSaturated {
		return fmt.Sprintf("%s is stuck in a non-normal state (no transition in the window)", reading.ID)
	}
	return fmt.Sprintf("%s cannot be believed (%s)", reading.ID, reading.Trust)
}

func describeUntrusted(reading internalcondition.Observation) string {
	if reading.UnavailableReason != "" {
		return fmt.Sprintf("%s: %s", reading.Trust, reading.UnavailableReason)
	}
	if reading.Trust == internalcondition.TrustSaturated {
		return fmt.Sprintf(
			"%s has held one non-normal status for the whole window, so its repeat events carry no new information and it is excluded from aggregates. Per alarm-hygiene discipline it converts to exactly one durable incident, then is shelved with an expiry or retired. Check `vrooli-autoheal incidents latest` for the incident, and read the underlying signal with `infrastructure-manager condition explain %s`.",
			reading.ID, reading.CellRef)
	}
	return fmt.Sprintf("%s carries verdict %s, so it contributes to no aggregate.", reading.ID, reading.Trust)
}

// severityForStage keeps ranking consistent with the cascade order rather than
// letting each source pick its own scale.
func severityForStage(stage Stage) int {
	switch stage {
	case StageIntegrity:
		return 10
	case StageSubstrate:
		return 8
	case StageAvailability:
		return 6
	case StageEfficiency:
		return 4
	default:
		return 1
	}
}

// MergedSource composes the coverage and condition joins into the single
// ranked surface a member reads with one call.
type MergedSource struct{ Sources []Source }

func (m MergedSource) Read(ctx context.Context) ([]Finding, []GapSource, error) {
	findings := make([]Finding, 0)
	gaps := make([]GapSource, 0)
	for _, source := range m.Sources {
		sourceFindings, sourceGaps, err := source.Read(ctx)
		if err != nil {
			return nil, nil, err
		}
		findings = append(findings, sourceFindings...)
		gaps = append(gaps, sourceGaps...)
	}
	return findings, gaps, nil
}
