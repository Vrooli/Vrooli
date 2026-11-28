package backends

import (
	"context"
	"errors"
	"strings"
	"testing"

	"image-tools/internal/storage"
)

func TestBuiltinProviderKeepsOperationsAndExecutionContract(t *testing.T) {
	ops := []string{"one"}
	provider := NewBuiltinProvider("local", "analysis", ops, func(_ context.Context, req Request) error {
		if req.Operation != "one" {
			t.Fatalf("operation = %q", req.Operation)
		}
		return nil
	})
	ops[0] = "changed"
	if got := provider.Operations()[0]; got != "one" {
		t.Fatalf("constructor retained caller-owned operations: %q", got)
	}
	if got := provider.Operations(); len(got) != 1 || got[0] != "one" {
		t.Fatalf("operations = %v", got)
	}
	got, err := provider.Execute(context.Background(), Request{
		Operation: "one",
		Output:    storage.OutputTarget{LocalPath: "/tmp/result"},
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if got.OutputRef != "/tmp/result" || got.Meta["backend"] != "local" {
		t.Fatalf("result = %+v", got)
	}
}

func TestBuiltinProviderPreservesScopedErrors(t *testing.T) {
	provider := NewBuiltinProvider("local", "analysis", []string{"one"}, func(context.Context, Request) error {
		return errors.New("failed")
	})
	if _, err := provider.Execute(context.Background(), Request{}); err == nil || !strings.Contains(err.Error(), "analysis: builtin backend \"local\" requires a local output path") {
		t.Fatalf("missing output error = %v", err)
	}
	_, err := provider.Execute(context.Background(), Request{Output: storage.OutputTarget{LocalPath: "/tmp/result"}})
	if err == nil || !strings.Contains(err.Error(), "analysis: builtin backend \"local\" execution failed: failed") {
		t.Fatalf("execution error = %v", err)
	}
}
