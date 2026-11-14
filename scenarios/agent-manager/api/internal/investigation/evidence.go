package investigation

import "fmt"

// ProjectionObservation describes the source identity and watermark observed
// by one bounded read. It is deliberately smaller than a report: adapters can
// populate it from events, receipts, or another projection without changing
// the investigation contract.
type ProjectionObservation struct {
	Plane             string
	SourceGeneration  string
	ClassifierVersion string
	ThroughRef        string
	ThroughSequence   int64
	Records           []ProjectedRecord
}

type ProjectedRecord struct {
	Ref      string
	Sequence int64
	Value    string
}

type ProjectionCut struct {
	Plane             string
	SourceGeneration  string
	ClassifierVersion string
	ThroughRef        string
	ThroughSequence   int64
}

type ProjectionRead struct {
	Coverage         CoveragePlane
	Records          []ProjectedRecord
	ExcludedAfterCut []ProjectedRecord
}

// ReadProjectionAtCut prevents a current, ahead-of-cut projection from
// contaminating a historical diagnosis. It also makes lag, schema mismatch,
// and an empty-but-healthy projection distinct from unavailable evidence.
func ReadProjectionAtCut(cut ProjectionCut, observed ProjectionObservation) ProjectionRead {
	read := ProjectionRead{Coverage: CoveragePlane{Plane: cut.Plane, State: CoverageUnknown}}
	if cut.Plane == "" || observed.Plane != cut.Plane {
		read.Coverage.Reason = "projection plane identity does not match the requested cut"
		return read
	}
	if cut.SourceGeneration != "" && observed.SourceGeneration != cut.SourceGeneration {
		read.Coverage.State = CoverageIncompatible
		read.Coverage.Reason = fmt.Sprintf("source generation %q does not match cut %q", observed.SourceGeneration, cut.SourceGeneration)
		return read
	}
	if cut.ClassifierVersion != "" && observed.ClassifierVersion != cut.ClassifierVersion {
		read.Coverage.State = CoverageIncompatible
		read.Coverage.Reason = fmt.Sprintf("classifier version %q does not match cut %q", observed.ClassifierVersion, cut.ClassifierVersion)
		return read
	}
	if observed.ThroughSequence < cut.ThroughSequence {
		read.Coverage.State = CoverageLagging
		read.Coverage.ThroughRef = observed.ThroughRef
		read.Coverage.Reason = "projection watermark is behind the evidence cut"
		return read
	}
	read.Coverage.State = CoverageComplete
	read.Coverage.ThroughRef = cut.ThroughRef
	for _, record := range observed.Records {
		if record.Sequence <= cut.ThroughSequence {
			read.Records = append(read.Records, record)
		} else {
			read.ExcludedAfterCut = append(read.ExcludedAfterCut, record)
		}
	}
	if observed.ThroughSequence > cut.ThroughSequence {
		read.Coverage.Reason = "projection was ahead and was truncated at the evidence cut"
	}
	return read
}

type ReconciliationResult struct {
	Observation     ProjectionObservation
	Coverage        CoveragePlane
	Identities      []string
	Reconciliations int
}

// ReconcileObservations models the owner adapter's bounded retry policy. A
// changing source is retained as partial evidence after the allowance is
// exhausted; callers never spin until the source becomes quiet.
func ReconcileObservations(observations []ProjectionObservation, maxReconciliations int) ReconciliationResult {
	if len(observations) == 0 {
		return ReconciliationResult{Coverage: CoveragePlane{State: CoverageUnknown, Reason: "no projection observation was available"}}
	}
	if maxReconciliations < 0 {
		maxReconciliations = 0
	}
	result := ReconciliationResult{Observation: observations[0], Coverage: CoveragePlane{Plane: observations[0].Plane, State: CoverageComplete, ThroughRef: observations[0].ThroughRef}}
	seen := map[string]struct{}{}
	addIdentity := func(observation ProjectionObservation) {
		identity := observation.SourceGeneration + "\x00" + observation.ClassifierVersion
		if _, ok := seen[identity]; !ok {
			seen[identity] = struct{}{}
			result.Identities = append(result.Identities, identity)
		}
	}
	addIdentity(result.Observation)
	for _, observation := range observations[1:] {
		if observation.SourceGeneration == result.Observation.SourceGeneration && observation.ClassifierVersion == result.Observation.ClassifierVersion {
			result.Observation = observation
			result.Coverage.ThroughRef = observation.ThroughRef
			continue
		}
		addIdentity(observation)
		if result.Reconciliations >= maxReconciliations {
			result.Coverage = CoveragePlane{Plane: observation.Plane, State: CoveragePartial, ThroughRef: observation.ThroughRef, Reason: "source changed again after the bounded reconciliation allowance"}
			result.Observation = observation
			return result
		}
		result.Reconciliations++
		result.Observation = observation
		result.Coverage.ThroughRef = observation.ThroughRef
	}
	return result
}

func RetentionGap(startSequence, earliestAvailableSequence int64) CoveragePlane {
	if startSequence < earliestAvailableSequence {
		return CoveragePlane{Plane: "events", State: CoverageRetentionGap, Reason: fmt.Sprintf("requested sequence begins at %d but retention begins at %d", startSequence, earliestAvailableSequence)}
	}
	return CoveragePlane{Plane: "events", State: CoverageComplete, Reason: "requested interval is retained"}
}
