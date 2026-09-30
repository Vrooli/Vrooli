package databasetest

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"sync"
	"testing"

	apidb "github.com/vrooli/api-core/database"
)

func TestFakeExecerRecordsQueriesInOrder(t *testing.T) {
	execer := &FakeExecer{}

	for _, query := range []string{"CREATE TABLE a();", "CREATE TABLE b();"} {
		result, err := execer.ExecContext(context.Background(), query)
		if err != nil {
			t.Fatalf("ExecContext(%q): %v", query, err)
		}
		if result == nil {
			t.Fatalf("ExecContext(%q) returned nil result", query)
		}
	}

	got := execer.SnapshotQueries()
	want := []string{"CREATE TABLE a();", "CREATE TABLE b();"}
	if !slices.Equal(got, want) {
		t.Fatalf("queries = %v, want %v", got, want)
	}
	if calls := execer.ExecCalls.Load(); calls != int64(len(want)) {
		t.Fatalf("ExecCalls = %d, want %d", calls, len(want))
	}
}

func TestFakeExecerAllCallErrorPreemptsRecording(t *testing.T) {
	sentinel := errors.New("database unavailable")
	execer := &FakeExecer{}
	execer.SetExecErr(sentinel)

	_, err := execer.ExecContext(context.Background(), "CREATE TABLE a();")
	if !errors.Is(err, sentinel) {
		t.Fatalf("ExecContext error = %v, want %v", err, sentinel)
	}
	if got := execer.SnapshotQueries(); len(got) != 0 {
		t.Fatalf("queries = %v, want empty", got)
	}
	if calls := execer.ExecCalls.Load(); calls != 1 {
		t.Fatalf("ExecCalls = %d, want 1", calls)
	}
}

func TestFakeExecerOrderedFailureOnlyFailsRequestedCall(t *testing.T) {
	sentinel := errors.New("fail second call")
	execer := &FakeExecer{FailOnCall: 2, FailErr: sentinel}

	if _, err := execer.ExecContext(context.Background(), "one"); err != nil {
		t.Fatalf("first call: %v", err)
	}
	if _, err := execer.ExecContext(context.Background(), "two"); !errors.Is(err, sentinel) {
		t.Fatalf("second call error = %v, want %v", err, sentinel)
	}
	if _, err := execer.ExecContext(context.Background(), "three"); err != nil {
		t.Fatalf("third call: %v", err)
	}

	got := execer.SnapshotQueries()
	want := []string{"one", "three"}
	if !slices.Equal(got, want) {
		t.Fatalf("queries = %v, want %v", got, want)
	}
}

func TestFakeExecerOrderedFailureDefaultsToNonNilError(t *testing.T) {
	execer := &FakeExecer{FailOnCall: 1}

	_, err := execer.ExecContext(context.Background(), "one")
	if !errors.Is(err, sql.ErrConnDone) {
		t.Fatalf("error = %v, want sql.ErrConnDone", err)
	}
}

func TestFakeExecerConcurrentCalls(t *testing.T) {
	execer := &FakeExecer{}
	const workers = 32

	var wg sync.WaitGroup
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		i := i
		go func() {
			defer wg.Done()
			if _, err := execer.ExecContext(context.Background(), fmt.Sprintf("query-%02d", i)); err != nil {
				t.Errorf("ExecContext: %v", err)
			}
		}()
	}
	wg.Wait()

	if calls := execer.ExecCalls.Load(); calls != workers {
		t.Fatalf("ExecCalls = %d, want %d", calls, workers)
	}
	if got := len(execer.SnapshotQueries()); got != workers {
		t.Fatalf("recorded queries = %d, want %d", got, workers)
	}
}

func TestFakeExecerIntegratesWithEnsureSchemas(t *testing.T) {
	execer := &FakeExecer{}
	err := apidb.EnsureSchemas(
		context.Background(),
		execer,
		apidb.SchemaProviderFunc(func() string { return "CREATE TABLE a();" }),
		apidb.SchemaProviderFunc(func() string { return "" }),
		apidb.SchemaProviderFunc(func() string { return "CREATE TABLE b();" }),
	)
	if err != nil {
		t.Fatalf("EnsureSchemas: %v", err)
	}

	got := execer.SnapshotQueries()
	want := []string{"CREATE TABLE a();", "CREATE TABLE b();"}
	if !slices.Equal(got, want) {
		t.Fatalf("queries = %v, want %v", got, want)
	}
}
