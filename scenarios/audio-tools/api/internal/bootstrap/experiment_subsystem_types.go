package bootstrap

import (
	"context"
	"time"

	measuretypes "audio-tools/internal/measures"
	"audio-tools/internal/modelpromotion"
	trustfloor "audio-tools/internal/qualification"
)

type experimentSubsystem struct {
	labRuntime            *labRuntimeDeps
	corpusCounter         corpusCounterProjection
	experimentCounter     measuretypes.ExperimentCounter
	qualificationEvidence modelpromotion.QualificationEvidenceStore
}

type corpusCountRepository interface {
	CountCreatedInWindow(context.Context, time.Time, time.Time) (int64, error)
}

type corpusCounterProjection struct{ repo corpusCountRepository }

func (p corpusCounterProjection) CountClipsCreatedInWindow(ctx context.Context, from, to time.Time) (int64, error) {
	if p.repo == nil {
		return 0, nil
	}
	return p.repo.CountCreatedInWindow(ctx, from, to)
}

type emptyExperimentCounter struct{}

func (emptyExperimentCounter) CountCreatedInRange(context.Context, time.Time, time.Time) (measuretypes.ExperimentCreatedCounts, error) {
	return measuretypes.ExperimentCreatedCounts{}, nil
}

type emptyQualificationEvidenceStore struct{}

func (emptyQualificationEvidenceStore) ListQualificationEvidence(context.Context, trustfloor.QualificationEvidenceFilter) ([]trustfloor.QualificationEvidence, error) {
	return nil, nil
}

func shouldStartLabWorker(labAPIEnabled, experimentServiceAvailable bool) bool {
	return labAPIEnabled && experimentServiceAvailable
}
