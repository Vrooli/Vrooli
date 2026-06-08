package eval

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	evalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/eval"
)

// codes extracts the finding codes in order for terse assertions.
func codes(ws []*evalv1.AdequacyWarning) []string {
	out := make([]string, len(ws))
	for i, w := range ws {
		out[i] = w.GetCode()
	}
	return out
}

func hasCode(ws []*evalv1.AdequacyWarning, code string) bool {
	for _, w := range ws {
		if w.GetCode() == code {
			return true
		}
	}
	return false
}

// positiveCase builds a strong positive case with a unique query + expected id.
func positiveCase(i int) *evalv1.EvalCase {
	return &evalv1.EvalCase{
		CaseId:    fmt.Sprintf("p%d", i),
		Query:     fmt.Sprintf("query number %d", i),
		Tags:      []string{"strong"},
		ExpectIds: []string{fmt.Sprintf("id-%d", i)},
	}
}

// richSuite returns a suite with n positive cases (mixed difficulty) + one
// negative — an adequate corpus apart from possibly being under the count floor.
func richSuite(n int) *evalv1.EvalSuite {
	s := &evalv1.EvalSuite{SuiteId: "s", ProviderId: "p"}
	for i := 0; i < n; i++ {
		c := positiveCase(i)
		if i%2 == 0 {
			c.Tags = []string{"weak-real"}
		}
		s.Cases = append(s.Cases, c)
	}
	s.Cases = append(s.Cases, &evalv1.EvalCase{
		CaseId: "neg", Query: "asdfqwer gibberish", Tags: []string{"gibberish"},
		ExpectNoStrongHit: true, ExpectMaxScore: 0.5,
	})
	return s
}

func TestCheckAdequacy_AdequateCorpusIsClean(t *testing.T) {
	got := CheckAdequacy(richSuite(MinPositiveCases), nil)
	require.Empty(t, got, "an adequate corpus produces no warnings: %v", codes(got))
}

func TestCheckAdequacy_TooFewCases(t *testing.T) {
	got := CheckAdequacy(richSuite(MinPositiveCases-1), nil)
	require.True(t, hasCode(got, "too_few_cases"), "got %v", codes(got))
}

func TestCheckAdequacy_NoNegatives(t *testing.T) {
	s := &evalv1.EvalSuite{SuiteId: "s", ProviderId: "p"}
	for i := 0; i < MinPositiveCases; i++ {
		c := positiveCase(i)
		if i%2 == 0 {
			c.Tags = []string{"weak-real"}
		}
		s.Cases = append(s.Cases, c)
	}
	got := CheckAdequacy(s, nil)
	require.True(t, hasCode(got, "no_negatives"), "got %v", codes(got))
	require.False(t, hasCode(got, "too_few_cases"))
}

func TestCheckAdequacy_ThinDifficulty(t *testing.T) {
	// All positive cases tagged "strong" — one band only.
	s := &evalv1.EvalSuite{SuiteId: "s", ProviderId: "p"}
	for i := 0; i < MinPositiveCases; i++ {
		s.Cases = append(s.Cases, positiveCase(i)) // all "strong"
	}
	s.Cases = append(s.Cases, &evalv1.EvalCase{
		CaseId: "neg", Query: "zzz", Tags: []string{"gibberish"}, ExpectNoStrongHit: true, ExpectMaxScore: 0.5,
	})
	got := CheckAdequacy(s, nil)
	require.True(t, hasCode(got, "thin_difficulty"), "got %v", codes(got))
}

func TestCheckAdequacy_DuplicateQuery(t *testing.T) {
	s := richSuite(MinPositiveCases)
	// Force a duplicate (case-insensitive + whitespace-insensitive).
	s.Cases[0].Query = "Restart   THE service"
	s.Cases[1].Query = "restart the service"
	got := CheckAdequacy(s, nil)
	require.True(t, hasCode(got, "duplicate_query"), "got %v", codes(got))
	// The finding's subject is the normalized duplicate query.
	for _, w := range got {
		if w.GetCode() == "duplicate_query" {
			require.Equal(t, "restart the service", w.GetSubject())
		}
	}
}

func TestCheckAdequacy_CoverageGapOnlyWithStrata(t *testing.T) {
	s := richSuite(MinPositiveCases)
	// Tag a couple of cases with a stratum so it's "covered".
	s.Cases[0].Tags = append(s.Cases[0].Tags, "type:command")

	// No strata supplied → no coverage finding (structural-only path).
	require.False(t, hasCode(CheckAdequacy(s, nil), "coverage_gap"))

	// Supplied strata: command is covered, doc is not.
	got := CheckAdequacy(s, []string{"type:command", "type:doc"})
	require.True(t, hasCode(got, "coverage_gap"), "got %v", codes(got))
	gaps := 0
	for _, w := range got {
		if w.GetCode() == "coverage_gap" {
			gaps++
			require.Equal(t, "type:doc", w.GetSubject())
		}
	}
	require.Equal(t, 1, gaps, "only the uncovered stratum is flagged")
}

func TestCheckAdequacy_NeverErrorsOnNilOrEmpty(t *testing.T) {
	require.Nil(t, CheckAdequacy(nil, nil))
	// An empty suite is degenerate but adequacy still must not panic; it reports
	// the structural shortfalls (too few + no negatives).
	got := CheckAdequacy(&evalv1.EvalSuite{SuiteId: "s", ProviderId: "p"}, nil)
	require.True(t, hasCode(got, "too_few_cases"))
	require.True(t, hasCode(got, "no_negatives"))
}
