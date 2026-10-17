package workspace

import (
	"context"
	"testing"
	"time"
)

type fakeRepo struct {
	profile      Profile
	input        UpdateInput
	availability Availability
}

func (f *fakeRepo) Get(context.Context) (Profile, error) { return f.profile, nil }
func (f *fakeRepo) Update(_ context.Context, input UpdateInput) (Profile, error) {
	f.input = input
	f.profile.Revision++
	f.profile.Timezone = input.Timezone
	f.profile.WeekStart = input.WeekStart
	f.profile.DailyCapacityMinutes = input.DailyCapacityMinutes
	f.profile.ReserveMinutes = input.ReserveMinutes
	f.profile.FocusSessionMinutes = input.FocusSessionMinutes
	return f.profile, nil
}

func (f *fakeRepo) ListAvailability(context.Context) (Availability, error) {
	return f.availability, nil
}

func (f *fakeRepo) ReplaceAvailability(_ context.Context, input AvailabilityInput) (Availability, error) {
	f.availability = Availability{Windows: input.Windows, Exceptions: input.Exceptions, Revision: f.profile.Revision + 1}
	return f.availability, nil
}

func TestServiceUpdatePersistsExplicitPlanningProfile(t *testing.T) {
	repo := &fakeRepo{profile: Profile{Revision: 1, UpdatedAt: time.Unix(1, 0)}}
	p, err := NewService(repo).Update(context.Background(), UpdateInput{
		Timezone: "America/New_York", WeekStart: "monday", DailyCapacityMinutes: 420,
		ReserveMinutes: 60, FocusSessionMinutes: 45, ExpectedRevision: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if p.Timezone != "America/New_York" || p.DailyCapacityMinutes != 420 || repo.input.ReserveMinutes != 60 {
		t.Fatalf("profile=%#v input=%#v", p, repo.input)
	}
}

func TestServiceRejectsAmbiguousAvailability(t *testing.T) {
	repo := &fakeRepo{profile: Profile{Revision: 3}}
	_, err := NewService(repo).ReplaceAvailability(context.Background(), AvailabilityInput{
		ExpectedRevision: 3,
		Windows: []AvailabilityWindow{
			{Weekday: 1, StartMinute: 540, EndMinute: 660, Timezone: "UTC"},
			{Weekday: 1, StartMinute: 600, EndMinute: 720, Timezone: "UTC"},
		},
	})
	if err == nil {
		t.Fatal("expected overlapping windows to be rejected")
	}
}

func TestServiceAcceptsProtectedException(t *testing.T) {
	repo := &fakeRepo{profile: Profile{Revision: 3}}
	a, err := NewService(repo).ReplaceAvailability(context.Background(), AvailabilityInput{
		ExpectedRevision: 3,
		Windows:          []AvailabilityWindow{{Weekday: 1, StartMinute: 540, EndMinute: 1020, Timezone: "America/New_York"}},
		Exceptions:       []AvailabilityException{{Date: "2026-09-21", StartMinute: 720, EndMinute: 780, Kind: "protected", Reason: "school pickup"}},
	})
	if err != nil || len(a.Exceptions) != 1 || a.Exceptions[0].Kind != "protected" {
		t.Fatalf("availability=%#v err=%v", a, err)
	}
}

func TestServiceRejectsUnsafeProfileValues(t *testing.T) {
	cases := []UpdateInput{
		{Timezone: "Not/AZone", WeekStart: "monday", DailyCapacityMinutes: 480, ReserveMinutes: 60, FocusSessionMinutes: 45},
		{Timezone: "UTC", WeekStart: "tuesday", DailyCapacityMinutes: 480, ReserveMinutes: 60, FocusSessionMinutes: 45},
		{Timezone: "UTC", WeekStart: "monday", DailyCapacityMinutes: 30, ReserveMinutes: 60, FocusSessionMinutes: 45},
		{Timezone: "UTC", WeekStart: "monday", DailyCapacityMinutes: 480, ReserveMinutes: 60, FocusSessionMinutes: 4},
	}
	for _, input := range cases {
		if _, err := NewService(&fakeRepo{}).Update(context.Background(), input); err == nil {
			t.Fatalf("expected validation error for %#v", input)
		}
	}
}
