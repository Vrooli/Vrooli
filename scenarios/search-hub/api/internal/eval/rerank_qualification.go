package eval

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	evalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/eval"
	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/routing"
	"google.golang.org/protobuf/proto"
)

const (
	minimumRerankReviewedPositives = 12
	minimumRerankHeldoutPositives  = 3
	maxRerankSnapshotCandidates    = 64
	maxRerankCandidateBytes        = 4096
)

// QualifyRerankCorpus reports whether an owner-authored suite contains a fixed,
// sufficiently broad reranker denominator and immutable pre-rerank candidate
// bytes. It never adds cases, tags, snapshots, or receipts. A false result is
// deliberately explicit and suitable for GetSuite's qualification readback.
func QualifyRerankCorpus(ctx context.Context, suite *evalv1.EvalSuite, verifier RerankCorpusOwnerReceiptVerifier) *evalv1.RerankCorpusQualification {
	result := &evalv1.RerankCorpusQualification{}
	if suite == nil {
		result.RefusalReason = "suite_missing"
		return result
	}
	result.CorpusSha256 = corpusDigest(suite)
	denominator := reviewedPositiveCaseIDs(suite)
	result.ReviewedPositive = int32(len(denominator))
	result.DenominatorSha256 = denominatorDigest(denominator)
	for _, c := range suite.GetCases() {
		if c != nil && positiveProtoCase(c) && !isExcludedEvaluationCase(c) && hasEvaluationTag(c.GetTags(), "heldout") {
			result.HeldoutPositive++
		}
	}
	result.CandidateSnapshotsSha256 = RerankCandidateSnapshotsDigest(suite)

	receipt := suite.GetRerankCorpusOwnerReceipt()
	if receipt == nil {
		result.RefusalReason = "owner_receipt_missing"
		return result
	}
	if strings.TrimSpace(receipt.GetOwnerProviderId()) == "" || receipt.GetOwnerProviderId() != suite.GetProviderId() {
		result.RefusalReason = "owner_provider_mismatch"
		return result
	}
	if strings.TrimSpace(receipt.GetApprovalRef()) == "" || strings.TrimSpace(receipt.GetSourceRevision()) == "" || strings.TrimSpace(receipt.GetApprovedBy()) == "" {
		result.RefusalReason = "owner_approval_provenance_incomplete"
		return result
	}
	approvedAt, err := time.Parse(time.RFC3339Nano, receipt.GetApprovedAt())
	if err != nil || approvedAt.After(time.Now().Add(5*time.Minute)) {
		result.RefusalReason = "owner_approval_time_invalid"
		return result
	}
	if len(denominator) < minimumRerankReviewedPositives {
		result.RefusalReason = "too_few_reviewed_positives"
		return result
	}
	if result.GetHeldoutPositive() < minimumRerankHeldoutPositives {
		result.RefusalReason = "too_few_owner_heldout_positives"
		return result
	}
	for _, id := range denominator {
		c := evaluationCaseByID(suite, id)
		if c == nil || len(c.GetPreRerankCandidates()) == 0 {
			result.RefusalReason = "candidate_snapshot_missing:" + id
			return result
		}
		if err := validatePreRerankCandidates(c.GetPreRerankCandidates()); err != nil {
			result.RefusalReason = "candidate_snapshot_invalid:" + id + ":" + err.Error()
			return result
		}
	}
	if !isSHA256(receipt.GetCorpusSha256()) || receipt.GetCorpusSha256() != ownerCorpusDigest(suite) {
		result.RefusalReason = "owner_corpus_digest_mismatch"
		return result
	}
	if !isSHA256(receipt.GetDenominatorSha256()) || receipt.GetDenominatorSha256() != result.GetDenominatorSha256() {
		result.RefusalReason = "owner_denominator_digest_mismatch"
		return result
	}
	if !isSHA256(receipt.GetCandidateSnapshotsSha256()) || receipt.GetCandidateSnapshotsSha256() != result.GetCandidateSnapshotsSha256() {
		result.RefusalReason = "owner_candidate_snapshot_digest_mismatch"
		return result
	}
	receiptDigest := RerankCorpusOwnerReceiptDigest(receipt)
	if !isSHA256(receipt.GetReceiptSha256()) || receipt.GetReceiptSha256() != receiptDigest {
		result.RefusalReason = "owner_receipt_digest_mismatch"
		return result
	}
	if verifier == nil {
		result.RefusalReason = "owner_receipt_verifier_unavailable"
		return result
	}
	if err := verifier.VerifyRerankCorpusOwnerReceipt(ctx, suite, receiptDigest); err != nil {
		result.RefusalReason = "owner_receipt_unverified"
		return result
	}
	result.OwnerReceiptSha256 = receiptDigest
	result.Qualified = true
	return result
}

// VerifyRerankCorpusOwnerReceipt returns an opaque evidence value only after
// structural checks and the provider-owned source verifier both pass.
func VerifyRerankCorpusOwnerReceipt(ctx context.Context, suite *evalv1.EvalSuite, verifier RerankCorpusOwnerReceiptVerifier) (VerifiedRerankCorpusOwnerReceipt, error) {
	qualification := QualifyRerankCorpus(ctx, suite, verifier)
	if !qualification.GetQualified() {
		return VerifiedRerankCorpusOwnerReceipt{}, fmt.Errorf("reranker corpus refused: %s", qualification.GetRefusalReason())
	}
	return VerifiedRerankCorpusOwnerReceipt{receiptSHA256: qualification.GetOwnerReceiptSha256(), verified: true}, nil
}

