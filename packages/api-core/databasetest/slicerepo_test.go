package databasetest

import (
	"context"
	"errors"
	"testing"
)

type sliceRepoItem struct {
	ID   string
	Name string
}

func newSliceRepo() *SliceRepo[sliceRepoItem] {
	return NewSliceRepo(
		func(item sliceRepoItem) string { return item.ID },
		func(item *sliceRepoItem, id string) { item.ID = id },
		func(id string) error { return errors.New("not found: " + id) },
	)
}

func TestSliceRepoCreateAssignsIDAndListHonorsLimit(t *testing.T) {
	repo := newSliceRepo()
	created, err := repo.Create(context.Background(), sliceRepoItem{Name: "alpha"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.ID == "" || len(repo.Items) != 1 {
		t.Fatalf("created = %#v, items = %d", created, len(repo.Items))
	}
	items, err := repo.List(context.Background(), 1)
	if err != nil || len(items) != 1 || items[0].Name != "alpha" {
		t.Fatalf("List = %#v, err = %v", items, err)
	}
}

func TestSliceRepoFailureInjectionPreventsMutation(t *testing.T) {
	repo := newSliceRepo()
	want := errors.New("create failed")
	repo.CreateErr = want
	if _, err := repo.Create(context.Background(), sliceRepoItem{}); !errors.Is(err, want) {
		t.Fatalf("Create err = %v, want %v", err, want)
	}
	if len(repo.Items) != 0 {
		t.Fatalf("failed Create mutated Items: %#v", repo.Items)
	}
}
