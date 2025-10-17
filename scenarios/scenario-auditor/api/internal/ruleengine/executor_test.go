package ruleengine

import (
	"reflect"
	"testing"
	"time"
)

func TestConvertStructViolationPreservesExtendedFields(t *testing.T) {
	discovered := time.Now().UTC().Truncate(time.Second)

	sample := struct {
		ID             string
		RuleID         string
		Type           string
		Severity       string
		Description    string
		FilePath       string
		LineNumber     int
		Recommendation string
		Standard       string
		DiscoveredAt   time.Time
	}{
		ID:             "violation-123",
		RuleID:         "rule-from-sample",
		Type:           "config",
		Severity:       "high",
		Description:    "Detailed context for the failure",
		FilePath:       "test/path/file.go",
		LineNumber:     42,
		Recommendation: "Do the right thing",
		Standard:       "configuration-v1",
		DiscoveredAt:   discovered,
	}

	violation, err := convertStructViolation(reflect.ValueOf(sample), "fallback-rule", "config")
	if err != nil {
		t.Fatalf("convertStructViolation returned error: %v", err)
	}
	if violation == nil {
		t.Fatalf("expected violation to be non-nil")
	}

	if violation.RuleID != "rule-from-sample" {
		t.Fatalf("expected RuleID=rule-from-sample, got %q", violation.RuleID)
	}
	if violation.ID != "violation-123" {
		t.Fatalf("expected ID=violation-123, got %q", violation.ID)
	}
	if violation.Type != "config" {
		t.Fatalf("expected Type=config, got %q", violation.Type)
	}
	if violation.Description != sample.Description {
		t.Fatalf("expected Description to match, got %q", violation.Description)
	}
	if violation.Message != sample.Description {
		t.Fatalf("expected Message fallback to description, got %q", violation.Message)
	}
	if violation.Line != 42 || violation.LineNumber != 42 {
		t.Fatalf("expected both Line and LineNumber to be 42, got %d/%d", violation.Line, violation.LineNumber)
	}
	if violation.DiscoveredAt != discovered {
		t.Fatalf("expected DiscoveredAt to be preserved, got %v", violation.DiscoveredAt)
	}
}

func TestConvertStructViolationLineFallback(t *testing.T) {
	sample := struct {
		Line int
	}{
		Line: 7,
	}

	violation, err := convertStructViolation(reflect.ValueOf(sample), "rule", "category")
	if err != nil {
		t.Fatalf("convertStructViolation returned error: %v", err)
	}
	if violation == nil {
		t.Fatalf("expected violation to be non-nil")
	}
	if violation.Line != 7 {
		t.Fatalf("expected Line=7, got %d", violation.Line)
	}
	if violation.LineNumber != 7 {
		t.Fatalf("expected LineNumber=7, got %d", violation.LineNumber)
	}
}
