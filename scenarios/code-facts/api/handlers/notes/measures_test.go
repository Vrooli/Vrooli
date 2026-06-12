package notes

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	measures "github.com/vrooli/measures-go"

	mocks "code-facts/internal/notes/mocks"
	clockmocks "code-facts/internal/testutil/mocks"
)

// TestNotesCountDeclarationIsFullTier pins the contract measures-health grades:
// the declaration is well-formed and its only param is the canonical
// time_window (so the measure earns full tier). A regression here would drop
// the template below the "ship measure-ready at top tier" guarantee.
func TestNotesCountDeclarationIsFullTier(t *testing.T) {
	decl := notesCountDeclaration()
	require.NoError(t, decl.Validate())
	require.Equal(t, "notes.count", decl.Name)
	require.Equal(t, "notes", decl.Domain)
	require.Equal(t, measures.EffectRead, decl.Effect)
	require.True(t, decl.RunEligible)
	require.Equal(t, "count", decl.Result.ValueField)

	w, ok := decl.Params["window"]
	require.True(t, ok, "window param must be declared")
	require.True(t, w.IsCanonical(), "window must be the canonical time_window type for full tier")
}

// TestRegisterNotesCountServesAnswer is the unit-level mirror of the
// measures-health behavioral probe: register → execute through the serve
// registry → assert the scalar value, that the window was resolved
// deterministically, and that mandatory provenance is stamped.
func TestRegisterNotesCountServesAnswer(t *testing.T) {
	now := time.Date(2026, 5, 6, 12, 0, 0, 0, time.UTC)
	clk := clockmocks.NewFakeClock(now)
	svc := &mocks.FakeService{CountOut: 9}

	reg := measures.NewRegistry(measures.WithClock(clk.Now))
	require.NoError(t, registerNotesCount(reg, svc, clk))

	res, err := reg.Execute(context.Background(), measures.MeasureRequest{
		Measure: "notes.count",
		Params:  map[string]string{"window": "this_week"},
	})
	require.NoError(t, err)
	require.Equal(t, "9", res.Value)
	require.NotEmpty(t, res.Provenance.ExecutedQuery)
	require.False(t, res.Provenance.ComputedAt.IsZero(), "serve helper must stamp computed_at")

	// Deterministic window: start of the 2026-05-06 ISO week is Monday 05-04.
	require.Len(t, svc.CountWindows, 1)
	require.Equal(t, time.Date(2026, 5, 4, 0, 0, 0, 0, time.UTC), svc.CountWindows[0][0].UTC())
	require.Equal(t, now, svc.CountWindows[0][1].UTC())
}

// TestRegisterNotesCountDefaultsWindow proves an omitted window resolves to the
// declared default (this_week) rather than erroring — the auto-answer path
// must never abstain on a defaulted canonical param.
func TestRegisterNotesCountDefaultsWindow(t *testing.T) {
	now := time.Date(2026, 5, 6, 12, 0, 0, 0, time.UTC)
	clk := clockmocks.NewFakeClock(now)
	svc := &mocks.FakeService{CountOut: 2}

	reg := measures.NewRegistry(measures.WithClock(clk.Now))
	require.NoError(t, registerNotesCount(reg, svc, clk))

	res, err := reg.Execute(context.Background(), measures.MeasureRequest{Measure: "notes.count"})
	require.NoError(t, err)
	require.Equal(t, "2", res.Value)
	require.Len(t, svc.CountWindows, 1)
	require.Equal(t, time.Date(2026, 5, 4, 0, 0, 0, 0, time.UTC), svc.CountWindows[0][0].UTC())
}
