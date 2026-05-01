package mocks

import (
	"context"
	"errors"
	"testing"
	"time"

	"agent-manager/internal/repository"
)

func TestFakeStatsRepository_Defaults(t *testing.T) {
	stats := NewFakeStatsRepository()

	counts, err := stats.GetRunStatusCounts(context.Background(), repository.StatsFilter{})
	if err != nil {
		t.Fatalf("GetRunStatusCounts returned error: %v", err)
	}
	if counts == nil {
		t.Fatal("expected default status counts")
	}

	duration, err := stats.GetDurationStats(context.Background(), repository.StatsFilter{})
	if err != nil {
		t.Fatalf("GetDurationStats returned error: %v", err)
	}
	if duration == nil {
		t.Fatal("expected default duration stats")
	}
}

func TestFakeStatsRepository_ErrorKnobs(t *testing.T) {
	stats := NewFakeStatsRepository()
	want := errors.New("stats unavailable")
	stats.StatusCountsErr = want

	_, err := stats.GetRunStatusCounts(context.Background(), repository.StatsFilter{})
	if !errors.Is(err, want) {
		t.Fatalf("expected configured error, got %v", err)
	}
}

func TestFakeStatsRepository_ReturnsSliceCopies(t *testing.T) {
	stats := NewFakeStatsRepository()
	stats.PopularModels = []string{"claude"}

	got, err := stats.GetPopularModels(context.Background(), timeNow(), 10)
	if err != nil {
		t.Fatalf("GetPopularModels returned error: %v", err)
	}
	got[0] = "changed"

	if stats.PopularModels[0] != "claude" {
		t.Fatal("expected stored model slice to be protected from caller mutation")
	}
}

func timeNow() time.Time {
	return time.Time{}
}
