package mocks

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"{{SCENARIO_ID}}/internal/notes"

	"github.com/google/uuid"
)

func TestFakeRepositoryCreate(t *testing.T) {
	ctx := context.Background()
	repo := NewFakeRepository()
	created, err := repo.Create(ctx, notes.Note{Title: "first"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if _, err := uuid.Parse(created.ID); err != nil {
		t.Errorf("Create() generated invalid UUID %q: %v", created.ID, err)
	}
	if len(repo.Items) != 1 || !reflect.DeepEqual(repo.Items[0], created) {
		t.Errorf("Create() items = %#v, want one item %#v", repo.Items, created)
	}
	if got := repo.CreateCalls.Load(); got != 1 {
		t.Errorf("CreateCalls = %d, want 1", got)
	}

	repo.CreateErr = errors.New("create failed")
	if got, err := repo.Create(ctx, notes.Note{ID: "kept"}); !errors.Is(err, repo.CreateErr) || !reflect.DeepEqual(got, notes.Note{}) {
		t.Errorf("Create() with arranged error = (%#v, %v), want zero note and %v", got, err, repo.CreateErr)
	}
	if len(repo.Items) != 1 {
		t.Errorf("Create() appended after arranged error; items = %#v", repo.Items)
	}
}

func TestFakeRepositoryGet(t *testing.T) {
	ctx := context.Background()
	repo := NewFakeRepository()
	want := notes.Note{ID: "note-1", Title: "found"}
	repo.Items = []notes.Note{{ID: "other"}, want}
	if got, err := repo.Get(ctx, want.ID); err != nil || !reflect.DeepEqual(got, want) {
		t.Errorf("Get() = (%#v, %v), want (%#v, nil)", got, err, want)
	}
	var notFound notes.ErrNoteNotFound
	if _, err := repo.Get(ctx, "missing"); !errors.As(err, &notFound) || notFound.ID != "missing" {
		t.Errorf("Get() missing error = %v, want ErrNoteNotFound for missing", err)
	}
	repo.GetErr = errors.New("get failed")
	if _, err := repo.Get(ctx, want.ID); !errors.Is(err, repo.GetErr) {
		t.Errorf("Get() arranged error = %v, want %v", err, repo.GetErr)
	}
	if got := repo.GetCalls.Load(); got != 3 {
		t.Errorf("GetCalls = %d, want 3", got)
	}
}

func TestFakeRepositoryList(t *testing.T) {
	ctx := context.Background()
	repo := NewFakeRepository()
	if got, err := repo.List(ctx, 2); err != nil || got != nil {
		t.Errorf("List() empty = (%#v, %v), want (nil, nil)", got, err)
	}
	repo.Items = []notes.Note{{ID: "one"}, {ID: "two"}, {ID: "three"}}
	for _, limit := range []int{0, -1} {
		if got, err := repo.List(ctx, limit); err != nil || got != nil {
			t.Errorf("List(%d) = (%#v, %v), want (nil, nil)", limit, got, err)
		}
	}
	got, err := repo.List(ctx, 2)
	if err != nil || len(got) != 2 || got[0].ID != "one" || got[1].ID != "two" {
		t.Fatalf("List(2) = (%#v, %v), want first two items", got, err)
	}
	got[0].ID = "changed"
	if repo.Items[0].ID != "one" {
		t.Error("List() returned a slice backed by repository Items")
	}
	got, err = repo.List(ctx, 99)
	if err != nil || len(got) != len(repo.Items) {
		t.Errorf("List(99) = (%#v, %v), want all items", got, err)
	}
	repo.ListErr = errors.New("list failed")
	if got, err := repo.List(ctx, 2); !errors.Is(err, repo.ListErr) || got != nil {
		t.Errorf("List() arranged error = (%#v, %v), want (nil, %v)", got, err, repo.ListErr)
	}
	if got := repo.ListCalls.Load(); got != 6 {
		t.Errorf("ListCalls = %d, want 6", got)
	}
}

func TestFakeRepositoryCount(t *testing.T) {
	ctx := context.Background()
	repo := NewFakeRepository()
	from := time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC)
	to := from.Add(time.Hour)
	repo.CountOut = 4
	if got, err := repo.Count(ctx, from, to); err != nil || got != 4 {
		t.Errorf("Count() = (%d, %v), want (4, nil)", got, err)
	}
	if len(repo.CountWindows) != 1 || repo.CountWindows[0] != [2]time.Time{from, to} {
		t.Errorf("CountWindows = %#v, want recorded interval", repo.CountWindows)
	}
	repo.CountErr = errors.New("count failed")
	if got, err := repo.Count(ctx, from, to); !errors.Is(err, repo.CountErr) || got != 0 {
		t.Errorf("Count() arranged error = (%d, %v), want (0, %v)", got, err, repo.CountErr)
	}
	if len(repo.CountWindows) != 2 {
		t.Errorf("CountWindows length = %d, want 2 including failed call", len(repo.CountWindows))
	}
}
