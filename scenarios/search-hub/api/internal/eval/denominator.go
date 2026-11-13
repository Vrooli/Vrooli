package eval

import (
	"context"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	aisearch "github.com/vrooli/ai-go/search"
	evalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/eval"
	"google.golang.org/protobuf/proto"
)

// NewFixedDenominatorSnapshot creates a typed, owner-persistable snapshot from
// exactly the currently reviewed positive cases. The reviewer and evidence URI
// are explicit; provider validation is separate evidence and is not accepted as
// a substitute for this independent review.
func NewFixedDenominatorSnapshot(suite *evalv1.EvalSuite, reviewerID, evidenceURI string, reviewedAt time.Time) (*evalv1.FixedDenominatorSnapshot, error) {
	if suite == nil {
		return nil, fmt.Errorf("suite is required")
	}
	if strings.TrimSpace(reviewerID) == "" || strings.EqualFold(strings.TrimSpace(reviewerID), strings.TrimSpace(suite.GetProviderId())) {
		return nil, fmt.Errorf("reviewer_id must identify a reviewer independent of provider %q", suite.GetProviderId())
	}
	if err := validateReviewEvidenceURI(evidenceURI); err != nil {
		return nil, err
	}
	if reviewedAt.IsZero() {
		return nil, fmt.Errorf("review timestamp is required")
	}
	ids := reviewedPositiveCaseIDs(suite)
	if len(ids) == 0 {
		return nil, fmt.Errorf("suite has no reviewed positive cases to freeze")
	}
	byID := make(map[string]*evalv1.EvalCase, len(suite.GetCases()))
	for _, c := range suite.GetCases() {
		if c != nil {
			byID[c.GetCaseId()] = c
		}
	}
	caseSnapshots := make([]*evalv1.EvalCase, 0, len(ids))
	searchCases := make([]aisearch.TestCase, 0, len(ids))
	for _, id := range ids {
		c := byID[id]
		if c == nil {
			return nil, fmt.Errorf("reviewed positive case %q is missing from suite", id)
		}
		copy := proto.Clone(c).(*evalv1.EvalCase)
		caseSnapshots = append(caseSnapshots, copy)
		searchCases = append(searchCases, protoCaseToSearchCase(copy))
	}
	sealed, err := aisearch.NewFixedDenominator(suite.GetSuiteId(), searchCases, strings.TrimSpace(reviewerID), strings.TrimSpace(evidenceURI))
	if err != nil {
		return nil, fmt.Errorf("seal fixed denominator: %w", err)
	}
	return &evalv1.FixedDenominatorSnapshot{
		SuiteId: suite.GetSuiteId(), ProviderId: suite.GetProviderId(), CorpusSha256: corpusDigest(suite),
		ReviewerId: strings.TrimSpace(reviewerID), ReviewEvidenceUri: strings.TrimSpace(evidenceURI),
		ReviewedAt: reviewedAt.UTC().Format(timeFormat), CaseIds: ids, Cases: caseSnapshots,
		DenominatorSha256: sealed.SHA256,
	}, nil
}

// ValidateFixedDenominatorSnapshot re-creates the ai-go seal from the full
// persisted cases and rejects changed labels, omitted IDs, duplicate IDs, or
// reviewer/evidence drift.
func ValidateFixedDenominatorSnapshot(snapshot *evalv1.FixedDenominatorSnapshot) (aisearch.FixedDenominator, error) {
	if snapshot == nil || strings.TrimSpace(snapshot.GetSuiteId()) == "" || strings.TrimSpace(snapshot.GetProviderId()) == "" || strings.TrimSpace(snapshot.GetCorpusSha256()) == "" {
		return aisearch.FixedDenominator{}, fmt.Errorf("fixed denominator snapshot identity is incomplete")
	}
	if strings.TrimSpace(snapshot.GetReviewerId()) == "" || strings.EqualFold(strings.TrimSpace(snapshot.GetReviewerId()), strings.TrimSpace(snapshot.GetProviderId())) {
		return aisearch.FixedDenominator{}, fmt.Errorf("fixed denominator reviewer is missing or is the provider itself")
	}
	if err := validateReviewEvidenceURI(snapshot.GetReviewEvidenceUri()); err != nil {
		return aisearch.FixedDenominator{}, err
	}
	if _, err := time.Parse(timeFormat, snapshot.GetReviewedAt()); err != nil {
		return aisearch.FixedDenominator{}, fmt.Errorf("fixed denominator review timestamp is invalid: %w", err)
	}
	if len(snapshot.GetCases()) == 0 {
		return aisearch.FixedDenominator{}, fmt.Errorf("fixed denominator snapshot has no cases")
	}
	ids := make([]string, 0, len(snapshot.GetCases()))
	cases := make([]aisearch.TestCase, 0, len(snapshot.GetCases()))
	for _, c := range snapshot.GetCases() {
		if c == nil {
			return aisearch.FixedDenominator{}, fmt.Errorf("fixed denominator snapshot contains a nil case")
		}
		ids = append(ids, c.GetCaseId())
		cases = append(cases, protoCaseToSearchCase(c))
	}
	sort.Strings(ids)
	wantIDs := append([]string(nil), snapshot.GetCaseIds()...)
	sort.Strings(wantIDs)
	if !equalStrings(ids, wantIDs) {
		return aisearch.FixedDenominator{}, fmt.Errorf("fixed denominator case IDs do not match its case snapshots")
	}
	sealed, err := aisearch.NewFixedDenominator(snapshot.GetSuiteId(), cases, snapshot.GetReviewerId(), snapshot.GetReviewEvidenceUri())
	if err != nil {
		return aisearch.FixedDenominator{}, err
	}
	if sealed.SHA256 != snapshot.GetDenominatorSha256() {
		return aisearch.FixedDenominator{}, fmt.Errorf("fixed denominator digest does not match its reviewed cases")
	}
	if err := sealed.Validate(); err != nil {
		return aisearch.FixedDenominator{}, err
	}
	return sealed, nil
}

