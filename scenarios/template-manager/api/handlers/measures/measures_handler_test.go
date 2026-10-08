package measures

import (
	"context"
	"errors"
	"testing"
	"time"

	"connectrpc.com/connect"

	tmmeasuresv1 "github.com/vrooli/vrooli/packages/proto/gen/go/template-manager/v1/measures"
	"github.com/vrooli/vrooli/scenarios/template-manager/api/internal/catalog"
)

type fakeStore struct {
	openDebt   int64
	streak     int64
	buckets    []catalog.StandingBucket
	versionLag int64
	err        error

	gotWindow     catalog.MeasureWindow
	gotTemplateID string
}

func (f *fakeStore) CountOpenDebt(_ context.Context, w catalog.MeasureWindow) (int64, error) {
	f.gotWindow = w
	return f.openDebt, f.err
}

func (f *fakeStore) DeepValidateGreenStreak(_ context.Context, templateID string) (int64, error) {
	f.gotTemplateID = templateID
	return f.streak, f.err
}

func (f *fakeStore) FleetStandingDistribution(context.Context) ([]catalog.StandingBucket, error) {
	return f.buckets, f.err
}

func (f *fakeStore) MaxVersionLag(context.Context) (int64, error) {
	return f.versionLag, f.err
}

func fixedNow() time.Time { return time.Date(2026, 7, 14, 12, 0, 0, 0, time.UTC) }

func TestOpenDebtCountReturnsCountAndResolvesDefaultWindow(t *testing.T) {
	store := &fakeStore{openDebt: 7}
	h := NewHandler(store, fixedNow)

	resp, err := h.OpenDebtCount(context.Background(), connect.NewRequest(&tmmeasuresv1.OpenDebtCountRequest{}))
	if err != nil {
		t.Fatalf("OpenDebtCount: %v", err)
	}
	if resp.Msg.Count != 7 {
		t.Fatalf("count = %d, want 7", resp.Msg.Count)
	}
	// A nil window resolves to the default this-week range, which must be non-zero.
	if store.gotWindow.From.IsZero() || store.gotWindow.To.IsZero() {
		t.Fatalf("default window not resolved: %+v", store.gotWindow)
	}
	if !store.gotWindow.To.After(store.gotWindow.From) {
		t.Fatalf("window end must be after start: %+v", store.gotWindow)
	}
}

func TestOpenDebtCountStoreErrorMapsToInternal(t *testing.T) {
	h := NewHandler(&fakeStore{err: errors.New("boom")}, fixedNow)

	_, err := h.OpenDebtCount(context.Background(), connect.NewRequest(&tmmeasuresv1.OpenDebtCountRequest{}))
	if connect.CodeOf(err) != connect.CodeInternal {
		t.Fatalf("code = %v, want Internal", connect.CodeOf(err))
	}
}

func TestDeepValidateGreenStreakForwardsTemplateID(t *testing.T) {
	store := &fakeStore{streak: 4}
	h := NewHandler(store, fixedNow)

	resp, err := h.DeepValidateGreenStreak(context.Background(), connect.NewRequest(&tmmeasuresv1.DeepValidateGreenStreakRequest{TemplateId: "react-vite"}))
	if err != nil {
		t.Fatalf("DeepValidateGreenStreak: %v", err)
	}
	if resp.Msg.Streak != 4 {
		t.Fatalf("streak = %d, want 4", resp.Msg.Streak)
	}
	if store.gotTemplateID != "react-vite" {
		t.Fatalf("template id not forwarded: %q", store.gotTemplateID)
	}
}

func TestFleetStandingDistributionMapsBuckets(t *testing.T) {
	store := &fakeStore{buckets: []catalog.StandingBucket{
		{Standing: "current", Count: 5},
		{Standing: "drifted", Count: 2},
	}}
	h := NewHandler(store, fixedNow)

	resp, err := h.FleetStandingDistribution(context.Background(), connect.NewRequest(&tmmeasuresv1.FleetStandingDistributionRequest{}))
	if err != nil {
		t.Fatalf("FleetStandingDistribution: %v", err)
	}
	if len(resp.Msg.Buckets) != 2 || resp.Msg.Buckets[0].Standing != "current" || resp.Msg.Buckets[0].Count != 5 {
		t.Fatalf("buckets mapped wrong: %#v", resp.Msg.Buckets)
	}
}

func TestMaxVersionLagReturnsLag(t *testing.T) {
	h := NewHandler(&fakeStore{versionLag: 3}, fixedNow)

	resp, err := h.MaxVersionLag(context.Background(), connect.NewRequest(&tmmeasuresv1.MaxVersionLagRequest{}))
	if err != nil {
		t.Fatalf("MaxVersionLag: %v", err)
	}
	if resp.Msg.Lag != 3 {
		t.Fatalf("lag = %d, want 3", resp.Msg.Lag)
	}
}

func TestMaxVersionLagStoreErrorMapsToInternal(t *testing.T) {
	h := NewHandler(&fakeStore{err: errors.New("boom")}, fixedNow)

	_, err := h.MaxVersionLag(context.Background(), connect.NewRequest(&tmmeasuresv1.MaxVersionLagRequest{}))
	if connect.CodeOf(err) != connect.CodeInternal {
		t.Fatalf("code = %v, want Internal", connect.CodeOf(err))
	}
}
