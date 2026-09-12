package database

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
)

// fakeExecer records every ExecContext call and can be configured to
// return an error on the Nth call (1-based). Avoids pulling a real
// SQLite driver into api-core's deps just for these tests.
type fakeExecer struct {
	calls   []string
	failOn  int // 1-based; 0 means never fail
	failErr error
}

func (f *fakeExecer) ExecContext(_ context.Context, query string, _ ...any) (sql.Result, error) {
	f.calls = append(f.calls, query)
	if f.failOn != 0 && len(f.calls) == f.failOn {
		return nil, f.failErr
	}
	return nil, nil
}

func TestSchemaProviderFunc_Schema(t *testing.T) {
	got := SchemaProviderFunc(func() string { return "x" }).Schema()
	if got != "x" {
		t.Fatalf("SchemaProviderFunc.Schema() = %q, want %q", got, "x")
	}
}

func TestEnsureSchemas_NoProviders(t *testing.T) {
	f := &fakeExecer{}
	if err := EnsureSchemas(context.Background(), f); err != nil {
		t.Fatalf("EnsureSchemas() with no providers: %v", err)
	}
	if len(f.calls) != 0 {
		t.Fatalf("expected zero ExecContext calls, got %d", len(f.calls))
	}
}

func TestEnsureSchemas_EmptySchemaSkips(t *testing.T) {
	f := &fakeExecer{}
	empty := SchemaProviderFunc(func() string { return "" })
	if err := EnsureSchemas(context.Background(), f, empty, empty); err != nil {
		t.Fatalf("EnsureSchemas() with empty providers: %v", err)
	}
	if len(f.calls) != 0 {
		t.Fatalf("empty schemas must skip; got %d ExecContext calls", len(f.calls))
	}
}

func TestEnsureSchemas_AppliesInOrder(t *testing.T) {
	f := &fakeExecer{}
	a := SchemaProviderFunc(func() string { return "CREATE TABLE a();" })
	b := SchemaProviderFunc(func() string { return "CREATE TABLE b();" })
	c := SchemaProviderFunc(func() string { return "CREATE TABLE c();" })
	if err := EnsureSchemas(context.Background(), f, a, b, c); err != nil {
		t.Fatalf("EnsureSchemas() error: %v", err)
	}
	if len(f.calls) != 3 {
		t.Fatalf("expected 3 ExecContext calls, got %d", len(f.calls))
	}
	want := []string{"CREATE TABLE a();", "CREATE TABLE b();", "CREATE TABLE c();"}
	for i, w := range want {
		if f.calls[i] != w {
			t.Fatalf("call[%d] = %q, want %q", i, f.calls[i], w)
		}
	}
}

func TestEnsureSchemas_MixedEmptyAndNonEmpty(t *testing.T) {
	f := &fakeExecer{}
	empty := SchemaProviderFunc(func() string { return "" })
	a := SchemaProviderFunc(func() string { return "CREATE TABLE a();" })
	if err := EnsureSchemas(context.Background(), f, empty, a, empty); err != nil {
		t.Fatalf("EnsureSchemas() error: %v", err)
	}
	if len(f.calls) != 1 || f.calls[0] != "CREATE TABLE a();" {
		t.Fatalf("expected one call to a; got %v", f.calls)
	}
}

func TestEnsureSchemas_Idempotent(t *testing.T) {
	f := &fakeExecer{}
	a := SchemaProviderFunc(func() string { return "CREATE TABLE IF NOT EXISTS a();" })
	ctx := context.Background()
	if err := EnsureSchemas(ctx, f, a); err != nil {
		t.Fatalf("first apply: %v", err)
	}
	if err := EnsureSchemas(ctx, f, a); err != nil {
		t.Fatalf("second apply: %v", err)
	}
	if len(f.calls) != 2 {
		t.Fatalf("expected 2 calls across two applies, got %d", len(f.calls))
	}
}

func TestEnsureSchemas_ErrorWrapsIndex(t *testing.T) {
	sentinel := errors.New("syntax error near 'xyz'")
	f := &fakeExecer{failOn: 2, failErr: sentinel}
	a := SchemaProviderFunc(func() string { return "CREATE TABLE a();" })
	b := SchemaProviderFunc(func() string { return "INVALID SQL;" })
	c := SchemaProviderFunc(func() string { return "CREATE TABLE c();" })
	err := EnsureSchemas(context.Background(), f, a, b, c)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, sentinel) {
		t.Fatalf("error must wrap sentinel via %%w; got: %v", err)
	}
	if !strings.Contains(err.Error(), "provider 2") {
		t.Fatalf("error must mention provider index (1-based); got: %v", err)
	}
	// First provider applied; third never reached.
	if len(f.calls) != 2 {
		t.Fatalf("expected 2 calls (a + failed b); got %d (%v)", len(f.calls), f.calls)
	}
}
