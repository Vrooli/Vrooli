package calendar

import (
	"context"
	"testing"
)

type fakeRepository struct {
	item      Allocation
	created   Allocation
	createErr error
}

func (f *fakeRepository) ListToday(context.Context, string) (Today, error) { return Today{}, nil }
func (f *fakeRepository) ListRange(context.Context, string, string) (DateRange, error) {
	return DateRange{}, nil
}
func (f *fakeRepository) WorkItem(context.Context, string) (Allocation, error) { return f.item, nil }
func (f *fakeRepository) Create(_ context.Context, item Allocation) (Allocation, error) {
	f.created = item
	return item, f.createErr
}

func (f *fakeRepository) CarryForward(_ context.Context, in CarryForwardInput) (Allocation, error) {
	return Allocation{ID: "carried-1", CarriedFromID: in.AllocationID, LocalDate: in.TargetLocalDate, StartMinutes: in.StartMinutes}, nil
}
func (f *fakeRepository) ListRoutines(context.Context) ([]Routine, error) { return nil, nil }
func (f *fakeRepository) CreateRoutine(_ context.Context, in CreateRoutineInput) (Routine, error) {
	return Routine{Title: in.Title, Kind: in.Kind}, nil
}

func (f *fakeRepository) ListRoutineOccurrences(context.Context, string, string) ([]RoutineOccurrence, error) {
	return nil, nil
}

func (f *fakeRepository) SkipRoutineOccurrence(context.Context, string, string, int64) error {
	return nil
}

func (f *fakeRepository) RescheduleRoutineOccurrence(context.Context, string, string, int, int64) error {
	return nil
}

func TestServiceCreateRejectsInvalidPlacement(t *testing.T) {
	repo := &fakeRepository{item: Allocation{WorkItemID: "work-1", Title: "Draft"}}
	_, err := NewService(repo).Create(context.Background(), CreateInput{WorkItemID: "work-1", LocalDate: "2026-09-19", StartMinutes: 600, DurationMinutes: 0})
	if err == nil {
		t.Fatal("expected duration validation")
	}
}

func TestServiceCreateEnrichesAcceptedAllocationFromWork(t *testing.T) {
	repo := &fakeRepository{item: Allocation{WorkItemID: "work-1", Title: "Draft", SourceLabel: "Cadence"}}
	got, err := NewService(repo).Create(context.Background(), CreateInput{WorkItemID: "work-1", LocalDate: "2026-09-19", StartMinutes: 600, DurationMinutes: 45})
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != "Draft" || got.SourceLabel != "Cadence" || repo.created.State != "" {
		t.Fatalf("unexpected allocation %#v", got)
	}
}

func TestServiceListRangeRejectsReversedDates(t *testing.T) {
	_, err := NewService(&fakeRepository{}).ListRange(context.Background(), "2026-09-22", "2026-09-21")
	if err == nil {
		t.Fatal("expected reversed date range validation")
	}
}

func TestServiceCarryForwardValidatesDateAndStart(t *testing.T) {
	service := NewService(&fakeRepository{})
	if _, err := service.CarryForward(context.Background(), CarryForwardInput{AllocationID: "allocation-1", TargetLocalDate: "not-a-date", StartMinutes: 600}); err == nil {
		t.Fatal("expected date validation")
	}
	if _, err := service.CarryForward(context.Background(), CarryForwardInput{AllocationID: "allocation-1", TargetLocalDate: "2026-09-22", StartMinutes: 1440}); err == nil {
		t.Fatal("expected start validation")
	}
	got, err := service.CarryForward(context.Background(), CarryForwardInput{AllocationID: "allocation-1", TargetLocalDate: "2026-09-22", StartMinutes: 600})
	if err != nil || got.CarriedFromID != "allocation-1" {
		t.Fatalf("got=%#v err=%v", got, err)
	}
}

func TestServiceValidatesFixedAndFlexibleRoutines(t *testing.T) {
	service := NewService(&fakeRepository{})
	got, err := service.CreateRoutine(context.Background(), CreateRoutineInput{Title: "Weekly review", Kind: "fixed", Timezone: "America/New_York", StartDate: "2026-09-21", Weekdays: []int{1}, StartMinute: 540, DurationMinutes: 45})
	if err != nil || got.Kind != "fixed" {
		t.Fatalf("routine=%#v err=%v", got, err)
	}
	_, err = service.CreateRoutine(context.Background(), CreateRoutineInput{Title: "Runs", Kind: "flexible", Timezone: "UTC", StartDate: "2026-09-21", Weekdays: []int{1, 3, 5}, FrequencyPerWeek: 4, StartMinute: 540, DurationMinutes: 30})
	if err == nil {
		t.Fatal("expected flexible frequency to fit eligible weekdays")
	}
}

func TestServiceRejectsInvalidRoutineSkip(t *testing.T) {
	service := NewService(&fakeRepository{})
	if err := service.SkipRoutineOccurrence(context.Background(), "", "2026-09-21", 1); err == nil {
		t.Fatal("expected routine id validation")
	}
	if err := service.SkipRoutineOccurrence(context.Background(), "routine-1", "not-a-date", 1); err == nil {
		t.Fatal("expected date validation")
	}
	if err := service.SkipRoutineOccurrence(context.Background(), "routine-1", "2026-09-21", 0); err == nil {
		t.Fatal("expected revision validation")
	}
}
