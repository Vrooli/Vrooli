package discovery

import (
	"context"
	"errors"
	"testing"

	vroolicli "github.com/vrooli/vrooli-cli-go"
)

// stubRunner returns a fixed output/error for any vrooli invocation.
type stubRunner struct {
	out []byte
	err error
}

func (s stubRunner) Run(context.Context, string, ...string) ([]byte, error) {
	return s.out, s.err
}

func (s stubRunner) RunCombined(context.Context, string, ...string) ([]byte, error) {
	return s.out, s.err
}

func TestEnumerateMapsAndFiltersTypedResources(t *testing.T) {
	out := []byte(`{"resources":[
		{"name":"claude-code","enabled":true,"driver":"external-cli","manifest_path":"/r/claude-code/resource.json"},
		{"name":"postgres","enabled":true,"driver":"compose-service","manifest_path":"/r/postgres/resource.json"}
	]}`)
	enum := &CLIResourceEnumerator{client: vroolicli.New(vroolicli.WithRunner(stubRunner{out: out}))}

	refs, err := enum.Enumerate(context.Background())
	if err != nil {
		t.Fatalf("Enumerate: %v", err)
	}
	if len(refs) != 1 || refs[0].Name != "claude-code" {
		t.Fatalf("expected only claude-code, got %+v", refs)
	}
}

// TestEnumeratePropagatesCLIError is the point of the migration: a CLI failure
// surfaces as an error instead of silently yielding zero backup targets.
func TestEnumeratePropagatesCLIError(t *testing.T) {
	enum := &CLIResourceEnumerator{client: vroolicli.New(vroolicli.WithRunner(stubRunner{err: errors.New("boom")}))}

	if _, err := enum.Enumerate(context.Background()); err == nil {
		t.Fatal("expected error to propagate, got nil")
	}
}
