package feedback

import (
	"path/filepath"
	"testing"
)

func newStoreInTempDir(t *testing.T) (*Store, string) {
	t.Helper()
	root := t.TempDir()
	resolve := func(name string) string {
		return filepath.Join(root, "initiatives", name)
	}
	return NewStore(resolve), root
}

func TestStore_SaveAndLoadRound(t *testing.T) {
	store, _ := newStoreInTempDir(t)
	round := Round{
		InitiativeName: "ui-rewrite",
		Number:         1,
		Slug:           "hello",
		Type:           RoundTypeFeedback,
		Status:         RoundStatusAwaitingUser,
		Submission: Submission{
			Text:      "fix it",
			CreatedAt: "2026-04-23T00:00:00Z",
		},
		CreatedAt: "2026-04-23T00:00:00Z",
		UpdatedAt: "2026-04-23T00:00:00Z",
	}
	if err := store.SaveRound(round); err != nil {
		t.Fatalf("save: %v", err)
	}
	got, err := store.LoadRound("ui-rewrite", 1)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if got.Submission.Text != "fix it" || got.Status != RoundStatusAwaitingUser {
		t.Fatalf("unexpected round: %+v", got)
	}
}

func TestStore_LoadRound_ReturnsNotFound(t *testing.T) {
	store, _ := newStoreInTempDir(t)
	_, err := store.LoadRound("missing", 1)
	if err != ErrRoundNotFound {
		t.Fatalf("expected ErrRoundNotFound, got %v", err)
	}
}

func TestStore_NextRoundNumber_FromEmpty(t *testing.T) {
	store, _ := newStoreInTempDir(t)
	n, err := store.NextRoundNumber("any")
	if err != nil {
		t.Fatalf("next: %v", err)
	}
	if n != 1 {
		t.Fatalf("expected 1, got %d", n)
	}
}

func TestStore_NextRoundNumber_Increments(t *testing.T) {
	store, _ := newStoreInTempDir(t)
	for i := 1; i <= 3; i++ {
		if err := store.SaveRound(Round{
			InitiativeName: "i",
			Number:         i,
			Slug:           "",
			Status:         RoundStatusAwaitingUser,
			CreatedAt:      "2026-04-23T00:00:00Z",
			UpdatedAt:      "2026-04-23T00:00:00Z",
		}); err != nil {
			t.Fatalf("save %d: %v", i, err)
		}
	}
	n, err := store.NextRoundNumber("i")
	if err != nil {
		t.Fatalf("next: %v", err)
	}
	if n != 4 {
		t.Fatalf("expected 4, got %d", n)
	}
}

func TestStore_ListRounds_SortsAscending(t *testing.T) {
	store, _ := newStoreInTempDir(t)
	for _, n := range []int{3, 1, 2} {
		if err := store.SaveRound(Round{
			InitiativeName: "i",
			Number:         n,
			Slug:           "x",
			CreatedAt:      "2026-04-23T00:00:00Z",
			UpdatedAt:      "2026-04-23T00:00:00Z",
		}); err != nil {
			t.Fatalf("save %d: %v", n, err)
		}
	}
	rounds, err := store.ListRounds("i")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(rounds) != 3 || rounds[0].Number != 1 || rounds[2].Number != 3 {
		t.Fatalf("unexpected order: %+v", rounds)
	}
}

func TestStore_ReserveRound_AllocatesDistinctSlots(t *testing.T) {
	store, _ := newStoreInTempDir(t)

	// Reserve sequentially; each call should bump the number AND create
	// the dir on disk.
	got := make(map[int]string, 3)
	for i := 0; i < 3; i++ {
		n, dir, err := store.ReserveRound("ui-rewrite", "ui-feedback")
		if err != nil {
			t.Fatalf("reserve %d: %v", i, err)
		}
		if _, dup := got[n]; dup {
			t.Fatalf("duplicate round number %d returned", n)
		}
		got[n] = dir
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 unique numbers, got %v", got)
	}
}

func TestStore_ReserveRound_IsRaceSafe(t *testing.T) {
	store, _ := newStoreInTempDir(t)
	const concurrency = 8

	type result struct {
		number int
		err    error
	}
	resCh := make(chan result, concurrency)
	start := make(chan struct{})
	for i := 0; i < concurrency; i++ {
		go func() {
			<-start
			n, _, err := store.ReserveRound("ui-rewrite", "ui-feedback")
			resCh <- result{number: n, err: err}
		}()
	}
	close(start)

	seen := make(map[int]struct{}, concurrency)
	for i := 0; i < concurrency; i++ {
		r := <-resCh
		if r.err != nil {
			t.Fatalf("reserve err: %v", r.err)
		}
		if _, dup := seen[r.number]; dup {
			t.Fatalf("two reservations returned same number %d", r.number)
		}
		seen[r.number] = struct{}{}
	}
	if len(seen) != concurrency {
		t.Fatalf("expected %d unique numbers, got %d (%v)", concurrency, len(seen), seen)
	}
}

func TestStore_DeleteRound_Idempotent(t *testing.T) {
	store, _ := newStoreInTempDir(t)
	if err := store.DeleteRound("i", 99); err != nil {
		t.Fatalf("delete missing: %v", err)
	}
	if err := store.SaveRound(Round{
		InitiativeName: "i",
		Number:         1,
		Slug:           "x",
		CreatedAt:      "t",
		UpdatedAt:      "t",
	}); err != nil {
		t.Fatal(err)
	}
	if err := store.DeleteRound("i", 1); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := store.LoadRound("i", 1); err != ErrRoundNotFound {
		t.Fatalf("expected not-found after delete, got %v", err)
	}
}

func TestSanitize_StripsUnsafeChars(t *testing.T) {
	cases := map[string]string{
		"UI Rewrite!":       "ui-rewrite",
		"  multi  space  ":  "multi-space",
		"a//b":              "a-b",
		"":                  "",
		"only---hyphens---": "only-hyphens",
	}
	for in, want := range cases {
		if got := Sanitize(in); got != want {
			t.Fatalf("Sanitize(%q) = %q, want %q", in, got, want)
		}
	}
}
