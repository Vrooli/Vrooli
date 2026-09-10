package evidence

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	evidencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/evidence"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// PublicationSchemaVersion is the wire schema of Publication.
const PublicationSchemaVersion = "1"

// Publication states.
const (
	StateRequested      = "requested"
	StateReviewPrepared = "review_prepared"
	StateApproved       = "approved"
	StateActivating     = "activating"
	StatePublished      = "published"
	StateRefused        = "refused"
	StateFailed         = "failed"
	StateUnknown        = "unknown"
)

// Refusal codes recorded on a publication.
const (
	RefusalEvidenceIncomplete = "evidence_incomplete"
	RefusalReviewMismatch     = "review_identity_mismatch"
	RefusalReviewNotApproved  = "review_not_approved"
	RefusalReviewRevoked      = "review_revoked"
	RefusalReviewUnavailable  = "review_unavailable"
	RefusalPlanMismatch       = "plan_digest_mismatch"
	RefusalTargetMismatch     = "target_release_mismatch"
)

// Refusal names why a publication effect was not performed.
type Refusal struct {
	Code   string `json:"code"`
	Reason string `json:"reason"`
}

// TargetReceipt is the target-owned active-release pointer as observed after
// activation. ReceiptDigest is sha256 over its canonical JSON.
type TargetReceipt struct {
	ActiveRelease   string `json:"active_release"`
	PreviousRelease string `json:"previous_release,omitempty"`
	ActivatedAt     string `json:"activated_at,omitempty"`
	OperationID     string `json:"operation_id,omitempty"`
	Fence           uint64 `json:"fence,omitempty"`
	ReceiptDigest   string `json:"receipt_digest,omitempty"`
}

// Digest computes the canonical digest of the pointer content.
func (t TargetReceipt) Digest() string {
	t.ReceiptDigest = ""
	raw, _ := json.Marshal(t)
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:])
}

// Publication is the durable governed publication record for one request
// key on one deployment.
type Publication struct {
	SchemaVersion            string         `json:"schema_version"`
	ID                       string         `json:"id"`
	DeploymentID             string         `json:"deployment_id"`
	RequestKey               string         `json:"request_key"`
	Identity                 ReviewIdentity `json:"identity"`
	IdentityDigest           string         `json:"identity_digest"`
	ReviewRef                string         `json:"review_ref,omitempty"`
	ReleaseDigest            string         `json:"release_digest"`
	PlanDigest               string         `json:"plan_digest,omitempty"`
	State                    string         `json:"state"`
	OperationID              string         `json:"operation_id,omitempty"`
	ActivatedReleaseDigest   string         `json:"activated_release_digest,omitempty"`
	PredecessorReleaseDigest string         `json:"predecessor_release_digest,omitempty"`
	TargetKey                string         `json:"target_key"`
	TargetReceipt            *TargetReceipt `json:"target_receipt,omitempty"`
	Refusal                  *Refusal       `json:"refusal,omitempty"`
	RequestedAt              time.Time      `json:"requested_at"`
	UpdatedAt                time.Time      `json:"updated_at"`
	PublishedAt              *time.Time     `json:"published_at,omitempty"`
}

// Refuse moves the publication into refused with a typed reason.
func (p *Publication) Refuse(code, reason string, now time.Time) {
	p.State = StateRefused
	p.Refusal = &Refusal{Code: code, Reason: reason}
	p.UpdatedAt = now.UTC()
}

// Decide re-checks the owner's review immediately before an effect. It
// returns a refusal when the review is absent, not approved, superseded, or
// bound to a different identity than the one stored at request time.
func Decide(p *Publication, review *ReviewSnapshot, reviewErr error) *Refusal {
	if reviewErr != nil {
		return &Refusal{Code: RefusalReviewUnavailable, Reason: "governance owner did not return the review: " + reviewErr.Error()}
	}
	if review == nil || strings.TrimSpace(review.Key) == "" {
		return &Refusal{Code: RefusalReviewUnavailable, Reason: "governance owner returned no review"}
	}
	if review.Key != p.ReviewRef {
		return &Refusal{Code: RefusalReviewMismatch, Reason: fmt.Sprintf("review %s is not the requested review %s", review.Key, p.ReviewRef)}
	}
	if err := review.MatchesIdentity(p.Identity); err != nil {
		return &Refusal{Code: RefusalReviewMismatch, Reason: err.Error()}
	}
	digest, err := p.Identity.Digest()
	if err != nil || digest != p.IdentityDigest {
		return &Refusal{Code: RefusalReviewMismatch, Reason: "stored identity digest no longer matches the requested identity"}
	}
	switch review.Status {
	case ReviewApproved:
		return nil
	case ReviewSuperseded:
		return &Refusal{Code: RefusalReviewRevoked, Reason: "review was superseded before the publication effect"}
	case ReviewPromoted:
		return &Refusal{Code: RefusalReviewRevoked, Reason: "review was already promoted; a new publication needs a new review"}
	default:
		return &Refusal{Code: RefusalReviewNotApproved, Reason: fmt.Sprintf("review status is %q, not approved", review.Status)}
	}
}

