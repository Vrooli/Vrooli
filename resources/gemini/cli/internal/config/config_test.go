package config

import (
	"errors"
	"path/filepath"
	"reflect"
	"testing"

	resourceenv "resource-gemini/cli/internal/env"
)

func TestContentStoreLifecycle(t *testing.T) {
	t.Parallel()

	store := NewContentStore(testRuntime(t))
	if err := store.EnsureInitialized(); err != nil {
		t.Fatalf("EnsureInitialized() error = %v", err)
	}
	if err := store.Add(ContentKindPrompt, "welcome", []byte("hello\n")); err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	got, err := store.Get(ContentKindPrompt, "welcome")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if string(got) != "hello\n" {
		t.Fatalf("Get() = %q", string(got))
	}

	list, err := store.List(ContentKindPrompt)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if !reflect.DeepEqual(list, []string{"prompt:welcome"}) {
		t.Fatalf("List() = %v", list)
	}

	if err := store.Remove(ContentKindPrompt, "welcome"); err != nil {
		t.Fatalf("Remove() error = %v", err)
	}
	if _, err := store.Get(ContentKindPrompt, "welcome"); !errors.Is(err, ErrContentNotFound) {
		t.Fatalf("Get() after remove error = %v", err)
	}
}

func TestGenerateContentEndpoint(t *testing.T) {
	t.Parallel()

	got := GenerateContentEndpoint("https://generativelanguage.googleapis.com/v1beta", "gemini-pro")
	want := "https://generativelanguage.googleapis.com/v1beta/models/gemini-pro:generateContent"
	if got != want {
		t.Fatalf("GenerateContentEndpoint() = %q, want %q", got, want)
	}
}

func testRuntime(t *testing.T) resourceenv.Runtime {
	t.Helper()

	root := t.TempDir()
	content := filepath.Join(root, "content")
	return resourceenv.Runtime{
		DataRoot:     root,
		ContentRoot:  content,
		PromptsDir:   filepath.Join(content, "prompts"),
		TemplatesDir: filepath.Join(content, "templates"),
		FunctionsDir: filepath.Join(content, "functions"),
		LogsDir:      filepath.Join(root, "logs"),
	}
}
