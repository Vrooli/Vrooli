package hostpressure

import (
	"context"
	"testing"
)

func TestReclaimOneBrakesSaturationAndRecyclesOneManagedIdleService(t *testing.T) {
	ctx := context.Background()
	p := []Process{{PID: 1, Name: "reranker", Resident: 100, Swapped: 1000}, {PID: 2, Name: "other", Resident: 100, Swapped: 900}}
	c := []ReclaimCandidate{{Service: "reranker", Process: p[0]}, {Service: "other", Process: p[1]}}
	var recycled []string
	d, err := ReclaimOne(ctx, p, c, ReclaimPolicy{MinimumSwapped: 500, Saturated: func(context.Context) (bool, error) { return false, nil }, Managed: func(context.Context, string) (bool, error) { return true, nil }, Serving: func(context.Context, string) (bool, error) { return false, nil }, Recycle: func(_ context.Context, s string) error { recycled = append(recycled, s); return nil }})
	if err != nil || d.Selected == nil || len(recycled) != 1 || recycled[0] != "reranker" {
		t.Fatalf("decision=%+v recycled=%v err=%v", d, recycled, err)
	}
	d, err = ReclaimOne(ctx, p, c, ReclaimPolicy{MinimumSwapped: 500, Saturated: func(context.Context) (bool, error) { return true, nil }, Recycle: func(context.Context, string) error { t.Fatal("recycled while saturated"); return nil }})
	if err != nil || d.Selected != nil || d.HeldReason == "" {
		t.Fatalf("saturated decision=%+v err=%v", d, err)
	}
}
