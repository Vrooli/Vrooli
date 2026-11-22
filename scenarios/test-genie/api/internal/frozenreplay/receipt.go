// Package frozenreplay defines the owner-side receipt contract for blind
// bounty replay. It does not execute tests or access workspaces; it adjudicates
// three independently produced, source-bound outcomes after Sandbox has sealed
// the candidate and revealed the reference.
package frozenreplay

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

type Variant string

const (
	Preimage  Variant = "preimage"
	Candidate Variant = "candidate"
	Reference Variant = "reference"
)

// RunEvidence is a locator and outcome emitted by the trusted Test Genie run
// owner. ArtifactDigest covers the immutable test output, not a caller-supplied
// summary. SourceIdentity is the exact workspace snapshot tested.
type RunEvidence struct {
	Variant         Variant `json:"variant"`
	RunID           string  `json:"run_id"`
	ProducerReceipt string  `json:"producer_receipt"`
	SourceIdentity  string  `json:"source_identity"`
	EvaluatorDigest string  `json:"evaluator_digest"`
	Status          string  `json:"status"` // passed or failed
	ArtifactDigest  string  `json:"artifact_digest"`
}

// Receipt binds all three outcomes to the same sealed bounty case and frozen
// evaluator. ReceiptDigest is SHA-256 over the other fields in this struct;
// callers must persist the returned value append-only and never edit it.
type Receipt struct {
	Version           uint32        `json:"version"`
	CaseID            string        `json:"case_id"`
	ProjectID         string        `json:"project_id"`
	FrozenEvaluator   string        `json:"frozen_evaluator_digest"`
	PreimageIdentity  string        `json:"preimage_identity"`
	CandidateSeal     string        `json:"candidate_seal_digest"`
	ReferenceIdentity string        `json:"reference_identity"`
	Evidence          []RunEvidence `json:"evidence"`
	Disposition       string        `json:"disposition"`
	ReceiptDigest     string        `json:"receipt_digest"`
}

var ErrInvalidReceipt = errors.New("invalid frozen replay receipt")

// EvidenceVerifier must resolve each run and producer receipt from Test
// Genie's immutable owner record and compare the artifact digest, test result,
// evaluator, and source identity. Adjudicate never trusts caller assertions
// about run provenance.
type EvidenceVerifier interface {
	VerifyEvidence(context.Context, RunEvidence) error
}

// EvidenceArtifactDigester lets the owner derive the artifact digest from its
// immutable producer record so clients do not have to reserialize protobufs.
type EvidenceArtifactDigester interface {
	ArtifactDigest(context.Context, RunEvidence) (string, error)
}

