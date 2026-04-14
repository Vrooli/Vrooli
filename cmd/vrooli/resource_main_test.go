package main

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func TestRunResourceStatusEnsuresResourceCLI(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writeResourceStatusFixture(t, root, "fixture-resource", "")

	t.Setenv("HOME", home)
	app := newTestApp(root)
	var ensured []string
	app.EnsureResourceCLIFn = func(rootArg, homeArg, name string) error {
		if rootArg != root || homeArg != home {
			t.Fatalf("ensure args = (%q, %q), want (%q, %q)", rootArg, homeArg, root, home)
		}
		ensured = append(ensured, name)
		return errors.New("ensure resource cli")
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.Run([]string{"resource", "status", "fixture-resource"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("run exit code = %d", code)
	}
	if got := strings.Join(ensured, "|"); got != "fixture-resource" {
		t.Fatalf("ensured = %q", got)
	}
	if !strings.Contains(stderr.String(), "ensure resource cli") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}
