package cliapp

import (
	"bytes"
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
}

func TestPrintReportJSON(t *testing.T) {
	var out bytes.Buffer
	err := PrintReportJSON(&out, MutationReport{
		Result: []string{"Task created"},
	})
	if err != nil {
		t.Fatalf("PrintReportJSON: %v", err)
	}
	if !strings.Contains(out.String(), "\"result\": [") {
		t.Fatalf("expected JSON output, got %q", out.String())
	}
}
