// Package source implements the provider-neutral source distribution core.
// It deliberately accepts a directory snapshot instead of a Git repository:
// source export is one-way, deterministic, and never owns repository history.
package source

import (
	closure "github.com/vrooli/vrooli/scenarios/scenario-dependency-analyzer/api/closure"
	"time"
)

// Closure inventory is owned by Scenario Dependency Analyzer. These aliases
// keep the source-ramp domain wire-neutral while preventing a second closure
// implementation from growing here.
type File = closure.File
type Node = closure.Node
type Rewrite = closure.Rewrite
type Closure = closure.SourceClosure

type Recipe struct {
	SchemaVersion     int    `json:"schemaVersion" yaml:"schema_version"`
	Scenario          string `json:"scenario" yaml:"scenario"`
	Mode              string `json:"mode" yaml:"mode"`
	ClosureRef        string `json:"closureRef" yaml:"closure_ref"`
	PublishPolicyRef  string `json:"publishPolicyRef" yaml:"publish_policy_ref"`
	RuntimeProfileRef string `json:"runtimeProfileRef" yaml:"runtime_profile_ref"`
	ArchiveFormat     string `json:"archiveFormat" yaml:"archive"`
	Deterministic     bool   `json:"deterministic" yaml:"deterministic"`
}

type ManifestEntry struct {
	Path       string `json:"path"`
	SHA256     string `json:"sha256"`
	Mode       uint32 `json:"mode"`
	SourcePath string `json:"sourcePath"`
	SizeBytes  int64  `json:"sizeBytes"`
}

type Artifact struct {
	ArtifactID     string          `json:"artifactId"`
	SourceDigest   string          `json:"sourceDigest"`
	RecipeDigest   string          `json:"recipeDigest"`
	ClosureDigest  string          `json:"closureDigest"`
	Files          []ManifestEntry `json:"files"`
	ManifestDigest string          `json:"manifestDigest"`
	ArchiveDigest  string          `json:"archiveDigest"`
	ArchivePath    string          `json:"archivePath"`
	Status         string          `json:"status"`
}

type Verification struct {
	VerificationID string    `json:"verificationId"`
	ArtifactID     string    `json:"artifactId"`
	Status         string    `json:"status"`
	GatesPassed    []string  `json:"gatesPassed"`
	Failures       []string  `json:"failures"`
	ReceiptDigest  string    `json:"receiptDigest"`
	VerifiedAt     time.Time `json:"verifiedAt"`
}

type Distribution struct {
	DistributionID     string    `json:"distributionId"`
	Scenario           string    `json:"scenario"`
	SourceDigest       string    `json:"sourceDigest"`
	ClosureDigest      string    `json:"closureDigest"`
	RecipeDigest       string    `json:"recipeDigest"`
	PolicyDigest       string    `json:"policyDigest"`
	ArtifactID         string    `json:"artifactId"`
	ArtifactDigest     string    `json:"artifactDigest"`
	VerificationStatus string    `json:"verificationStatus"`
	VerificationReceipt string   `json:"verificationReceipt"`
	DeploymentDecision string    `json:"deploymentManagerDecision"`
	PublicationStatus  string    `json:"publicationStatus"`
	DestinationKind    string    `json:"destinationKind"`
	DestinationRef     string    `json:"destinationReference"`
	DestinationRevision string   `json:"destinationRevision"`
	ReadbackReceipt    string    `json:"readbackReceipt"`
	DriftState         string    `json:"driftState"`
	SourceOfTruth      string    `json:"sourceOfTruth"`
	SourceTimestamp    time.Time `json:"sourceTimestamp"`
	WorkflowURL        string    `json:"workflowUrl"`
	Contents           []DistributionContent `json:"contents,omitempty"`
	Exclusions         []DistributionExclusion `json:"exclusions,omitempty"`
	UnresolvedObligations []string `json:"unresolvedObligations,omitempty"`
	RuntimeRequirements []string `json:"runtimeRequirements,omitempty"`
	UpdatedAt          time.Time `json:"updatedAt"`
}

type DistributionContent struct {
	Path       string `json:"path"`
	SourcePath string `json:"sourcePath"`
	Category   string `json:"category"`
	Digest     string `json:"digest"`
	SizeBytes  int64  `json:"sizeBytes"`
}

type DistributionExclusion struct {
	Path       string `json:"path"`
	Category   string `json:"category"`
	SafeReason string `json:"safeReason"`
}