// RerankCandidateSnapshotsDigest hashes the exact ordered SearchHit bytes for
// every fixed positive denominator case. Case ordering is canonical; candidate
// order is preserved because it is part of the pre-rerank input.
func RerankCandidateSnapshotsDigest(suite *evalv1.EvalSuite) string {
	if suite == nil {
		return ""
	}
	ids := reviewedPositiveCaseIDs(suite)
	sets := make([]*evalv1.RerankCandidateSet, 0, len(ids))
	for _, id := range ids {
		c := evaluationCaseByID(suite, id)
		if c == nil {
			return ""
		}
		sets = append(sets, &evalv1.RerankCandidateSet{CaseId: id, Candidates: c.GetPreRerankCandidates()})
	}
	canonical := &evalv1.RerankCandidateSetList{Sets: sets}
	encoded, err := (proto.MarshalOptions{Deterministic: true}).Marshal(canonical)
	if err != nil {
		return ""
	}
	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:])
}

// RerankCorpusOwnerReceiptDigest returns a stable content digest over every
// owner receipt claim except receipt_sha256 itself. This detects accidental or
// stale content changes; the caller still needs the provider-owned source path
// to establish that the receipt was actually issued by its owner.
func RerankCorpusOwnerReceiptDigest(receipt *evalv1.RerankCorpusOwnerReceipt) string {
	if receipt == nil {
		return ""
	}
	canonical := proto.Clone(receipt).(*evalv1.RerankCorpusOwnerReceipt)
	canonical.ReceiptSha256 = ""
	encoded, err := (proto.MarshalOptions{Deterministic: true}).Marshal(canonical)
	if err != nil {
		return ""
	}
	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:])
}

func ownerCorpusDigest(suite *evalv1.EvalSuite) string {
	if suite == nil {
		return ""
	}
	canonical := proto.Clone(suite).(*evalv1.EvalSuite)
	canonical.Name = ""
	canonical.Description = ""
	canonical.CreatedAt = ""
	canonical.UpdatedAt = ""
	canonical.RerankCorpusOwnerReceipt = nil
	encoded, err := (proto.MarshalOptions{Deterministic: true}).Marshal(canonical)
	if err != nil {
		return ""
	}
	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:])
}

func validatePreRerankCandidates(candidates []*routingv1.SearchHit) error {
	if len(candidates) == 0 || len(candidates) > maxRerankSnapshotCandidates {
		return fmt.Errorf("candidate count must be in [1,%d]", maxRerankSnapshotCandidates)
	}
	seen := make(map[string]struct{}, len(candidates))
	for i, candidate := range candidates {
		if candidate == nil || strings.TrimSpace(candidate.GetId()) == "" {
			return fmt.Errorf("candidate %d has no identity", i)
		}
		if _, exists := seen[candidate.GetId()]; exists {
			return fmt.Errorf("candidate %d duplicates id %q", i, candidate.GetId())
		}
		seen[candidate.GetId()] = struct{}{}
		if candidate.GetRerankScore() != 0 {
			return fmt.Errorf("candidate %d is not pre-rerank (rerank score is set)", i)
		}
		if math.IsNaN(candidate.GetScore()) || math.IsInf(candidate.GetScore(), 0) || candidate.GetScore() < 0 || candidate.GetScore() > 1 {
			return fmt.Errorf("candidate %d has an invalid normalized retrieval score", i)
		}
		if len(rerankInputText(candidate)) > maxRerankCandidateBytes {
			return fmt.Errorf("candidate %d input exceeds %d bytes", i, maxRerankCandidateBytes)
		}
	}
	return nil
}

func rerankInputText(hit *routingv1.SearchHit) string {
	parts := make([]string, 0, 4)
	if typ := strings.TrimSpace(hit.GetType()); typ != "" {
		parts = append(parts, "type: "+typ)
	}
	title := strings.TrimSpace(hit.GetTitle())
	if title == "" {
		title = strings.TrimSpace(hit.GetId())
	}
	if title != "" {
		parts = append(parts, "title: "+title)
	}
	if path := strings.TrimSpace(hit.GetPath()); path != "" && path != title {
		parts = append(parts, "path: "+path)
	}
	if snippet := strings.Join(strings.Fields(hit.GetSnippet()), " "); snippet != "" {
		if len(snippet) > 120 {
			snippet = snippet[:120] + "…"
		}
		parts = append(parts, "snippet: "+snippet)
	}
	return strings.Join(parts, "\n")
}

func evaluationCaseByID(suite *evalv1.EvalSuite, id string) *evalv1.EvalCase {
	for _, c := range suite.GetCases() {
		if c != nil && c.GetCaseId() == id {
			return c
		}
	}
	return nil
}

func RerankHeldoutIDs(suite *evalv1.EvalSuite) []string {
	if suite == nil {
		return nil
	}
	var ids []string
	for _, c := range suite.GetCases() {
		if c != nil && positiveProtoCase(c) && !isExcludedEvaluationCase(c) && hasEvaluationTag(c.GetTags(), "heldout") {
			ids = append(ids, c.GetCaseId())
		}
	}
	sort.Strings(ids)
	return ids
}
