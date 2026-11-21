package eval

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"slices"
	"sort"
	"strings"
	"time"

	evalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/eval"
)

// IndependentReviewReceipt is a query-free attestation written to Source
// Ledger by a verified independent reviewer. It carries hashes and immutable
// source references, never query or result text.
type IndependentReviewReceipt struct {
	SchemaVersion int                            `json:"schema_version"`
	SuiteID       string                         `json:"suite_id"`
	ProviderID    string                         `json:"provider_id"`
	CorpusSHA256  string                         `json:"corpus_sha256"`
	ReviewerID    string                         `json:"reviewer_id"`
	Cases         []IndependentReviewReceiptCase `json:"cases"`
}

type IndependentReviewReceiptCase struct {
	CaseID      string                      `json:"case_id"`
	QuerySHA256 string                      `json:"query_sha256"`
	ExpectedIDs []string                    `json:"expected_ids"`
	Sources     []IndependentReviewIDSource `json:"sources"`
}

type IndependentReviewIDSource struct {
	ExpectedID   string `json:"expected_id"`
	SourceURI    string `json:"source_uri"`
	SourceSHA256 string `json:"source_sha256"`
}

// IndependentReviewTemplate is safe for source-owner routing: it exposes the
// immutable case IDs, query hashes, and labels but never the query strings.
type IndependentReviewTemplate struct {
	SchemaVersion int                             `json:"schema_version"`
	SuiteID       string                          `json:"suite_id"`
	ProviderID    string                          `json:"provider_id"`
	CorpusSHA256  string                          `json:"corpus_sha256"`
	Cases         []IndependentReviewTemplateCase `json:"cases"`
}

type IndependentReviewTemplateCase struct {
	CaseID      string                      `json:"case_id"`
	QuerySHA256 string                      `json:"query_sha256"`
	ExpectedIDs []string                    `json:"expected_ids"`
	Sources     []IndependentReviewIDSource `json:"sources"`
}

// BuildIndependentReviewTemplate emits only the metadata needed to bind a
// source-owner review. Query strings remain in the suite owner and are hashed
// in memory here.
func BuildIndependentReviewTemplate(suite *evalv1.EvalSuite) (*IndependentReviewTemplate, error) {
	if suite == nil {
		return nil, fmt.Errorf("suite is required")
	}
	byID := make(map[string]*evalv1.EvalCase, len(suite.GetCases()))
	for _, c := range suite.GetCases() {
		if c != nil {
			byID[c.GetCaseId()] = c
		}
	}
	ids := reviewedPositiveCaseIDs(suite)
	if len(ids) == 0 {
		return nil, fmt.Errorf("suite has no denominator-eligible positive cases")
	}
	template := &IndependentReviewTemplate{SchemaVersion: 1, SuiteID: suite.GetSuiteId(), ProviderID: suite.GetProviderId(), CorpusSHA256: corpusDigest(suite), Cases: make([]IndependentReviewTemplateCase, 0, len(ids))}
	for _, id := range ids {
		c := byID[id]
		if c == nil {
			return nil, fmt.Errorf("denominator-eligible case %q is missing", id)
		}
		queryDigest := sha256.Sum256([]byte(c.GetQuery()))
		template.Cases = append(template.Cases, IndependentReviewTemplateCase{
			CaseID: id, QuerySHA256: hex.EncodeToString(queryDigest[:]),
			ExpectedIDs: append([]string(nil), c.GetExpectIds()...),
			Sources:     reviewSourcePlaceholders(c.GetExpectIds()),
		})
	}
	return template, nil
}

