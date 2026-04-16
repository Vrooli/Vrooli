package cliapp

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestRenderOperationalReport(t *testing.T) {
	var out bytes.Buffer
	err := RenderOperationalReport(&out, OperationalReport{
		Status: []string{
			"Healthy",
			"2 dependencies connected",
		},
		Triage: []TriageGroup{
			{
				Heading: "Auto-fix now",
				Items: []string{
					"refresh cached prompts",
				},
			},
			{
				Heading: "Manual review",
				Items: []string{
					"confirm API token rotation",
				},
			},
		},
		NextSteps: []string{
			"demo status --json",
			"demo configure token <token>",
		},
	})
	if err != nil {
		t.Fatalf("RenderOperationalReport: %v", err)
	}

	got := out.String()
	for _, needle := range []string{
		"Status:\n  Healthy\n  2 dependencies connected\n",
		"Triage:\n  Auto-fix now:\n    refresh cached prompts\n  Manual review:\n    confirm API token rotation\n",
		"Next Steps:\n  demo status --json\n  demo configure token <token>\n",
	} {
		if !strings.Contains(got, needle) {
			t.Fatalf("output missing %q in %q", needle, got)
		}
	}
	assertSectionOrder(t, got, "Status:\n", "Triage:\n", "Next Steps:\n")
}

func TestRenderOperationalReportMultilineItems(t *testing.T) {
	var out bytes.Buffer
	err := RenderOperationalReport(&out, OperationalReport{
		Status: []string{
			"Healthy\nVersion: 1.2.3",
		},
		Triage: []TriageGroup{
			{
				Heading: "Manual review",
				Items: []string{
					"Primary issue\nFollow-up detail",
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("RenderOperationalReport: %v", err)
	}

	got := out.String()
	for _, needle := range []string{
		"Status:\n  Healthy\n  Version: 1.2.3\n",
		"Triage:\n  Manual review:\n    Primary issue\n    Follow-up detail\n",
	} {
		if !strings.Contains(got, needle) {
			t.Fatalf("output missing %q in %q", needle, got)
		}
	}
}

func TestRenderOperationalReportEmptyState(t *testing.T) {
	var out bytes.Buffer
	if err := RenderOperationalReport(&out, OperationalReport{}); err != nil {
		t.Fatalf("RenderOperationalReport: %v", err)
	}

	want := "Status:\n  (no status reported)\nNext Steps:\n  (none)\n"
	if out.String() != want {
		t.Fatalf("output = %q, want %q", out.String(), want)
	}
}

func TestRenderListReport(t *testing.T) {
	var out bytes.Buffer
	err := RenderListReport(&out, ListReport{
		Summary: []string{
			"2 tasks found",
		},
		Results: []string{
			"task-1  pending",
			"task-2  done",
		},
		RetrievalHints: []string{
			"demo tasks get task-1",
		},
	})
	if err != nil {
		t.Fatalf("RenderListReport: %v", err)
	}

	got := out.String()
	for _, needle := range []string{
		"Summary:\n  2 tasks found\n",
		"Results:\n  task-1  pending\n  task-2  done\n",
		"Retrieval Hints:\n  demo tasks get task-1\n",
	} {
		if !strings.Contains(got, needle) {
			t.Fatalf("output missing %q in %q", needle, got)
		}
	}
	assertSectionOrder(t, got, "Summary:\n", "Results:\n", "Retrieval Hints:\n")
}

func TestRenderListReportEmptyState(t *testing.T) {
	var out bytes.Buffer
	if err := RenderListReport(&out, ListReport{}); err != nil {
		t.Fatalf("RenderListReport: %v", err)
	}

	want := "Summary:\n  (no summary available)\nResults:\n  (none)\nRetrieval Hints:\n  (none)\n"
	if out.String() != want {
		t.Fatalf("output = %q, want %q", out.String(), want)
	}
}

func TestRenderMutationReport(t *testing.T) {
	var out bytes.Buffer
	err := RenderMutationReport(&out, MutationReport{
		Result: []string{
			"Task created",
		},
		Changes: []string{
			"id=task-1",
			"status=pending",
		},
		NextCommand: []string{
			"demo tasks get task-1",
		},
	})
	if err != nil {
		t.Fatalf("RenderMutationReport: %v", err)
	}

	got := out.String()
	for _, needle := range []string{
		"Result:\n  Task created\n",
		"What Changed:\n  id=task-1\n  status=pending\n",
		"Next Command:\n  demo tasks get task-1\n",
	} {
		if !strings.Contains(got, needle) {
			t.Fatalf("output missing %q in %q", needle, got)
		}
	}
	assertSectionOrder(t, got, "Result:\n", "What Changed:\n", "Next Command:\n")
}

func TestRenderMutationReportEmptyState(t *testing.T) {
	var out bytes.Buffer
	if err := RenderMutationReport(&out, MutationReport{}); err != nil {
		t.Fatalf("RenderMutationReport: %v", err)
	}

	want := "Result:\n  (no result reported)\nWhat Changed:\n  (none)\nNext Command:\n  (none)\n"
	if out.String() != want {
		t.Fatalf("output = %q, want %q", out.String(), want)
	}
}

func TestPrintReportJSON(t *testing.T) {
	var out bytes.Buffer
	report := MutationReport{
		Result:      []string{"Task created"},
		Changes:     []string{"id=task-1"},
		NextCommand: []string{"demo tasks get task-1"},
	}
	if err := PrintReportJSON(&out, report); err != nil {
		t.Fatalf("PrintReportJSON: %v", err)
	}

	var decoded MutationReport
	if err := json.Unmarshal(out.Bytes(), &decoded); err != nil {
		t.Fatalf("json unmarshal: %v", err)
	}
	if strings.Join(decoded.Result, "\n") != strings.Join(report.Result, "\n") {
		t.Fatalf("decoded result = %#v, want %#v", decoded.Result, report.Result)
	}
	if strings.Join(decoded.Changes, "\n") != strings.Join(report.Changes, "\n") {
		t.Fatalf("decoded changes = %#v, want %#v", decoded.Changes, report.Changes)
	}
	if strings.Join(decoded.NextCommand, "\n") != strings.Join(report.NextCommand, "\n") {
		t.Fatalf("decoded next command = %#v, want %#v", decoded.NextCommand, report.NextCommand)
	}
}

func assertSectionOrder(t *testing.T, output string, sections ...string) {
	t.Helper()

	last := -1
	for _, section := range sections {
		index := strings.Index(output, section)
		if index < 0 {
			t.Fatalf("section %q missing from %q", section, output)
		}
		if index <= last {
			t.Fatalf("section %q out of order in %q", section, output)
		}
		last = index
	}
}