// Adjudicate accepts only an explicitly complete three-way run set. The
// evaluator must have failed on the true preimage and passed on both the
// candidate and merged reference. Any other outcome is retained as a failed
// qualification receipt rather than silently omitted.
func Adjudicate(ctx context.Context, input Receipt, verifier EvidenceVerifier) (Receipt, error) {
	if ctx == nil {
		return Receipt{}, invalid("context is required")
	}
	if verifier == nil {
		return Receipt{}, invalid("owner evidence verifier is required")
	}
	input.Version = 1
	input.ReceiptDigest = ""
	if strings.TrimSpace(input.CaseID) == "" || strings.TrimSpace(input.ProjectID) == "" {
		return Receipt{}, invalid("case and project identity are required")
	}
	if !validDigest(input.FrozenEvaluator) || !validDigest(input.CandidateSeal) {
		return Receipt{}, invalid("frozen evaluator and candidate seal must be sha256 digests")
	}
	if !validDigest(input.PreimageIdentity) || !validDigest(input.ReferenceIdentity) {
		return Receipt{}, invalid("preimage and reference source identities must be sha256 digests")
	}
	if len(input.Evidence) != 3 {
		return Receipt{}, invalid("exactly three variant outcomes are required")
	}
	byVariant := make(map[Variant]RunEvidence, 3)
	seenRuns := map[string]struct{}{}
	input.Evidence = append([]RunEvidence(nil), input.Evidence...)
	for index, evidence := range input.Evidence {
		if evidence.Variant != Preimage && evidence.Variant != Candidate && evidence.Variant != Reference {
			return Receipt{}, invalid("unknown variant %q", evidence.Variant)
		}
		if _, exists := byVariant[evidence.Variant]; exists {
			return Receipt{}, invalid("duplicate %s outcome", evidence.Variant)
		}
		if evidence.ArtifactDigest == "" {
			digester, ok := verifier.(EvidenceArtifactDigester)
			if !ok {
				return Receipt{}, invalid("%s outcome lacks an owner-derived artifact digest", evidence.Variant)
			}
			artifactDigest, err := digester.ArtifactDigest(ctx, evidence)
			if err != nil {
				return Receipt{}, invalid("%s outcome artifact is unavailable from its owner record: %v", evidence.Variant, err)
			}
			evidence.ArtifactDigest = artifactDigest
			input.Evidence[index].ArtifactDigest = artifactDigest
		}
		if evidence.RunID == "" || evidence.ProducerReceipt == "" || !validIdentity(evidence.SourceIdentity) || !validDigest(evidence.EvaluatorDigest) || !validDigest(evidence.ArtifactDigest) {
			return Receipt{}, invalid("%s outcome lacks trusted run provenance", evidence.Variant)
		}
		if evidence.EvaluatorDigest != input.FrozenEvaluator {
			return Receipt{}, invalid("%s outcome used a different evaluator than the frozen version", evidence.Variant)
		}
		if err := verifier.VerifyEvidence(ctx, evidence); err != nil {
			return Receipt{}, invalid("%s outcome is not present in the owner record: %v", evidence.Variant, err)
		}
		if _, exists := seenRuns[evidence.RunID]; exists {
			return Receipt{}, invalid("variant outcomes must come from distinct runs")
		}
		seenRuns[evidence.RunID] = struct{}{}
		if evidence.Status != "passed" && evidence.Status != "failed" {
			return Receipt{}, invalid("%s outcome has unsupported status %q", evidence.Variant, evidence.Status)
		}
		byVariant[evidence.Variant] = evidence
	}
	pre, hasPre := byVariant[Preimage]
	candidate, hasCandidate := byVariant[Candidate]
	reference, hasReference := byVariant[Reference]
	if !hasPre || !hasCandidate || !hasReference {
		return Receipt{}, invalid("preimage, candidate, and reference outcomes are all required")
	}
	if !validDigest(candidate.SourceIdentity) || pre.SourceIdentity != input.PreimageIdentity || candidate.SourceIdentity != input.CandidateSeal || reference.SourceIdentity != input.ReferenceIdentity {
		return Receipt{}, invalid("run source identity does not match its sealed variant identity")
	}
	input.Evidence = []RunEvidence{pre, candidate, reference}
	if pre.Status == "failed" && candidate.Status == "passed" && reference.Status == "passed" {
		input.Disposition = "qualified"
	} else {
		input.Disposition = "not_qualified"
	}
	payload, err := json.Marshal(input)
	if err != nil {
		return Receipt{}, fmt.Errorf("marshal replay receipt: %w", err)
	}
	digest := sha256.Sum256(payload)
	input.ReceiptDigest = "sha256:" + hex.EncodeToString(digest[:])
	return input, nil
}

// Verify detects edits to a receipt after its producer has sealed it.
func Verify(receipt Receipt) error {
	claimed := receipt.ReceiptDigest
	if !validDigest(claimed) {
		return invalid("receipt digest is missing or malformed")
	}
	receipt.ReceiptDigest = ""
	encoded, err := json.Marshal(receipt)
	if err != nil {
		return fmt.Errorf("marshal replay receipt: %w", err)
	}
	digest := sha256.Sum256(encoded)
	if claimed != "sha256:"+hex.EncodeToString(digest[:]) {
		return invalid("receipt digest does not match its contents")
	}
	return nil
}

func validIdentity(identity string) bool {
	return strings.TrimSpace(identity) != "" && !strings.ContainsAny(identity, "\r\n\x00")
}

func validDigest(value string) bool {
	if !strings.HasPrefix(value, "sha256:") || len(value) != len("sha256:")+64 {
		return false
	}
	_, err := hex.DecodeString(strings.TrimPrefix(value, "sha256:"))
	return err == nil
}

func invalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrInvalidReceipt, fmt.Sprintf(format, args...))
}
