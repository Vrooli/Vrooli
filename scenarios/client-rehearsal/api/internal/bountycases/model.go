package bountycases

import "context"

// CaseBundle is the immutable, trusted registration consumed by the private
// Program Runtime binding. It deliberately contains no evaluator bytes and
// no caller-provided validation intents. Its JSON field names are the
// protojson lowerCamelCase contract used by program-runtime.bounty-replay.
type CaseBundle struct {
	SchemaVersion               uint32        `json:"schemaVersion"`
	BundleVersion               uint32        `json:"bundleVersion"`
	CaseID                      string        `json:"caseId"`
	Project                     string        `json:"project"`
	Repository                  string        `json:"repository"`
	Title                       string        `json:"title"`
	Task                        string        `json:"task"`
	PublicURL                   string        `json:"publicUrl"`
	IssueNumber                 uint64        `json:"issueNumber"`
	PullRequestURL              string        `json:"pullRequestUrl"`
	PullRequestNumber           uint64        `json:"pullRequestNumber"`
	SourceCommit                string        `json:"sourceCommit"`
	ReferenceCommit             string        `json:"referenceCommit"`
	SourceProjectRoot           string        `json:"sourceProjectRoot"`
	ReferenceProjectRoot        string        `json:"referenceProjectRoot"`
	FrozenEvaluatorDigest       string        `json:"frozenEvaluatorDigest"`
	FrozenEvaluatorBundleDigest string        `json:"frozenEvaluatorBundleDigest"`
	FrozenEvaluatorRoot         string        `json:"frozenEvaluatorRoot"`
	FrozenEvaluatorEntrypoint   string        `json:"frozenEvaluatorEntrypoint"`
	TestGenieProjectID          string        `json:"testGenieProjectId"`
	TestGeniePlan               TestGeniePlan `json:"testGeniePlan"`
	BundleDigest                string        `json:"bundleDigest"`
}

type TestGeniePlan struct {
	ValidationPurpose string               `json:"validationPurpose"`
	RequiredStrength  string               `json:"requiredStrength"`
	Phases            []string             `json:"phases"`
	EvidencePolicy    ReplayEvidencePolicy `json:"evidencePolicy"`
	DeadlinePolicy    ReplayDeadlinePolicy `json:"deadlinePolicy"`
}

type ReplayEvidencePolicy struct {
	RequireBehavioralBefore    bool     `json:"requireBehavioralBefore"`
	RequireSourceSnapshot      bool     `json:"requireSourceSnapshot"`
	AllowAuthorizedDegradation bool     `json:"allowAuthorizedDegradation"`
	RequiredEvidenceKinds      []string `json:"requiredEvidenceKinds"`
}

type ReplayDeadlinePolicy struct {
	QueueBudgetMilliseconds     uint64 `json:"queueBudgetMilliseconds"`
	ExecutionBudgetMilliseconds uint64 `json:"executionBudgetMilliseconds"`
	MaximumAttempts             uint32 `json:"maximumAttempts"`
}

type EvaluatorFile struct {
	RelativePath string `json:"relativePath"`
	Content      []byte `json:"content"`
	SHA256       string `json:"sha256"`
}

type FrozenEvaluatorBundle struct {
	EvaluatorBundleID string          `json:"evaluatorBundleId"`
	CaseID            string          `json:"caseId"`
	BundleVersion     uint32          `json:"bundleVersion"`
	Entrypoint        string          `json:"entrypoint"`
	EvaluatorDigest   string          `json:"evaluatorDigest"`
	ContentDigest     string          `json:"contentDigest"`
	Files             []EvaluatorFile `json:"files"`
}

// SolverProjection is the only case representation suitable for a solver.
// It carries the task and preimage checkout identity but has no fields for
// reference roots, evaluator identity/content or hidden metadata.
type SolverProjection struct {
	CaseID            string `json:"caseId"`
	Project           string `json:"project"`
	Repository        string `json:"repository"`
	Title             string `json:"title"`
	Task              string `json:"task"`
	PublicURL         string `json:"publicUrl"`
	IssueNumber       uint64 `json:"issueNumber"`
	SourceCommit      string `json:"sourceCommit"`
	SourceProjectRoot string `json:"sourceProjectRoot"`
}

type Lookup struct {
	CaseID    string
	PublicURL string
}

type Repository interface {
	Register(context.Context, CaseBundle) (CaseBundle, error)
	Resolve(context.Context, Lookup) (CaseBundle, error)
	GetVersion(context.Context, string, uint32) (CaseBundle, error)
}

type Service interface {
	Register(context.Context, CaseBundle) (CaseBundle, error)
	Resolve(context.Context, Lookup) (CaseBundle, error)
	ResolveEvaluatorBundle(context.Context, string) (FrozenEvaluatorBundle, error)
	ProjectForSolver(CaseBundle) SolverProjection
}
