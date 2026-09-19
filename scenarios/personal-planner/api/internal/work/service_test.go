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
func (f *fakeRepo) Get(context.Context, string) (WorkItem, error) { return WorkItem{}, nil }
func (f *fakeRepo) List(context.Context, int) ([]WorkItem, error) { return nil, nil }
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
