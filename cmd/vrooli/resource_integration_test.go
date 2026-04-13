package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestSplitRunResourceStatusUsesNativeController(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writeResourceStatusFixture(t, root, "fixture-resource", `{"installed":true,"running":true,"healthy":true,"message":"healthy"}`)

	t.Setenv("HOME", home)
	app := newTestApp(root)
	app.execCommand = func(spec commandSpec) error {
		t.Fatalf("resource status should not route through CLI bash shim: %+v", spec)
		return nil
	}

	var stdout bytes.Buffer
	code := app.Run([]string{"resource", "status", "fixture-resource"}, &stdout, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("run exit code = %d", code)
	}
	output := stdout.String()
	if !strings.Contains(output, "fixture-resource") || !strings.Contains(output, "healthy") {
		t.Fatalf("stdout = %q", output)
	}
}