// ValidateIndependentReviewReceipt checks every case and expected ID against
// the current Search Hub suite. query_sha256 is compared in memory; query text
// is neither returned nor included in the receipt.
func ValidateIndependentReviewReceipt(suite *evalv1.EvalSuite, reviewerID, verifiedActorID, verifiedStatus, entryCreatedAt string, body []byte) error {
	if suite == nil {
		return fmt.Errorf("suite is required")
	}
	if strings.TrimSpace(verifiedStatus) != "verified" || strings.TrimSpace(verifiedActorID) == "" {
		return fmt.Errorf("review receipt requires a verified Source Ledger actor")
	}
	if strings.TrimSpace(reviewerID) == "" || strings.TrimSpace(reviewerID) != strings.TrimSpace(verifiedActorID) {
		return fmt.Errorf("reviewer_id must match the verified Source Ledger actor")
	}
	if strings.EqualFold(strings.TrimSpace(reviewerID), strings.TrimSpace(suite.GetProviderId())) {
		return fmt.Errorf("reviewer must be independent of provider %q", suite.GetProviderId())
	}
	var receipt IndependentReviewReceipt
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&receipt); err != nil {
		return fmt.Errorf("decode independent review receipt: %w", err)
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return fmt.Errorf("independent review receipt contains trailing JSON")
	}
	if receipt.SchemaVersion != 1 || receipt.SuiteID != suite.GetSuiteId() || receipt.ProviderID != suite.GetProviderId() {
		return fmt.Errorf("review receipt suite/provider identity does not match current suite")
	}
	if !isSHA256(receipt.CorpusSHA256) || receipt.CorpusSHA256 != corpusDigest(suite) {
		return fmt.Errorf("review receipt corpus digest does not match current suite")
	}
	if strings.TrimSpace(receipt.ReviewerID) != strings.TrimSpace(reviewerID) {
		return fmt.Errorf("receipt reviewer does not match verified actor")
	}
	createdAt, err := time.Parse(time.RFC3339Nano, entryCreatedAt)
	if err != nil || createdAt.IsZero() {
		return fmt.Errorf("Source Ledger entry creation time is invalid")
	}
	wantIDs := reviewedPositiveCaseIDs(suite)
	if len(wantIDs) == 0 || len(receipt.Cases) != len(wantIDs) {
		return fmt.Errorf("receipt must cover every current positive case exactly once")
	}
	casesByID := make(map[string]*evalv1.EvalCase, len(suite.GetCases()))
	for _, c := range suite.GetCases() {
		if c != nil {
			casesByID[c.GetCaseId()] = c
		}
	}
	seen := make(map[string]struct{}, len(receipt.Cases))
	for _, item := range receipt.Cases {
		c := casesByID[item.CaseID]
		if c == nil || !slices.Contains(wantIDs, item.CaseID) {
			return fmt.Errorf("receipt contains an unknown or non-positive case")
		}
		if _, ok := seen[item.CaseID]; ok {
			return fmt.Errorf("receipt contains a duplicate case")
		}
		seen[item.CaseID] = struct{}{}
		if item.QuerySHA256 != querySHA256(c.GetQuery()) {
			return fmt.Errorf("receipt query digest does not match case %q", item.CaseID)
		}
		wantExpected := append([]string(nil), c.GetExpectIds()...)
		gotExpected := append([]string(nil), item.ExpectedIDs...)
		sort.Strings(wantExpected)
		sort.Strings(gotExpected)
		if len(gotExpected) == 0 || !slices.Equal(wantExpected, gotExpected) {
			return fmt.Errorf("receipt expected IDs do not match case %q", item.CaseID)
		}
		if err := validateExpectedIDSources(item.CaseID, wantExpected, item.Sources); err != nil {
			return err
		}
	}
	return nil
}

func reviewSourcePlaceholders(expectedIDs []string) []IndependentReviewIDSource {
	sources := make([]IndependentReviewIDSource, 0, len(expectedIDs))
	for _, id := range expectedIDs {
		sources = append(sources, IndependentReviewIDSource{ExpectedID: id})
	}
	return sources
}

func validateExpectedIDSources(caseID string, expectedIDs []string, sources []IndependentReviewIDSource) error {
	if len(sources) != len(expectedIDs) {
		return fmt.Errorf("case %q requires one source proof per expected ID", caseID)
	}
	want := make(map[string]struct{}, len(expectedIDs))
	for _, id := range expectedIDs {
		want[id] = struct{}{}
	}
	seen := make(map[string]struct{}, len(sources))
	for _, source := range sources {
		if _, ok := want[source.ExpectedID]; !ok {
			return fmt.Errorf("case %q source proof refers to an unexpected ID", caseID)
		}
		if _, ok := seen[source.ExpectedID]; ok {
			return fmt.Errorf("case %q contains duplicate source proof for an expected ID", caseID)
		}
		seen[source.ExpectedID] = struct{}{}
		if !isSHA256(source.SourceSHA256) || !validReviewSourceURI(source.SourceURI) {
			return fmt.Errorf("case %q expected ID %q requires a stable source URI and SHA-256", caseID, source.ExpectedID)
		}
	}
	return nil
}

func querySHA256(query string) string {
	digest := sha256.Sum256([]byte(query))
	return hex.EncodeToString(digest[:])
}

func validReviewSourceURI(raw string) bool {
	u, err := url.Parse(strings.TrimSpace(raw))
	return err == nil && u != nil && u.Scheme != "" && u.Host != "" && u.User == nil && u.RawQuery == "" && u.Fragment == ""
}
