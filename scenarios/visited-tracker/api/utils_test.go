package main

import (
	"testing"
)

func TestStrPtr(t *testing.T) {
	test := "test string"
	ptr := strPtr(test)

	if ptr == nil {
		t.Error("strPtr should return non-nil pointer")
	}

	if *ptr != test {
		t.Errorf("Expected %s, got %s", test, *ptr)
	}
}

// Test updateStalenessScores function
