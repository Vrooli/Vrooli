package ports

import "testing"

func TestVectorFilterZeroValue(t *testing.T) {
	var filter VectorFilter
	if filter.Namespaces != nil || filter.Visibility != nil || filter.Tags != nil {
		t.Fatalf("expected zero-value slices to be nil")
	}
}
