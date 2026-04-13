//go:build integration
// +build integration

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
