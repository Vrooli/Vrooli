package measures

import (
	"context"
	"testing"

	measurelib "github.com/vrooli/measures-go"
)

func newTestRegistry(t *testing.T, handler *Handler) *measurelib.Registry {
	t.Helper()
	registry, err := handler.Registry()
	if err != nil {
		t.Fatalf("build measure registry: %v", err)
	}
	return registry
}

func executeRegisteredMeasure(t *testing.T, registry *measurelib.Registry, request measurelib.MeasureRequest) measurelib.MeasureResult {
	t.Helper()
	result, err := registry.Execute(context.Background(), request)
	if err != nil {
		t.Fatalf("execute measure %q: %v", request.Measure, err)
	}
	return result
}