// ResolvePredecessor picks the predecessor release: the target's own
// previous-release pointer when present, otherwise the most recent published
// release for the deployment before this publication. Both sources are
// target- or history-owned; the caller never supplies it.
func ResolvePredecessor(receipt *TargetReceipt, history []Publication, current *Publication) string {
	if receipt != nil && strings.TrimSpace(receipt.PreviousRelease) != "" {
		return receipt.PreviousRelease
	}
	var best *Publication
	for i := range history {
		h := &history[i]
		if h.State != StatePublished || h.PublishedAt == nil || (current != nil && h.ID == current.ID) {
			continue
		}
		if current != nil && current.PublishedAt != nil && h.PublishedAt.After(*current.PublishedAt) {
			continue
		}
		if best == nil || h.PublishedAt.After(*best.PublishedAt) {
			best = h
		}
	}
	if best == nil {
		return ""
	}
	return best.ActivatedReleaseDigest
}

// RecordActivation records the actual activated release from the target
// receipt. A pointer naming a different release than the approved one is a
// target_release_mismatch: the publication is failed, never reported as
// published for the approved digest.
func RecordActivation(p *Publication, receipt TargetReceipt, history []Publication, now time.Time) {
	receipt.ReceiptDigest = receipt.Digest()
	p.TargetReceipt = &receipt
	p.UpdatedAt = now.UTC()
	if !strings.EqualFold(strings.TrimPrefix(receipt.ActiveRelease, "sha256:"), strings.TrimPrefix(p.ReleaseDigest, "sha256:")) {
		p.State = StateFailed
		p.ActivatedReleaseDigest = receipt.ActiveRelease
		p.Refusal = &Refusal{Code: RefusalTargetMismatch, Reason: fmt.Sprintf("target reports active release %s, approved release is %s", short(receipt.ActiveRelease), short(p.ReleaseDigest))}
		return
	}
	p.State = StatePublished
	p.ActivatedReleaseDigest = receipt.ActiveRelease
	p.PredecessorReleaseDigest = ResolvePredecessor(&receipt, history, p)
	published := now.UTC()
	p.PublishedAt = &published
	p.Refusal = nil
}

// Proto projects the publication onto the wire.
func (p Publication) Proto(evidence *Summary) *evidencev1.Publication {
	out := &evidencev1.Publication{
		SchemaVersion: PublicationSchemaVersion, Id: p.ID, DeploymentId: p.DeploymentID, RequestKey: p.RequestKey,
		Identity: p.Identity.Proto(), ReviewRef: p.ReviewRef, ReleaseDigest: p.ReleaseDigest, PlanDigest: p.PlanDigest,
		State: p.State, OperationId: p.OperationID, ActivatedReleaseDigest: p.ActivatedReleaseDigest,
		PredecessorReleaseDigest: p.PredecessorReleaseDigest, TargetKey: p.TargetKey, RequestedAt: timestamppb.New(p.RequestedAt),
	}
	if p.TargetReceipt != nil {
		out.TargetReceipt = &evidencev1.TargetReceipt{ActiveRelease: p.TargetReceipt.ActiveRelease, PreviousRelease: p.TargetReceipt.PreviousRelease, ActivatedAt: p.TargetReceipt.ActivatedAt, OperationId: p.TargetReceipt.OperationID, Fence: p.TargetReceipt.Fence, ReceiptDigest: p.TargetReceipt.ReceiptDigest}
	}
	if p.Refusal != nil {
		out.Refusal = &evidencev1.Refusal{Code: p.Refusal.Code, Reason: p.Refusal.Reason}
	}
	if p.PublishedAt != nil {
		out.PublishedAt = timestamppb.New(*p.PublishedAt)
	}
	if evidence != nil {
		out.Evidence = evidence.Proto()
	}
	return out
}
