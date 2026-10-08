package evidence

import (
	"context"
	"time"
)

type Repository interface {
	CreateReceipt(context.Context, NewObservation) (Receipt, error)
	GetReceipt(context.Context, string) (Receipt, error)
	CreatePassage(context.Context, string, int, int) (Passage, error)
	GetPassage(context.Context, string) (Passage, error)
	ExpireContent(context.Context, time.Time) (int, error)
}

// AssessmentWriter is an optional extension implemented by durable evidence
// stores. Keeping it separate preserves the transport-neutral receipt seam.
type AssessmentWriter interface {
	CreateAssessment(context.Context, Assessment) (Assessment, error)
}

type AssessmentReader interface {
	GetAssessment(context.Context, string) (Assessment, error)
}

// RetentionOperator is implemented by durable stores that can preview and
// execute bounded content expiry while protecting artifacts referenced by an
// active evaluation or investigation.
type RetentionOperator interface {
	PreviewContentExpiry(context.Context, time.Time, []string) (RetentionPreview, error)
	ExpireContentExcept(context.Context, time.Time, []string) (int, error)
}

type RetentionPreview struct {
	Before        time.Time
	ArtifactCount int
	ContentBytes  int64
}
