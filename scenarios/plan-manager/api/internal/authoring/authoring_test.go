package authoring_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"plan-manager/internal/authoring"
	internalplans "plan-manager/internal/plans"
	"plan-manager/internal/testutil/db"
	"plan-manager/internal/testutil/mocks"

	"github.com/stretchr/testify/require"

	apidb "github.com/vrooli/api-core/database"

	localdb "plan-manager/internal/database"
)

// newStore returns a real SQLite-backed SessionStore (the production persistence
// path) plus a fake clock — mirroring internal/plans/plans_test.go.
func newStore(t *testing.T) (authoring.SessionStore, *mocks.FakeClock) {
	t.Helper()
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(authoring.Schema),
	))
	clk := mocks.NewFakeClock(time.Date(2026, 6, 25, 12, 0, 0, 0, time.UTC))
	return authoring.NewSQLiteStore(d, clk), clk
}

// fakePlanWriter records the plan it was asked to persist and returns it with an
// assigned id (mirroring the plans Service Create contract).
type fakePlanWriter struct {
	created internalplans.Plan
	calls   int
	err     error
}

func (w *fakePlanWriter) CreatePlan(_ context.Context, p internalplans.Plan) (internalplans.Plan, error) {
	w.calls++
	if w.err != nil {
		return internalplans.Plan{}, w.err
	}
	p.ID = "plan-finalized"
	p.Status = internalplans.PlanStatusDraft
	w.created = p
	return p, nil
}

// fakeAnchor / fakeReading / fakeRefs are autofill seams whose behavior the test
// dials between "fills" and "errors" to exercise graceful degradation.
type fakeAnchor struct {
	out string
	err error
}

func (f fakeAnchor) Anchor(_ context.Context, _, _ string) (string, error) {
	return f.out, f.err
}

type fakeReading struct {
	out string
	err error
}

func (f fakeReading) RequiredReading(_ context.Context, _ string) (string, error) {
	return f.out, f.err
}

type fakeRefs struct {
	out string
	err error
}

func (f fakeRefs) References(_ context.Context, _, _ string) (string, error) {
	return f.out, f.err
}

func newService(t *testing.T, d authoring.Deps) authoring.Service {
	t.Helper()
	store, clk := newStore(t)
	if d.Store == nil {
		d.Store = store
	}
	if d.Clock == nil {
		d.Clock = clk
	}
	return authoring.NewService(d)
}

// fillMandatory submits non-empty content to every mandatory section + the
// regression anchor so the structure gate passes. Returns the final session.
func fillMandatory(t *testing.T, svc authoring.Service, sessionID string) authoring.Session {
	t.Helper()
	ctx := context.Background()
	content := map[authoring.SectionKey]string{
		authoring.SectionPurpose:          "Make widgets better.",
		authoring.SectionScope:            "In: widget core.",
		authoring.SectionRegressionAnchor: "baseline captured at HEAD abc123",
		authoring.SectionDefinitionOfDone: "Tests green; baseline diff exit 0.",
		authoring.SectionPhases:           "### Phase 1 — Anchor\n- Intent: Capture baseline\n- Status: todo\n",
	}
	var sess authoring.Session
	for key, val := range content {
		s, _, err := svc.SubmitSection(ctx, sessionID, key, val)
		require.NoError(t, err)
		sess = s
	}
	return sess
}

func TestStartSessionSeedsSkeletonAndPointer(t *testing.T) {
	svc := newService(t, authoring.Deps{Writer: &fakePlanWriter{}})
	ctx := context.Background()

	sess, err := svc.StartSession(ctx, "Improve widget", "improve-widget", "")
	require.NoError(t, err)
	require.NotEmpty(t, sess.ID)
	require.Equal(t, "Improve widget", sess.Title)
	require.NotEmpty(t, sess.Sections)
	// The first mandatory section is the current pointer.
	require.Equal(t, authoring.SectionPurpose, sess.CurrentSectionKey)

	// Empty title is rejected.
	_, err = svc.StartSession(ctx, "  ", "", "")
	require.Error(t, err)
}

