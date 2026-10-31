package support

import (
	"flag"
	"testing"
)

func TestRequireArg(t *testing.T) {
	t.Run("returns error when no args", func(t *testing.T) {
		fs := flag.NewFlagSet("test", flag.ContinueOnError)
		err := RequireArg(fs, "test <id>")
		if err == nil || err.Error() != "usage: test <id>" {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("succeeds with args", func(t *testing.T) {
		fs := flag.NewFlagSet("test", flag.ContinueOnError)
		if err := fs.Parse([]string{"my-id"}); err != nil {
			t.Fatalf("Parse: %v", err)
		}
		if err := RequireArg(fs, "test <id>"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestParseFlags(t *testing.T) {
	fs, jsonOut, err := ParseFlags("test", []string{"--json", "arg1"})
	if err != nil {
		t.Fatalf("ParseFlags: %v", err)
	}
	if !*jsonOut {
		t.Fatal("expected json flag to be enabled")
	}
	if fs.NArg() != 1 || fs.Arg(0) != "arg1" {
		t.Fatalf("unexpected args: %v", fs.Args())
	}
}

func TestUnmarshal(t *testing.T) {
	var payload struct {
		Name string `json:"name"`
	}
	if err := Unmarshal([]byte(`{"name":"test"}`), &payload); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if payload.Name != "test" {
		t.Fatalf("unexpected name %q", payload.Name)
	}
}
