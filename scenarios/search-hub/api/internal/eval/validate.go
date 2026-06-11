package eval

import (
	"strings"

	evalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/eval"
)

// Normalize applies a suite's defaults in place before validation and
// persistence: an unspecified state becomes "active"; identity/text fields are
// trimmed so two suites that differ only by trailing spaces don't create
// distinct rows. Normalize never rejects — Validate is the gate.
func Normalize(s *evalv1.EvalSuite) {
	if s == nil {
		return
	}
	s.SuiteId = strings.TrimSpace(s.SuiteId)
	s.ProviderId = strings.TrimSpace(s.ProviderId)
	s.Name = strings.TrimSpace(s.Name)
	s.State = strings.TrimSpace(s.State)
	if s.State == "" {
		s.State = "active"
	}
	for _, c := range s.Cases {
		if c == nil {
			continue
		}
		c.CaseId = strings.TrimSpace(c.CaseId)
		c.Query = strings.TrimSpace(c.Query)
		c.Status = strings.TrimSpace(c.Status)
	}
}

// Validate enforces the suite invariants the runner depends on. It assumes
// Normalize has already run.
//
//   - suite_id, provider_id are required (the runner resolves the provider).
//   - at least one case, each with a case_id and a non-empty query.
//   - case_ids must be unique within the suite (runs key per-case results on them).
//   - status is empty/"reviewed"/"candidate".
//   - a case is either positive (expects ids) or a
//     negative gibberish case (expect_no_strong_hit). A negative case with an
//     expect_id is contradictory and rejected.
//
// The first failing rule is returned as ErrInvalidSuite.
func Validate(s *evalv1.EvalSuite) error {
	if s == nil {
		return ErrInvalidSuite{Field: "suite", Reason: "required"}
	}
	if s.SuiteId == "" {
		return ErrInvalidSuite{Field: "suite_id", Reason: "required"}
	}
	if s.ProviderId == "" {
		return ErrInvalidSuite{Field: "provider_id", Reason: "required (the runner reuses that provider's endpoint)"}
	}
	if len(s.Cases) == 0 {
		return ErrInvalidSuite{Field: "cases", Reason: "at least one case is required"}
	}
	seen := make(map[string]struct{}, len(s.Cases))
	for i, c := range s.Cases {
		if c == nil {
			return ErrInvalidSuite{Field: "cases", Reason: "nil case"}
		}
		if c.CaseId == "" {
			return ErrInvalidSuite{Field: "cases.case_id", Reason: "required"}
		}
		if _, dup := seen[c.CaseId]; dup {
			return ErrInvalidSuite{Field: "cases.case_id", Reason: "duplicate case_id " + c.CaseId}
		}
		seen[c.CaseId] = struct{}{}
		if c.Query == "" {
			return ErrInvalidSuite{Field: "cases.query", Reason: "required for case " + c.CaseId}
		}
		switch c.GetStatus() {
		case "", "reviewed", "candidate":
		default:
			return ErrInvalidSuite{Field: "cases.status", Reason: "must be reviewed or candidate: " + c.CaseId}
		}
		if c.ExpectNoStrongHit {
			if len(c.ExpectIds) > 0 {
				return ErrInvalidSuite{Field: "cases.expect_ids", Reason: "a gibberish case (expect_no_strong_hit) must not also expect ids: " + c.CaseId}
			}
			if c.ExpectMaxScore <= 0 {
				return ErrInvalidSuite{Field: "cases.expect_max_score", Reason: "a gibberish case must set expect_max_score (the junk-rejection ceiling): " + c.CaseId}
			}
			continue
		}
		if (c.ExpectWithinTopK > 0 || c.ExpectMinScore > 0) && len(c.ExpectIds) == 0 {
			return ErrInvalidSuite{Field: "cases.expect_ids", Reason: "a positive case must set expect_ids: " + c.CaseId}
		}
		_ = i
	}
	return nil
}
