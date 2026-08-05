package journal

import "time"

const UnclassifiedFacet = "unclassified"

type (
	Entry struct {
		ID, Scope, Body, FacetID, Kind, ImportKey string
		Existing                                  bool
		Attribution                               Attribution
		Import                                    ImportProvenance
		Correlation                               Correlation
		FacetTexts                                []FacetText
		CreatedAt                                 time.Time
	}
	RetryItem struct {
		ID, Reason string
		Entry      Entry
	}
	RetryResult struct {
		Processed, Deferred, AlreadyResolved int
	}
	Attribution      struct{ ActorID, ActorKind, SourceRuntime string }
	ImportProvenance struct {
		Harness, Path string
		ImportedAt    time.Time
	}
	Correlation struct{ RunID, WorkflowExecutionID, ActorKind string }
	FacetText   struct {
		ID, Kind, Text, EmbeddingRef string
		Vector                       []float64
	}
)