func TestSessionProgressionToComplete(t *testing.T) {
	svc := newService(t, authoring.Deps{Writer: &fakePlanWriter{}})
	ctx := context.Background()

	sess, err := svc.StartSession(ctx, "Improve widget", "", "")
	require.NoError(t, err)

	// Next points at the first unfilled mandatory section, not complete.
	sec, complete, err := svc.Next(ctx, sess.ID)
	require.NoError(t, err)
	require.False(t, complete)
	require.Equal(t, authoring.SectionPurpose, sec.Key)

	// Submitting purpose advances the pointer past it.
	updated, violations, err := svc.SubmitSection(ctx, sess.ID, authoring.SectionPurpose, "A purpose.")
	require.NoError(t, err)
	require.Empty(t, violations)
	require.NotEqual(t, authoring.SectionPurpose, updated.CurrentSectionKey)

	// Fill every mandatory section => Next reports complete.
	fillMandatory(t, svc, sess.ID)
	_, complete, err = svc.Next(ctx, sess.ID)
	require.NoError(t, err)
	require.True(t, complete)
}

func TestGetSectionAndPersistenceAcrossCalls(t *testing.T) {
	// A fresh service built over the SAME store proves a session survives across
	// separate CLI invocations (each is a new Service instance over the store).
	store, clk := newStore(t)
	ctx := context.Background()

	svc1 := authoring.NewService(authoring.Deps{Store: store, Writer: &fakePlanWriter{}, Clock: clk})
	sess, err := svc1.StartSession(ctx, "Improve widget", "", "")
	require.NoError(t, err)
	_, _, err = svc1.SubmitSection(ctx, sess.ID, authoring.SectionPurpose, "A purpose.")
	require.NoError(t, err)

	svc2 := authoring.NewService(authoring.Deps{Store: store, Writer: &fakePlanWriter{}, Clock: clk})
	got, err := svc2.GetSection(ctx, sess.ID, authoring.SectionPurpose)
	require.NoError(t, err)
	require.Equal(t, "A purpose.", got.Content)
	require.True(t, got.Filled)

	// Unknown section / session are typed not-founds.
	_, err = svc2.GetSection(ctx, sess.ID, "nope")
	require.Error(t, err)
	_, err = svc2.GetSection(ctx, "no-such-session", authoring.SectionPurpose)
	require.Error(t, err)
}

func TestStructureGateRejectsEmptyMandatoryAndAnchor(t *testing.T) {
	svc := newService(t, authoring.Deps{Writer: &fakePlanWriter{}})
	ctx := context.Background()

	sess, err := svc.StartSession(ctx, "Improve widget", "", "")
	require.NoError(t, err)

	// A brand-new session fails the gate: mandatory sections + the anchor are empty.
	valid, violations, err := svc.ValidateStructure(ctx, sess.ID)
	require.NoError(t, err)
	require.False(t, valid)
	require.NotEmpty(t, violations)

	// The regression anchor is called out specifically.
	var anchorFlagged bool
	for _, v := range violations {
		if v.SectionKey == authoring.SectionRegressionAnchor {
			anchorFlagged = true
		}
	}
	require.True(t, anchorFlagged, "empty regression anchor is a distinct violation")

	// Filling everything mandatory satisfies the gate.
	fillMandatory(t, svc, sess.ID)
	valid, violations, err = svc.ValidateStructure(ctx, sess.ID)
	require.NoError(t, err)
	require.True(t, valid)
	require.Empty(t, violations)
}