func resolveFixedDenominator(ctx context.Context, suite *evalv1.EvalSuite, reader FixedDenominatorReader, now time.Time) (*evalv1.FixedDenominatorSnapshot, aisearch.FixedDenominator, bool, error) {
	if reader != nil {
		snapshot, err := reader.GetFixedDenominator(ctx, suite.GetSuiteId())
		if err != nil {
			return nil, aisearch.FixedDenominator{}, false, fmt.Errorf("load fixed denominator: %w", err)
		}
		if snapshot == nil {
			return nil, aisearch.FixedDenominator{}, false, fmt.Errorf("fixed denominator independent review is required for suite %q", suite.GetSuiteId())
		}
		if snapshot.GetProviderId() != suite.GetProviderId() {
			return nil, aisearch.FixedDenominator{}, false, fmt.Errorf("fixed denominator provider %q does not match suite provider %q", snapshot.GetProviderId(), suite.GetProviderId())
		}
		sealed, err := ValidateFixedDenominatorSnapshot(snapshot)
		if err != nil {
			return nil, aisearch.FixedDenominator{}, false, fmt.Errorf("validate fixed denominator: %w", err)
		}
		return snapshot, sealed, true, nil
	}
	// Offline runners preserve their historical diagnostic behavior, but never
	// mint acceptance evidence from ordinary suite labels. Production wiring
	// supplies the durable reader and requires its independently reviewed row.
	return nil, aisearch.FixedDenominator{}, false, nil
}

// SuiteWithFixedDenominator overlays the immutable reviewed cases onto the
// current suite. New positives remain visible as pending candidates; removed
// or edited reviewed cases are restored from the snapshot and therefore remain
// queryable and scoreable.
func SuiteWithFixedDenominator(live *evalv1.EvalSuite, snapshot *evalv1.FixedDenominatorSnapshot) (*evalv1.EvalSuite, error) {
	if live == nil {
		return nil, fmt.Errorf("live suite is required")
	}
	if _, err := ValidateFixedDenominatorSnapshot(snapshot); err != nil {
		return nil, err
	}
	if live.GetSuiteId() != snapshot.GetSuiteId() {
		return nil, fmt.Errorf("fixed denominator suite %q does not match live suite %q", snapshot.GetSuiteId(), live.GetSuiteId())
	}
	frozen := make(map[string]*evalv1.EvalCase, len(snapshot.GetCases()))
	for _, c := range snapshot.GetCases() {
		frozen[c.GetCaseId()] = proto.Clone(c).(*evalv1.EvalCase)
	}
	current := proto.Clone(live).(*evalv1.EvalSuite)
	merged := make([]*evalv1.EvalCase, 0, len(live.GetCases())+len(frozen))
	// Preserve the review snapshot's order at the front. The shared grader
	// consumes the cached response queue in the same order, including duplicate
	// query/scope pairs with different expected-document labels.
	for _, c := range snapshot.GetCases() {
		merged = append(merged, proto.Clone(c).(*evalv1.EvalCase))
	}
	for _, c := range current.GetCases() {
		if c == nil {
			continue
		}
		if _, ok := frozen[c.GetCaseId()]; ok {
			continue
		}
		if positiveProtoCase(c) && !isExcludedEvaluationCase(c) {
			pending := proto.Clone(c).(*evalv1.EvalCase)
			pending.Status = "candidate"
			pending.Tags = append(pending.GetTags(), "fixed-denominator-pending-review")
			merged = append(merged, pending)
			continue
		}
		merged = append(merged, proto.Clone(c).(*evalv1.EvalCase))
	}
	current.Cases = merged
	return current, nil
}

func applyFixedDenominatorMetrics(aggregate *evalv1.EvalAggregate, report *aisearch.SuiteReport) {
	if aggregate == nil || report == nil {
		return
	}
	aggregate.RetrievalDenominator = int32(report.GradeablePositives)
	aggregate.RetrievalHits = int32(report.Hits)
	if report.GradeablePositives > 0 {
		recall := report.Recall
		aggregate.FixedDenominatorRecall = &recall
	}
}

func validateReviewEvidenceURI(raw string) error {
	value := strings.TrimSpace(raw)
	parsed, err := url.Parse(value)
	if err != nil || parsed == nil || parsed.Scheme == "" || parsed.Host == "" || parsed.User != nil || parsed.Fragment != "" {
		return fmt.Errorf("review_evidence_uri must be an absolute URI with a host and no credentials or fragment")
	}
	return nil
}
