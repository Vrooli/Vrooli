package adapters

import (
	"context"
	"unit-health/internal/testquality"
)

type SyntaxCollector interface {
	CollectSyntax(context.Context, QualityInput) ([]testquality.Result, testquality.Reason)
}

// SourceAnalyzer is separate from runtime collection and configuration parsing.
// A language adapter can provide it without implementing unrelated UI checks.
type SourceAnalyzer interface {
	Adapter
	AnalyzeSource(context.Context, QualityInput) ([]testquality.Result, testquality.Reason)
}

type SourceEvidenceAnalyzer interface {
	AnalyzeSourceEvidence(context.Context, QualityInput) ([]testquality.Result, []testquality.TestLinks, testquality.Reason)
}

type ExecutionEvidenceInput struct {
	Root, RunID  string
	Stdout       []byte
	Complete     bool
	Declarations []testquality.TestLinks
}
type ExecutionEvidenceAdapter interface {
	PrepareExecutionEvidence(string, []string) ([]string, bool)
	ObserveExecutionEvidence(ExecutionEvidenceInput) ([]testquality.TestLinks, testquality.Reason)
}

// QualityCollector is an optional adapter capability. The kernel supplies a
// command-unique run identity and never interprets framework evidence itself.
type QualityCollector interface {
	QualityArtifact(root, runID string) Artifact
	CollectQuality(input QualityInput) ([]testquality.Result, testquality.Reason)
}

// QualityEvidenceCollector returns assertions and declared links from one
// validated native artifact read, avoiding a competing execution parser.
type QualityEvidenceCollector interface {
	CollectQualityEvidence(QualityInput) ([]testquality.Result, []testquality.TestLinks, testquality.Reason)
}

type QualityInput struct {
	Workspace string
	Root      string
	RunID     string
	Artifact  Artifact
	TestKind  string
	Executed  bool
}
