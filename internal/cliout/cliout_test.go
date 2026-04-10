package cliout

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestParseFormat(t *testing.T) {
	if got, err := ParseFormat("", false); err != nil || got != FormatHuman {
		t.Fatalf("ParseFormat human = %q, %v", got, err)
	}

	if got, err := ParseFormat("", true); err != nil || got != FormatJSON {
		t.Fatalf("ParseFormat json flag = %q, %v", got, err)
	}

	if _, err := ParseFormat("xml", false); err == nil {
		t.Fatalf("expected invalid format error")
	}
}

func TestWriteJSON(t *testing.T) {
	var buffer bytes.Buffer
	if err := WriteJSON(&buffer, map[string]string{"status": "ok"}); err != nil {
		t.Fatalf("WriteJSON: %v", err)
	}

	output := buffer.String()
	if !strings.Contains(output, "\"status\": \"ok\"") {
		t.Fatalf("expected formatted json, got %q", output)
	}
}

func TestRenderTable(t *testing.T) {
	var buffer bytes.Buffer
	if err := RenderTable(&buffer, []string{"Name", "Status"}, [][]string{{"api", "healthy"}}); err != nil {
		t.Fatalf("RenderTable: %v", err)
	}

	output := buffer.String()
	if !strings.Contains(output, "Name") || !strings.Contains(output, "healthy") {
		t.Fatalf("expected headers and row in table output, got %q", output)
	}
}

func TestDefaultColorEnabledRespectsNoColor(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	if DefaultColorEnabled(os.Stdout) {
		t.Fatalf("expected NO_COLOR to disable color")
	}
}
