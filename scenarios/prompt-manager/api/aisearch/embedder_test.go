package aisearch

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

// embeddingResponse is a test-only mirror of the legacy on-the-wire shape,
// retained so the fake-ollama httptest servers in older tests still compile.
// New tests should use newEmbedderWithRunner directly; these dead servers can
// be removed in a follow-up cleanup.
type embeddingResponse struct {
	Embedding []float64 `json:"embedding"`
}

// fakeEmbedRunner captures invocation details and returns canned stdout/error.
type fakeEmbedRunner struct {
	stdout   []byte
	err      error
	gotArgs  []string
	gotStdin string
}

func (f *fakeEmbedRunner) run(_ context.Context, args []string, stdin []byte) ([]byte, error) {
	f.gotArgs = append([]string(nil), args...)
	f.gotStdin = string(stdin)
	return f.stdout, f.err
}

// fakeEmbedderOK returns an Embedder that always yields a fixed 3-dim vector.
func fakeEmbedderOK() Embedder {
	return newEmbedderWithRunner("nomic-embed-text", func(_ context.Context, _ []string, _ []byte) ([]byte, error) {
		return []byte(`{"embedding":[0.1,0.2,0.3]}`), nil
	})
}

// fakeEmbedderErr returns an Embedder whose every Embed call fails.
func fakeEmbedderErr() Embedder {
	return newEmbedderWithRunner("nomic-embed-text", func(_ context.Context, _ []string, _ []byte) ([]byte, error) {
		return nil, fmt.Errorf("HTTP 500: ollama down")
	})
}

func TestEmbedder_Embed_Success(t *testing.T) {
	want := []float64{0.1, 0.2, 0.3, 0.4}
	r := &fakeEmbedRunner{stdout: []byte(`{"embedding":[0.1,0.2,0.3,0.4]}` + "\n")}

	e := newEmbedderWithRunner("nomic-embed-text", r.run)
	got, err := e.Embed(context.Background(), "hello world")
	if err != nil {
		t.Fatalf("Embed: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Embed = %v, want %v", got, want)
	}
	wantArgs := []string{"resource-ollama", "gateway", "embed", "--model", "nomic-embed-text", "--json", "--input-stdin"}
	if !reflect.DeepEqual(r.gotArgs, wantArgs) {
		t.Fatalf("argv = %v, want %v", r.gotArgs, wantArgs)
	}
	if r.gotStdin != "hello world" {
		t.Fatalf("stdin = %q", r.gotStdin)
	}
}

func TestEmbedder_Embed_RunnerError(t *testing.T) {
	r := &fakeEmbedRunner{err: errors.New("exit status 1: HTTP 503")}
	e := newEmbedderWithRunner("m", r.run)
	_, err := e.Embed(context.Background(), "x")
	if err == nil || !strings.Contains(err.Error(), "503") {
		t.Fatalf("expected upstream error to surface, got %v", err)
	}
}

func TestEmbedder_Embed_DecodeError(t *testing.T) {
	r := &fakeEmbedRunner{stdout: []byte("not json")}
	e := newEmbedderWithRunner("m", r.run)
	_, err := e.Embed(context.Background(), "x")
	if err == nil || !strings.Contains(err.Error(), "decode") {
		t.Fatalf("expected decode error, got %v", err)
	}
}

func TestEmbedder_Embed_EmptyVector(t *testing.T) {
	r := &fakeEmbedRunner{stdout: []byte(`{"embedding":[]}`)}
	e := newEmbedderWithRunner("m", r.run)
	if _, err := e.Embed(context.Background(), "x"); err == nil {
		t.Fatal("expected error for empty vector")
	}
}

func TestEmbedder_DefaultModel(t *testing.T) {
	r := &fakeEmbedRunner{stdout: []byte(`{"embedding":[0.1]}`)}
	e := newEmbedderWithRunner("", r.run)
	if _, err := e.Embed(context.Background(), "x"); err != nil {
		t.Fatalf("Embed: %v", err)
	}
	for i, a := range r.gotArgs {
		if a == "--model" && i+1 < len(r.gotArgs) {
			if r.gotArgs[i+1] != "nomic-embed-text" {
				t.Fatalf("default model = %q", r.gotArgs[i+1])
			}
			return
		}
	}
	t.Fatal("--model flag not found in argv")
}

func TestEmbedder_Available_Success(t *testing.T) {
	e := fakeEmbedderOK()
	if !e.Available(context.Background()) {
		t.Error("Available = false, want true")
	}
}

func TestEmbedder_Available_RunnerError(t *testing.T) {
	e := fakeEmbedderErr()
	if e.Available(context.Background()) {
		t.Error("Available = true, want false")
	}
}
