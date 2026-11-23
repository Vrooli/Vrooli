package models

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/vrooli/packages/capacity/modelretention"
)

func TestCallersReportsWhoUsesEachModelWithItsRoles(t *testing.T) {
	h := newRetentionHarness(t)
	at := time.Date(2026, 11, 21, 12, 0, 0, 0, time.UTC)
	rec := modelretention.NewRecorder("ollama", h.paths.UsageFile, func() time.Time { return at })
	for i := 0; i < 3; i++ {
		rec.Record("gemma4:12b")
	}
	rec.RecordCaller("gemma4:12b", "scenario:ai-gateway-start-api/ai-gateway-api", at)
	rec.RecordCaller("gemma4:12b", "scenario:ai-gateway-start-api/ai-gateway-api", at)
	rec.Record("nomic-embed-text")
	rec.RecordCaller("nomic-embed-text", "agent:codex/resource-ollama", at)
	if err := rec.Flush(context.Background()); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	handlers := &Handlers{Stdout: &out, Stderr: &out, Retention: h.deps}
	if err := handlers.Callers([]string{"--json", "--model", "gemma4:12b"}); err != nil {
		t.Fatal(err)
	}
	var report CallerReport
	if err := json.Unmarshal(out.Bytes(), &report); err != nil {
		t.Fatalf("callers --json is not a report: %v\n%s", err, out.String())
	}
	if len(report.Models) != 1 {
		t.Fatalf("models = %+v, want only the requested model", report.Models)
	}
	m := report.Models[0]
	if m.Model != "gemma4:12b" || m.Requests != 3 || m.Unattributed != 1 || strings.Join(m.Roles, ",") != "code.local" {
		t.Fatalf("model report = %+v, want 3 requests, 1 unattributed, role code.local", m)
	}
	if len(m.Callers) != 1 || m.Callers[0].Caller != "scenario:ai-gateway-start-api/ai-gateway-api" || m.Callers[0].Requests != 2 {
		t.Fatalf("callers = %+v", m.Callers)
	}

	out.Reset()
	if err := handlers.Callers(nil); err != nil {
		t.Fatal(err)
	}
	text := out.String()
	for _, want := range []string{"gemma4:12b  requests=3", "roles: code.local", "scenario:ai-gateway-start-api/ai-gateway-api", "(before caller tracking)", "nomic-embed-text:latest", "agent:codex/resource-ollama"} {
		if !strings.Contains(text, want) {
			t.Fatalf("callers output lacks %q:\n%s", want, text)
		}
	}
}
