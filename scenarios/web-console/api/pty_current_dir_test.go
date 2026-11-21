package main

import (
	"context"
	"errors"
	"testing"

	platform "github.com/vrooli/platform-go"
	"web-console/internal/ptyfake"
)

func TestCurrentDir_ReturnsErrorWhenUnsupported(t *testing.T) {
	fake := &ptyfake.FakePTY{}
	fake.CurrentDirErr = platform.ErrUnsupported
	got, err := fake.CurrentDir(context.Background())
	if got != "" {
		t.Fatalf("working directory = %q, want empty on unsupported host", got)
	}
	if !errors.Is(err, platform.ErrUnsupported) {
		t.Fatalf("error = %v, want platform.ErrUnsupported", err)
	}
}

func TestCurrentDir_ReturnsConfiguredDirBeforeStart(t *testing.T) {
	got, err := (&realPTY{}).CurrentDir(context.Background())
	if err != nil {
		t.Fatalf("pre-start CurrentDir error: %v", err)
	}
	if got == "" {
		t.Fatal("pre-start CurrentDir returned an empty directory")
	}
}
