package cliout

import (
	"bytes"
	"errors"
	"os"
	"strings"
	"testing"
)

func TestParseFormat(t *testing.T) {
	if got, err := ParseFormat("", false); err != nil || got != FormatHuman {
		t.Fatalf("ParseFormat human = %q, %v", got, err)
	}

	if got, err := ParseFormat("JSON", false); err != nil || got != FormatJSON {
		t.Fatalf("ParseFormat uppercase json = %q, %v", got, err)
	}

	if got, err := ParseFormat("json", false); err != nil || got != FormatJSON {
		t.Fatalf("ParseFormat explicit json = %q, %v", got, err)
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

func TestSuccessFieldsAddsSuccessKey(t *testing.T) {
	payload := SuccessFields(map[string]any{"report": "ok"})
	if payload[EnvelopeKeySuccess] != true {
		t.Fatalf("success field = %#v", payload[EnvelopeKeySuccess])
	}
	if payload["report"] != "ok" {
		t.Fatalf("report field = %#v", payload["report"])
	}
}

func TestWriteSuccessJSON(t *testing.T) {
	var buffer bytes.Buffer
	if err := WriteSuccessJSON(&buffer, "report", map[string]string{"status": "ok"}); err != nil {
		t.Fatalf("WriteSuccessJSON: %v", err)
	}

	output := buffer.String()
	if !strings.Contains(output, "\"success\": true") || !strings.Contains(output, "\"report\":") {
		t.Fatalf("expected success envelope, got %q", output)
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

func TestDefaultColorEnabledReturnsFalseForNilAndDumbTerm(t *testing.T) {
	if DefaultColorEnabled(nil) {
		t.Fatalf("expected nil stream to disable color")
	}

	t.Setenv("TERM", "dumb")
	if DefaultColorEnabled(os.Stdout) {
		t.Fatalf("expected TERM=dumb to disable color")
	}
}

func TestDefaultColorEnabledReturnsFalseForRegularFiles(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "cliout-*.txt")
	if err != nil {
		t.Fatalf("CreateTemp: %v", err)
	}
	t.Cleanup(func() {
		_ = file.Close()
	})

	if DefaultColorEnabled(file) {
		t.Fatalf("expected regular files to be treated as non-terminal streams")
	}
}

func TestDefaultColorEnabledReturnsTrueForCharacterDevices(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	t.Setenv("TERM", "xterm-256color")

	device, err := os.Open("/dev/null")
	if err != nil {
		t.Skipf("open /dev/null: %v", err)
	}
	t.Cleanup(func() {
		_ = device.Close()
	})

	if !DefaultColorEnabled(device) {
		t.Fatalf("expected character devices to be treated as color-capable streams")
	}
}

func TestDefaultColorEnabledReturnsFalseWhenStatFails(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "cliout-*.txt")
	if err != nil {
		t.Fatalf("CreateTemp: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	if DefaultColorEnabled(file) {
		t.Fatalf("expected failed Stat call to disable color")
	}
}

func TestColorEnabledForFileModeRequiresCharacterDevice(t *testing.T) {
	if !colorEnabledForFileMode(os.ModeCharDevice) {
		t.Fatalf("expected character devices to allow color by default")
	}
	if colorEnabledForFileMode(0) {
		t.Fatalf("expected regular files to disable color")
	}
}

func TestWriteJSONRequiresWriter(t *testing.T) {
	if err := WriteJSON(nil, map[string]string{"status": "ok"}); err == nil {
		t.Fatalf("expected nil writer to fail")
	}
}

func TestWriteJSONPropagatesWriterErrors(t *testing.T) {
	writer := &failingWriter{err: errors.New("boom")}
	if err := WriteJSON(writer, map[string]string{"status": "ok"}); err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("expected writer error, got %v", err)
	}
}

func TestRenderTableRequiresWriter(t *testing.T) {
	if err := RenderTable(nil, []string{"Name"}, [][]string{{"alpha"}}); err == nil {
		t.Fatalf("expected nil writer to fail")
	}
}

func TestRenderTableNoHeadersIsNoOp(t *testing.T) {
	var buffer bytes.Buffer
	if err := RenderTable(&buffer, nil, [][]string{{"alpha"}}); err != nil {
		t.Fatalf("RenderTable: %v", err)
	}
	if buffer.Len() != 0 {
		t.Fatalf("expected no output when headers are omitted, got %q", buffer.String())
	}
}

func TestRenderTablePadsShortRows(t *testing.T) {
	var buffer bytes.Buffer
	if err := RenderTable(&buffer, []string{"Name", "Status", "Ports"}, [][]string{{"alpha", "running"}}); err != nil {
		t.Fatalf("RenderTable: %v", err)
	}

	output := buffer.String()
	if !strings.Contains(output, "Name") || !strings.Contains(output, "running") {
		t.Fatalf("unexpected table output: %q", output)
	}
	if strings.Count(output, "\n") != 2 {
		t.Fatalf("expected header and one row, got %q", output)
	}
}

func TestRenderTableTruncatesExtraCells(t *testing.T) {
	var buffer bytes.Buffer
	if err := RenderTable(&buffer, []string{"Name", "Status"}, [][]string{{"alpha", "running", "ignored"}}); err != nil {
		t.Fatalf("RenderTable: %v", err)
	}

	output := buffer.String()
	if strings.Contains(output, "ignored") {
		t.Fatalf("expected extra cells to be ignored, got %q", output)
	}
}

func TestRenderTablePropagatesWriterErrors(t *testing.T) {
	writer := &failingWriter{err: errors.New("boom")}
	if err := RenderTable(writer, []string{"Name"}, [][]string{{"alpha"}}); err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("expected writer error, got %v", err)
	}
}

type failingWriter struct {
	err error
}

func (w *failingWriter) Write(p []byte) (int, error) {
	return 0, w.err
}
