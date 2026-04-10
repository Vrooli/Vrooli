package repository

import "testing"

func TestStatsFilter_ZeroValueIsValid(t *testing.T) {
	var filter StatsFilter
	if !filter.Window.Start.IsZero() {
		t.Fatal("expected zero-value Start time")
	}
	if !filter.Window.End.IsZero() {
		t.Fatal("expected zero-value End time")
	}
}
