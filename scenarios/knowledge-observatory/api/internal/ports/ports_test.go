package ports

import "testing"

func TestVectorFilterZeroValue(t *testing.T) {
	var filter VectorFilter
	if filter.Namespaces != nil || filter.Visibility != nil || filter.Tags != nil {
		t.Fatalf("expected zero-value slices to be nil")
	}
}

func TestJobStatusZeroValue(t *testing.T) {
	var status JobStatus
	if status.JobID != "" || status.Status != "" {
		t.Fatalf("expected zero-value status fields")
	}
}
