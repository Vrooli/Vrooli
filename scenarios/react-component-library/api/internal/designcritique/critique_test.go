package designcritique

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
	dbtest "github.com/vrooli/api-core/databasetest"
)

type verifyFunc func(context.Context, Target, Evidence) error

func (f verifyFunc) Verify(ctx context.Context, t Target, e Evidence) error { return f(ctx, t, e) }
func reviewFixture() Review {
	r := Review{Target: Target{Scenario: "demo", DesignID: "home", Revision: strings.Repeat("a", 64), RenderHash: strings.Repeat("b", 64)}, RubricVersion: RubricVersion, PolicyVersion: PolicyVersion, Critic: Critic{Kind: "model", ID: "critic", Version: "1", Model: "fixture", Profile: "review-v1"}, Findings: []Finding{}}
	for _, d := range Dimensions() {
		r.Ratings = append(r.Ratings, Rating{Dimension: d, Score: 4, Rationale: "Fixture rationale", Evidence: []Evidence{{CaptureID: "capture", Artifact: "bas:producer:image", Region: "$page", State: "ready", Width: 390, Height: 844}}})
	}
	return r
}
func TestVisualFloorCannotHideFailedDimensionOrEstablishAcceptance(t *testing.T) {
	r := reviewFixture()
	calls := 0
	verifier := verifyFunc(func(context.Context, Target, Evidence) error { calls++; return nil })
	result, err := Evaluate(context.Background(), r, verifier)
	require.NoError(t, err)
	require.True(t, result.VisualFloorMet)
	require.False(t, result.AcceptanceEstablished)
	require.Equal(t, 1, calls)
	r.Ratings[4].Score = 2
	result, err = Evaluate(context.Background(), r, verifier)
	require.NoError(t, err)
	require.False(t, result.VisualFloorMet)
	require.Equal(t, []string{"state_recovery"}, result.BlockingDimensions)
	require.Equal(t, 2, result.MinimumScore)
	r.Ratings[4].Score = 4
	r.Findings = []Finding{{Dimension: "state_recovery", Severity: "major", Rationale: "Error state lacks recovery", Correction: "Add a reachable retry action", Evidence: r.Ratings[4].Evidence[0]}}
	result, err = Evaluate(context.Background(), r, verifier)
	require.NoError(t, err)
	require.False(t, result.VisualFloorMet)
	require.Equal(t, []int{0}, result.BlockingFindings)
}
func TestCritiqueRequiresCompleteAttributableEvidence(t *testing.T) {
	for _, name := range []string{"invalid evidence render", "missing dimension", "duplicate dimension", "missing evidence", "missing rationale", "missing critic", "missing profile", "unknown policy", "invalid score", "finding lacks correction"} {
		t.Run(name, func(t *testing.T) {
			r := reviewFixture()
			switch name {
			case "invalid evidence render":
				r.Ratings[0].Evidence[0].RenderHash = "arbitrary"
			case "missing dimension":
				r.Ratings = r.Ratings[:7]
			case "duplicate dimension":
				r.Ratings[1].Dimension = r.Ratings[0].Dimension
			case "missing evidence":
				r.Ratings[0].Evidence = nil
			case "missing rationale":
				r.Ratings[0].Rationale = ""
			case "missing critic":
				r.Critic.ID = ""
			case "missing profile":
				r.Critic.Profile = ""
			case "unknown policy":
				r.PolicyVersion = "other"
			case "invalid score":
				r.Ratings[0].Score = 5
			case "finding lacks correction":
				r.Findings = []Finding{{Dimension: "hierarchy", Severity: "major", Rationale: "Unclear", Evidence: r.Ratings[0].Evidence[0]}}
			}
			require.Error(t, Validate(r))
		})
	}
	_, err := Evaluate(context.Background(), reviewFixture(), nil)
	require.Error(t, err)
	_, err = Evaluate(context.Background(), reviewFixture(), verifyFunc(func(context.Context, Target, Evidence) error { return errors.New("wrong revision") }))
	require.ErrorContains(t, err, "wrong revision")
}

