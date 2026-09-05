package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestReadEventNormalizesNativePayload(t *testing.T) {
	event, err := readEvent(strings.NewReader(`{"tool_name":"shell","command":"pnpm add example","cwd":"/workspace"}`), "codex")
	if err != nil {
		t.Fatal(err)
	}
	if event.Runner != "codex" || event.Tool != "shell" || event.Shell != "pnpm add example" {
		t.Fatalf("normalized event = %+v", event)
	}
}

func TestStatusDoesNotRequireProviderProcess(t *testing.T) {
	var output bytes.Buffer
	err := run([]string{"status", "--snapshot-dir", t.TempDir()}, strings.NewReader(""), &output, &output)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), `"status": "unavailable"`) {
		t.Fatalf("status output = %s", output.String())
	}
}
