package credentials

import (
	"context"
	"errors"
	"sync"
	"testing"

	"scenario-to-cloud/domain"
)

func TestMemoryStoreAtomicallyClaimsOneOpenOperationPerBinding(t *testing.T) {
	store := NewMemoryStore()
	start := make(chan struct{})
	results := make(chan error, 2)
	var ready sync.WaitGroup
	var done sync.WaitGroup
	for _, id := range []string{"operation-a", "operation-b"} {
		rotation := &domain.CredentialRotation{ID: id, DeploymentID: "deployment", BindingID: "binding", State: domain.RotationPlanned}
		ready.Add(1)
		done.Add(1)
		go func() {
			defer done.Done()
			ready.Done()
			<-start
			results <- store.CreateRotation(context.Background(), rotation)
		}()
	}
	ready.Wait()
	close(start)
	done.Wait()
	close(results)

	created, rejected := 0, 0
	for err := range results {
		switch {
		case err == nil:
			created++
		case errors.Is(err, domain.ErrCredentialOperationConflict):
			rejected++
		default:
			t.Fatalf("CreateRotation error = %v", err)
		}
	}
	if created != 1 || rejected != 1 {
		t.Fatalf("created=%d rejected=%d, want one of each", created, rejected)
	}

	var active domain.CredentialRotation
	for _, id := range []string{"operation-a", "operation-b"} {
		stored, _ := store.GetRotation(context.Background(), id)
		if stored != nil {
			active = *stored
		}
	}
	if err := active.TransitionTo(domain.RotationFailed); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveRotation(context.Background(), &active); err != nil {
		t.Fatal(err)
	}
	if err := store.CreateRotation(context.Background(), &domain.CredentialRotation{ID: "operation-c", DeploymentID: "deployment", BindingID: "binding", State: domain.RotationPlanned}); err != nil {
		t.Fatalf("create after terminal state: %v", err)
	}
}