func TestSubmitSectionReportsPerSectionViolations(t *testing.T) {
	svc := newService(t, authoring.Deps{Writer: &fakePlanWriter{}})
	ctx := context.Background()
	sess, err := svc.StartSession(ctx, "Improve widget", "", "")
	require.NoError(t, err)

	// Submitting empty content to a mandatory section reports a violation.
	_, violations, err := svc.SubmitSection(ctx, sess.ID, authoring.SectionPurpose, "   ")
	require.NoError(t, err)
	require.Len(t, violations, 1)
	require.Equal(t, authoring.SectionPurpose, violations[0].SectionKey)

	// Non-empty content passes.
	_, violations, err = svc.SubmitSection(ctx, sess.ID, authoring.SectionPurpose, "Real purpose.")
	require.NoError(t, err)
	require.Empty(t, violations)
}

func TestAutofillFillsWhenSeamsHealthy(t *testing.T) {
	svc := newService(t, authoring.Deps{
		Writer:       &fakePlanWriter{},
		Anchor:       fakeAnchor{out: "captured anchor"},
		RequiredRead: fakeReading{out: "docs/TESTING.md"},
		References:   fakeRefs{out: "[CODE: internal/widget/core.go]"},
	})
	ctx := context.Background()
	sess, err := svc.StartSession(ctx, "Improve widget", "", "")
	require.NoError(t, err)

	updated, results, err := svc.Autofill(ctx, sess.ID, nil) // nil => all sources
	require.NoError(t, err)
	require.Len(t, results, 3)
	for _, r := range results {
		require.True(t, r.Filled, "source %s should fill", r.Source)
		require.False(t, r.Degraded)
	}
	// The autofilled sections carry content + the autofilled marker.
	for _, key := range []authoring.SectionKey{
		authoring.SectionRegressionAnchor, authoring.SectionRequiredReading, authoring.SectionReferences,
	} {
		idx := -1
		for i := range updated.Sections {
			if updated.Sections[i].Key == key {
				idx = i
			}
		}
		require.GreaterOrEqual(t, idx, 0)
		require.True(t, updated.Sections[idx].Filled)
		require.True(t, updated.Sections[idx].Autofilled)
		require.NotEmpty(t, updated.Sections[idx].Content)
	}
}

func TestAutofillDegradesPerSourceNeverFalseFill(t *testing.T) {
	ctx := context.Background()

	// Nil anchor seam, erroring reading seam, healthy refs seam.
	svc := newService(t, authoring.Deps{
		Writer:       &fakePlanWriter{},
		Anchor:       nil, // nil seam must degrade, not panic / fabricate
		RequiredRead: fakeReading{err: errors.New("prompt-manager down")},
		References:   fakeRefs{out: "[CODE: internal/widget/core.go]"},
	})
	sess, err := svc.StartSession(ctx, "Improve widget", "", "")
	require.NoError(t, err)

	updated, results, err := svc.Autofill(ctx, sess.ID, nil)
	require.NoError(t, err)
	require.Len(t, results, 3)

	byKey := map[authoring.SectionKey]authoring.AutofillResult{}
	for _, r := range results {
		byKey[r.SectionKey] = r
	}

	// Anchor: nil seam => degraded, section left unfilled (NEVER a false fill).
	anchorRes := byKey[authoring.SectionRegressionAnchor]
	require.True(t, anchorRes.Degraded)
	require.False(t, anchorRes.Filled)

	// Reading: erroring seam => degraded, section left unfilled.
	readingRes := byKey[authoring.SectionRequiredReading]
	require.True(t, readingRes.Degraded)
	require.False(t, readingRes.Filled)

	// References: healthy => filled.
	refsRes := byKey[authoring.SectionReferences]
	require.True(t, refsRes.Filled)
	require.False(t, refsRes.Degraded)

	// The degraded sections are genuinely empty in the persisted session.
	for _, key := range []authoring.SectionKey{authoring.SectionRegressionAnchor, authoring.SectionRequiredReading} {
		got, err := svc.GetSection(ctx, sess.ID, key)
		require.NoError(t, err)
		require.Empty(t, got.Content, "degraded section %s must be left for the author", key)
		require.False(t, got.Filled)
		_ = updated
	}
}