func TestCritiquePersistenceIsImmutableAndRejectsForgedAssessment(t *testing.T) {
	ctx := context.Background()
	db := dbtest.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(ctx, db, apidb.SchemaProviderFunc(Schema)))
	repo := NewSQLiteRepository(db)
	svc := Service{Repository: repo, Verifier: verifyFunc(func(context.Context, Target, Evidence) error { return nil })}
	review := reviewFixture()
	first, err := svc.Record(ctx, "review", review)
	require.NoError(t, err)
	again, err := svc.Record(ctx, "review", review)
	require.NoError(t, err)
	require.Equal(t, first, again)
	svc.Verifier = verifyFunc(func(context.Context, Target, Evidence) error { return errors.New("image expired") })
	recovered, err := svc.Record(ctx, "review", review)
	require.NoError(t, err)
	require.Equal(t, first, recovered)
	svc.Verifier = verifyFunc(func(context.Context, Target, Evidence) error { return nil })
	review.Ratings[0].Score = 2
	_, err = svc.Record(ctx, "review", review)
	require.ErrorIs(t, err, ErrConflict)
	stored, err := repo.Get(ctx, first.ID)
	require.NoError(t, err)
	require.Equal(t, first, stored)
	assessment := aggregate(review)
	assessment.AcceptanceEstablished = true
	_, err = repo.Create(ctx, "forged", review, assessment)
	require.ErrorContains(t, err, "aggregation")
	_, err = db.ExecContext(ctx, `UPDATE design_critiques SET review_hash='tampered' WHERE id=?`, first.ID)
	require.NoError(t, err)
	_, err = repo.Get(ctx, first.ID)
	require.ErrorContains(t, err, "integrity")
	svc.Verifier = verifyFunc(func(context.Context, Target, Evidence) error { return errors.New("image expired") })
	_, err = svc.Record(ctx, "unavailable", review)
	require.ErrorContains(t, err, "image expired")
}

func TestListCritiquesScopesAndPaginatesVerifiedRecords(t *testing.T) {
	ctx := context.Background()
	db := dbtest.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(ctx, db, apidb.SchemaProviderFunc(Schema)))
	repo := NewSQLiteRepository(db)
	review := reviewFixture()
	for i := 0; i < 23; i++ {
		review.Critic.ID = fmt.Sprintf("critic-%d", i)
		_, err := repo.Create(ctx, fmt.Sprintf("key-%d", i), review, aggregate(review))
		require.NoError(t, err)
	}
	other := review
	other.Target.RenderHash = strings.Repeat("c", 64)
	otherRecord, err := repo.Create(ctx, "other", other, aggregate(other))
	require.NoError(t, err)
	first, next, err := repo.List(ctx, review.Target, "")
	require.NoError(t, err)
	require.Len(t, first, 20)
	require.NotEmpty(t, next)
	second, last, err := repo.List(ctx, review.Target, next)
	require.NoError(t, err)
	require.Len(t, second, 3)
	require.Empty(t, last)
	seen := map[string]bool{}
	for _, record := range append(first, second...) {
		require.Equal(t, review.Target, record.Review.Target)
		require.False(t, seen[record.ID])
		seen[record.ID] = true
	}
	_, _, err = repo.List(ctx, review.Target, otherRecord.ID)
	require.ErrorContains(t, err, "another target")
	_, err = db.ExecContext(ctx, `UPDATE design_critiques SET review_hash='tampered' WHERE id=?`, first[0].ID)
	require.NoError(t, err)
	_, _, err = repo.List(ctx, review.Target, "")
	require.ErrorContains(t, err, "integrity")
}

func TestCritiqueChecksEveryDistinctRenderEvidence(t *testing.T) {
	review := reviewFixture()
	first := review.Ratings[0].Evidence[0]
	second := first
	second.RenderHash = strings.Repeat("c", 64)
	review.Ratings[0].Evidence = []Evidence{first, second}
	calls := 0
	_, err := Evaluate(context.Background(), review, verifyFunc(func(_ context.Context, _ Target, e Evidence) error {
		calls++
		if e.RenderHash != "" {
			return errors.New("alternate capture mismatch")
		}
		return nil
	}))
	require.ErrorContains(t, err, "alternate capture mismatch")
	require.Equal(t, 2, calls)
}
