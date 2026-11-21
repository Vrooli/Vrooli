package main

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/vrooli/api-core/schedule"
	measures "github.com/vrooli/measures-go"
	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/store"
)

const receiptMeasureWindow = 24 * time.Hour

// receiptMeasuresHandler exposes the durable receipt projection through the
// repository-wide measures contract. The measure intentionally has no caller
// supplied window yet: the first production signal is a conservative, fixed
// recent window whose provenance is explicit and whose result is safe for
// unattended reads.
func receiptMeasuresHandler(aggregateStore store.ReceiptAggregateStore, clk schedule.Clock) (http.Handler, error) {
	registry := measures.NewRegistry(measures.WithClock(clk.Now))
	declaration := measures.MeasureDeclaration{
		Name:      "receipts.invocations",
		Domain:    "receipts",
		Intent:    "How many durable receipt invocations were observed in the last 24 hours.",
		Questions: []string{"how many receipt invocations occurred in the last day", "how many durable receipt invocations were observed"},
		Result: measures.Result{
			Kind:            measures.ResultScalar,
			ValueField:      "count",
			Unit:            "invocations",
			SummaryTemplate: "{count} receipt invocations in the last 24 hours",
		},
		Effect:      measures.EffectRead,
		RunEligible: true,
		Service:     "ReceiptAggregateService",
		Method:      "AggregateReceipts",
	}
	if err := registry.Register(declaration, func(ctx context.Context, _ measures.MeasureRequest) (measures.MeasureResult, error) {
		now := clk.Now().UTC()
		if aggregateStore == nil {
			return measures.MeasureResult{}, nil
		}
		aggregates, err := aggregateStore.AggregateReceipts(ctx, store.ReceiptAggregateFilter{
			Since: now.Add(-receiptMeasureWindow),
			Until: now,
		})
		if err != nil {
			return measures.MeasureResult{}, err
		}
		var total int64
		for _, aggregate := range aggregates {
			if aggregate != nil {
				total += aggregate.GetInvocationCount()
			}
		}
		return measures.MeasureResult{
			Value: strconv.FormatInt(total, 10),
			Provenance: measures.Provenance{
				ExecutedQuery: "receipt aggregates grouped by target scenario and operation over the previous 24 hours",
			},
		}, nil
	}); err != nil {
		return nil, err
	}
	return registry.Handler(), nil
}