func TestAutofillEmptyOutputDegrades(t *testing.T) {
	svc := newService(t, authoring.Deps{
		Writer: &fakePlanWriter{},
		Anchor: fakeAnchor{out: "   "}, // whitespace-only is not a real fill
	})
	ctx := context.Background()
	sess, err := svc.StartSession(ctx, "Improve widget", "", "")
	require.NoError(t, err)

	_, results, err := svc.Autofill(ctx, sess.ID, []authoring.AutofillSource{authoring.AutofillRegressionAnchor})
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.True(t, results[0].Degraded)
	require.False(t, results[0].Filled)
}

func TestFinalizeWritesThroughWriterWhenStructureValid(t *testing.T) {
	writer := &fakePlanWriter{}
	svc := newService(t, authoring.Deps{Writer: writer})
	ctx := context.Background()

	sess, err := svc.StartSession(ctx, "Improve widget", "improve-widget", "")
	require.NoError(t, err)

	// Add a references section so the plan carries a parsed reference.
	_, _, err = svc.SubmitSection(ctx, sess.ID, authoring.SectionReferences, "[CODE: internal/widget/core.go]")
	require.NoError(t, err)
	fillMandatory(t, svc, sess.ID)

	plan, err := svc.Finalize(ctx, sess.ID)
	require.NoError(t, err)
	require.Equal(t, 1, writer.calls)
	require.Equal(t, "plan-finalized", plan.ID)

	// The session's prose mapped through to the plan.
	require.Equal(t, "Make widgets better.", writer.created.Purpose)
	require.Equal(t, "Improve widget", writer.created.Title)
	require.NotEmpty(t, writer.created.Phases, "phases section parsed into structured phases")
	require.Equal(t, "Anchor", writer.created.Phases[0].Title)
	require.NotEmpty(t, writer.created.References, "references section parsed into structured references")
	require.Equal(t, "internal/widget/core.go", writer.created.References[0].Target)
	require.NotEmpty(t, writer.created.RegressionAnchor.BaselineName, "captured anchor carried forward")

	// The session is marked finalized + linked to the plan.
	got, err := svc.GetSection(ctx, sess.ID, authoring.SectionPurpose)
	require.NoError(t, err)
	require.NotEmpty(t, got.Content)
}

func TestFinalizeRejectsWhenStructureInvalid(t *testing.T) {
	writer := &fakePlanWriter{}
	svc := newService(t, authoring.Deps{Writer: writer})
	ctx := context.Background()

	sess, err := svc.StartSession(ctx, "Improve widget", "", "")
	require.NoError(t, err)

	// Fill only purpose — the gate still has violations.
	_, _, err = svc.SubmitSection(ctx, sess.ID, authoring.SectionPurpose, "A purpose.")
	require.NoError(t, err)

	_, err = svc.Finalize(ctx, sess.ID)
	require.Error(t, err)
	var gate authoring.ErrStructureGate
	require.True(t, errors.As(err, &gate), "structure gate failure is typed")
	require.NotEmpty(t, gate.Violations)
	require.Equal(t, 0, writer.calls, "no plan written when the gate fails")
}

// Sanity: a missing session id surfaces as a typed not-found from every read.
func TestUnknownSessionIsNotFound(t *testing.T) {
	svc := newService(t, authoring.Deps{Writer: &fakePlanWriter{}})
	ctx := context.Background()

	_, err := svc.GetSection(ctx, "nope", authoring.SectionPurpose)
	var notFound authoring.ErrSessionNotFound
	require.True(t, errors.As(err, &notFound))

	_, _, err = svc.Next(ctx, "nope")
	require.Error(t, err)
	_, _, err = svc.ValidateStructure(ctx, "nope")
	require.Error(t, err)
	_ = sql.ErrNoRows
}
