package main

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	facts "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts"
)

func TestLightScannerUsesCodeFactsFileMetrics(t *testing.T) {
	called := false
	source := reductionSourceFunc(func(_ context.Context, request *connect.Request[facts.DescribeCodeFactsRequest]) (*connect.Response[facts.CodeFactsReport], error) {
		called = true
		if len(request.Msg.GetInclude()) != 1 || request.Msg.GetInclude()[0] != facts.FactFamily_FACT_FAMILY_FILE_METRICS {
			t.Fatalf("requested families = %v", request.Msg.GetInclude())
		}
		return connect.NewResponse(&facts.CodeFactsReport{Facts: []*facts.GenericFact{{
			Family: facts.FactFamily_FACT_FAMILY_FILE_METRICS,
			Attributes: map[string]string{
				"lexical_metrics_version": "1", "file_path": "api/main.go", "line_count": "3", "file_extension": ".go", "language": "go",
				"code_lines": "2", "comment_lines": "1", "todo_count": "1", "fixme_count": "0", "hack_count": "0",
				"as_any_count": "0", "as_type_assertion_count": "0", "ts_ignore_count": "0", "non_null_assertion_count": "0",
				"fallback_import_count": "0", "fallback_function_count": "0",
			},
		}}}), nil
	})
	scanner := NewLightScanner(t.TempDir(), 0).withCodeFacts(source)
	got, err := scanner.collectFileMetricsContext(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	want := FileMetric{Path: "api/main.go", Lines: 3, Extension: ".go", Language: "go", CodeLines: 2, CommentLines: 1, TodoCount: 1}
	if !called || len(got) != 1 || got[0] != want {
		t.Fatalf("Code Facts was not used or metrics differ: called=%t got=%+v", called, got)
	}
}

func TestDetailedFileMetricsConsumesExistingInventory(t *testing.T) {
	root := t.TempDir()
	metrics, err := CollectDetailedFileMetricsFromInventory(root, []FileMetric{{Path: "api/unknown.zz", Lines: 7, Extension: ".zz"}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(metrics) != 1 || metrics[0].LineCount != 7 {
		t.Fatalf("metrics = %+v, want catalog line count preserved for unsupported language", metrics)
	}
}

func TestLightScannerHasNoLocalInventoryFallback(t *testing.T) {
	if _, err := NewLightScanner(t.TempDir(), 0).collectFileMetricsContext(context.Background()); err == nil {
		t.Fatal("file metric collection without Code Facts must fail closed")
	}
}

func TestDurationMillisecondsRoundsPositiveSubMillisecondScansUp(t *testing.T) {
	if got := durationMilliseconds(time.Microsecond); got != 1 {
		t.Fatalf("durationMilliseconds(1µs) = %d, want 1", got)
	}
	if got := durationMilliseconds(2 * time.Millisecond); got != 2 {
		t.Fatalf("durationMilliseconds(2ms) = %d, want 2", got)
	}
}
