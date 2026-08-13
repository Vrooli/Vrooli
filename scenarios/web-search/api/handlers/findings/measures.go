package findings

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	internalfindings "web-search/internal/findings"

	"github.com/vrooli/api-core/schedule"

	"github.com/vrooli/api-core/database"
	measures "github.com/vrooli/measures-go"
	measuresv1 "github.com/vrooli/vrooli/packages/proto/gen/go/measures/v1"
)

// MeasureName is the canonical "<domain>.<command>" id of the findings measure.
// It must match the manifest group + command ("findings" + "count") so the
// measures-health behavioral probe finds the registered compute func.
const MeasureName = "findings.count"

func findingsCountDeclaration() measures.MeasureDeclaration {
	return measures.MeasureDeclaration{
		Name:   MeasureName,
		Domain: "findings",
		Intent: "How many findings were captured in a given time window.",
		Questions: []string{
			"how many findings were captured this week",
			"findings added last month",
			"how many findings did we record in the last 7 days",
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
			Unit:            "findings",
			SummaryTemplate: "{count} findings captured ({window})",
		},
		Effect:      measures.EffectRead,
		RunEligible: true,
		Service:     "FindingsService",
		Method:      "CountFindings",
	}
}

// MeasuresHandler builds the measures-go serve registry for the findings domain
// and returns it as an http.Handler to mount at /measures (see api/main.go).
func MeasuresHandler(db *database.RoutedDB, clk schedule.Clock) (http.Handler, error) {
	svc := internalfindings.NewService(internalfindings.NewSQLiteRepository(db, clk))
	reg := measures.NewRegistry(measures.WithClock(clk.Now))
	if err := registerFindingsCount(reg, svc, clk); err != nil {
		return nil, err
	}
	return reg.Handler(), nil
}

func registerFindingsCount(reg *measures.Registry, svc internalfindings.Service, clk schedule.Clock) error {
	decl := findingsCountDeclaration()
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
					"SELECT COUNT(*) FROM findings WHERE created_at >= %q AND created_at < %q",
					rng.From.UTC().Format(time.RFC3339), rng.To.UTC().Format(time.RFC3339),
				),
			},
		}, nil
	})
}

// resolveCountWindow maps a request's canonical TimeWindow to a concrete
// [from, to) range, defaulting to this_week when unset. Shared by the
// CountFindings RPC handler and the measure compute path so both resolve dates
// identically.
func resolveCountWindow(tw *measuresv1.TimeWindow, now time.Time) (measures.Range, error) {
	if tw == nil || tw.GetWindow() == nil {
		return measures.ResolveToken(measures.TokenThisWeek, now, time.UTC)
	}
	return measures.ResolveTimeWindow(tw, now, time.UTC)
}
