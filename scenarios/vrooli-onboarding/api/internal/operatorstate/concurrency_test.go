package operatorstate_test

import (
	"context"
	"sync"
	"testing"
)

// [REQ:ONB-STATE-CONCURRENT-SAFE]
func TestConcurrentPatchesEvidencePreservesDisjointOperatorChoices(t *testing.T) {
	service := newService(t)
	ctx := context.Background()
	patches := []string{`{"scenarios":{"alpha":{"enabled":true}}}`, `{"resources":{"ollama":{"enabled":true}}}`}
	var group sync.WaitGroup
	errs := make(chan error, len(patches))
	for _, patch := range patches {
		group.Add(1)
		go func(patch string) {
			defer group.Done()
			_, err := service.Apply(ctx, []byte(patch))
			errs <- err
		}(patch)
	}
	group.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	current, err := service.Load(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if current.Scenarios["alpha"].Enabled == nil || !*current.Scenarios["alpha"].Enabled || current.Resources["ollama"].Enabled == nil || !*current.Resources["ollama"].Enabled {
		t.Fatalf("concurrent patches lost a choice: %#v", current)
	}
}
