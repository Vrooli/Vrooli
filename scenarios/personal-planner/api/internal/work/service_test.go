package work

import (
	"context"
	"testing"
)

type fakeRepo struct{ created WorkItem }

func (f *fakeRepo) Create(_ context.Context, item WorkItem) (WorkItem, error) {
	f.created = item
	return item, nil
}
func (f *fakeRepo) Get(context.Context, string) (WorkItem, error)             { return WorkItem{}, nil }
func (f *fakeRepo) List(context.Context, int) ([]WorkItem, error)             { return nil, nil }
func (f *fakeRepo) Snooze(context.Context, string, string, string) error      { return nil }
func (f *fakeRepo) UpdateEstimate(context.Context, string, int, string) error { return nil }
func (f *fakeRepo) Complete(context.Context, string) error                    { return nil }
func TestServiceCreateValidatesAndTrims(t *testing.T) {
	repo := &fakeRepo{}
	got, err := NewService(repo).Create(context.Background(), CreateInput{Title: "  Draft story  ", RemainingMinutes: 45})
	if err != nil || got.Title != "Draft story" || repo.created.Title != "Draft story" {
		t.Fatalf("got %#v err %v", got, err)
	}
}

func TestServiceRejectsNegativeEffort(t *testing.T) {
	if _, err := NewService(&fakeRepo{}).Create(context.Background(), CreateInput{Title: "Draft", RemainingMinutes: -1}); err == nil {
		t.Fatal("expected negative effort rejection")
	}
}

func TestServiceSnoozeValidatesDate(t *testing.T) {
	if err := NewService(&fakeRepo{}).Snooze(context.Background(), "work-1", "tomorrow", "not_now"); err == nil {
		t.Fatal("expected invalid snooze date rejection")
	}
}

func TestServiceCompleteValidatesID(t *testing.T) {
	if err := NewService(&fakeRepo{}).Complete(context.Background(), " "); err == nil {
		t.Fatal("expected missing ID rejection")
	}
}

func TestServiceUpdateEstimateValidatesInput(t *testing.T) {
	service := NewService(&fakeRepo{})
	if err := service.UpdateEstimate(context.Background(), " ", 30, ""); err == nil {
		t.Fatal("expected missing ID rejection")
	}
	if err := service.UpdateEstimate(context.Background(), "work-1", -1, ""); err == nil {
		t.Fatal("expected negative estimate rejection")
	}
}
