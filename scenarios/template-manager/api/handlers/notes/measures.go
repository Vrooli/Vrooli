package notes

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"template-manager/internal/clock"
	internalnotes "template-manager/internal/notes"

	"github.com/vrooli/api-core/database"
	measures "github.com/vrooli/measures-go"
	measuresv1 "github.com/vrooli/vrooli/packages/proto/gen/go/measures/v1"
)

// MeasureName is the canonical "<domain>.<command>" id of the notes measure —
// it must match the manifest group + command ("notes" + "count") so the
// measures-health behavioral probe (which executes by manifest-derived name)
// finds the registered compute func.
const MeasureName = "notes.count"

// notesCountDeclaration is the runtime serve declaration for the `notes count`
// measure. It mirrors the manifest `measure` block (the static SSOT cli-health
// and measures-health validate); the curated prose is kept in sync by the
// manifest cross-check test. The single `window` param is the canonical
// time_window type, which is what grades the measure at full tier.
func notesCountDeclaration() measures.MeasureDeclaration {
	return measures.MeasureDeclaration{
		Name:   MeasureName,
		Domain: "notes",
		Intent: "How many notes were created in a given time window.",
		Questions: []string{
			"how many notes were created this week",
			"notes created last month",
			"how many notes did we add in the last 7 days",
		},
		Params: map[string]measures.Param{
			"window": {
				Name:    "window",
				Type:    measures.ParamTypeTimeWindow,
				Default: string(measures.TokenThisWeek),
			},
		},
		Result: measures.Result{
			Kind:            measures.ResultScalar,
			ValueField:      "count",
			Unit:            "notes",
			SummaryTemplate: "{count} notes created ({window})",
		},
		Effect:      measures.EffectRead,
		RunEligible: true,
		Service:     "NotesService",
		Method:      "CountNotes",
	}
}

// MeasuresHandler builds the measures-go serve registry for the notes domain
// and returns it as an http.Handler to mount at /measures (see api/main.go).
// It owns its own repository + service over the shared db handle so the
// measure travels with the notes domain — when a real scenario deletes
// features/notes it deletes this file and the one mount line, with no central
// residue. A real multi-domain scenario registers each domain's measures on a
// single shared registry instead.
func MeasuresHandler(db *database.RoutedDB, clk clock.Clock) (http.Handler, error) {
	svc := internalnotes.NewService(internalnotes.NewSQLiteRepository(db, clk))
	reg := measures.NewRegistry(measures.WithClock(clk.Now))
	if err := registerNotesCount(reg, svc, clk); err != nil {
		return nil, err
	}
	return reg.Handler(), nil
}

// registerNotesCount wires the notes.count declaration to its compute func.
// The compute func resolves the canonical time_window token deterministically
// (no LLM, explicit UTC) and counts via the same service method the CountNotes
// RPC uses, stamping mandatory provenance describing the executed range.
func registerNotesCount(reg *measures.Registry, svc internalnotes.Service, clk clock.Clock) error {
	decl := notesCountDeclaration()
	return reg.Register(decl, func(ctx context.Context, req measures.MeasureRequest) (measures.MeasureResult, error) {
		token := measures.TimeWindowToken(req.Params["window"])
		if token == "" {
			token = measures.TokenThisWeek
		}
		rng, err := measures.ResolveToken(token, clk.Now(), time.UTC)
		if err != nil {
			return measures.MeasureResult{}, err
		}
		n, err := svc.CountInWindow(ctx, rng.From, rng.To)
		if err != nil {
			return measures.MeasureResult{}, err
		}
		return measures.MeasureResult{
			Value: strconv.Itoa(n),
			Provenance: measures.Provenance{
				ExecutedQuery: fmt.Sprintf(
					"SELECT COUNT(*) FROM notes WHERE created_at >= %q AND created_at < %q",
					rng.From.UTC().Format(time.RFC3339), rng.To.UTC().Format(time.RFC3339),
				),
			},
		}, nil
	})
}

// resolveCountWindow maps a request's canonical TimeWindow to a concrete
// [from, to) range, defaulting to this_week when the window is unset. Shared by
// the CountNotes RPC handler and exercised by the measure compute path so both
// resolve dates identically.
func resolveCountWindow(tw *measuresv1.TimeWindow, now time.Time) (measures.Range, error) {
	if tw == nil || tw.GetWindow() == nil {
		return measures.ResolveToken(measures.TokenThisWeek, now, time.UTC)
	}
	return measures.ResolveTimeWindow(tw, now, time.UTC)
}
