package main

import (
	"testing"
)

func TestGenerateIssues_TestFileDupUsesHigherThreshold(t *testing.T) {
	dup := 20.0
	metrics := []DetailedFileMetrics{
		{FilePath: "api/handler_test.go", DuplicationPct: &dup},
	}

	config := DefaultIssueGeneratorConfig()
	// Default: 10% for normal, 30% for test
	issues := GenerateIssuesFromMetrics("test-scenario", metrics, config)

	for _, i := range issues {
		if i.Category == "duplication" {
			t.Errorf("test file at 20%% dup should NOT generate duplication issue (test threshold 30%%)")
		}
	}
}

func TestGenerateIssues_NonTestFileDupUsesStandardThreshold(t *testing.T) {
	dup := 20.0
	metrics := []DetailedFileMetrics{
		{FilePath: "api/handler.go", DuplicationPct: &dup},
	}

	config := DefaultIssueGeneratorConfig()
	issues := GenerateIssuesFromMetrics("test-scenario", metrics, config)

	found := false
	for _, i := range issues {
		if i.Category == "duplication" {
			found = true
			if i.Severity != "low" {
				t.Errorf("expected severity 'low' for 20%% dup (threshold 10%%, needs >20%% for medium), got %q", i.Severity)
			}
		}
	}
	if !found {
		t.Error("non-test file at 20%% dup should generate a duplication issue (threshold 10%%)")
	}
}

func TestGenerateIssues_TSTestFileThreshold(t *testing.T) {
	dup := 25.0
	testFiles := []string{
		"ui/src/App.test.ts",
		"ui/src/utils.test.tsx",
		"ui/src/hooks.spec.ts",
		"ui/src/component.spec.tsx",
	}

	config := DefaultIssueGeneratorConfig()

	for _, path := range testFiles {
		metrics := []DetailedFileMetrics{
			{FilePath: path, DuplicationPct: &dup},
		}
		issues := GenerateIssuesFromMetrics("test-scenario", metrics, config)
		for _, i := range issues {
			if i.Category == "duplication" {
				t.Errorf("%s at 25%% dup should NOT generate duplication issue (test threshold 30%%)", path)
			}
		}
	}
}

func TestGenerateIssues_TestFileDupExceedsTestThreshold(t *testing.T) {
	dup := 35.0
	metrics := []DetailedFileMetrics{
		{FilePath: "api/handler_test.go", DuplicationPct: &dup},
	}

	config := DefaultIssueGeneratorConfig()
	issues := GenerateIssuesFromMetrics("test-scenario", metrics, config)

	found := false
	for _, i := range issues {
		if i.Category == "duplication" {
			found = true
		}
	}
	if !found {
		t.Error("test file at 35%% dup should generate a duplication issue (test threshold 30%%)")
	}
}

func TestGenerateIssues_ZeroTestThresholdFallsBackToStandard(t *testing.T) {
	dup := 15.0
	metrics := []DetailedFileMetrics{
		{FilePath: "api/handler_test.go", DuplicationPct: &dup},
	}

	config := DefaultIssueGeneratorConfig()
	config.HighDuplicationPctTest = 0 // disabled

	issues := GenerateIssuesFromMetrics("test-scenario", metrics, config)
	found := false
	for _, i := range issues {
		if i.Category == "duplication" {
			found = true
		}
	}
	if !found {
		t.Error("with test threshold=0, test file at 15%% dup should use standard threshold (10%%)")
	}
}
