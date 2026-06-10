package findings

import (
	"context"
	"errors"
	"testing"
)

// fakeMutService implements Service with controllable outcomes for the
// mutation paths the notify decorator wraps.
type fakeMutService struct {
	Service
	fail bool
}

var errBoom = errors.New("boom")

func (f *fakeMutService) Add(context.Context, NewFinding) (Finding, error) {
	if f.fail {
		return Finding{}, errBoom
	}
	return Finding{ID: "f-1"}, nil
}

func (f *fakeMutService) Edit(context.Context, string, EditInput) (Finding, error) {
	if f.fail {
		return Finding{}, errBoom
	}
	return Finding{ID: "f-1"}, nil
}

func (f *fakeMutService) Supersede(context.Context, string, string, string) (Finding, error) {
	if f.fail {
		return Finding{}, errBoom
	}
	return Finding{ID: "f-1"}, nil
}

func (f *fakeMutService) Flag(context.Context, string, string) (Finding, error) {
	if f.fail {
		return Finding{}, errBoom
	}
	return Finding{ID: "f-1"}, nil
}

func (f *fakeMutService) ResolveDispute(context.Context, string, string, string, string) (Finding, error) {
	if f.fail {
		return Finding{}, errBoom
	}
	return Finding{ID: "f-1"}, nil
}

func (f *fakeMutService) Prune(_ context.Context, dryRun bool) ([]string, error) {
	if f.fail {
		return nil, errBoom
	}
	if dryRun {
		return []string{"f-1"}, nil
	}
	return []string{"f-1"}, nil
}

func (f *fakeMutService) List(context.Context, ListFilter) ([]Finding, error) {
	return nil, nil
}

// TestMutationNotifyFiresOncePerSuccessfulWrite: every successful content
// mutation must fire exactly one notify (the index-freshness kick); reads and
// failed mutations must not.
func TestMutationNotifyFiresOncePerSuccessfulWrite(t *testing.T) {
	ctx := context.Background()

	kicks := 0
	svc := WithMutationNotify(&fakeMutService{}, func() { kicks++ })

	mutations := []func() error{
		func() error { _, err := svc.Add(ctx, NewFinding{}); return err },
		func() error { _, err := svc.Edit(ctx, "f-1", EditInput{}); return err },
		func() error { _, err := svc.Supersede(ctx, "f-1", "f-2", "r"); return err },
		func() error { _, err := svc.Flag(ctx, "f-1", "r"); return err },
		func() error { _, err := svc.ResolveDispute(ctx, "f-1", "keep", "", "r"); return err },
		func() error { _, err := svc.Prune(ctx, false); return err },
	}
	for i, m := range mutations {
		before := kicks
		if err := m(); err != nil {
			t.Fatalf("mutation %d: %v", i, err)
		}
		if kicks != before+1 {
			t.Fatalf("mutation %d: kicks = %d, want %d (exactly one per successful write)", i, kicks, before+1)
		}
	}

	// Reads must not kick.
	if _, err := svc.List(ctx, ListFilter{}); err != nil {
		t.Fatalf("List: %v", err)
	}
	// Dry-run prune mutates nothing — no kick.
	if _, err := svc.Prune(ctx, true); err != nil {
		t.Fatalf("Prune(dry): %v", err)
	}
	if kicks != len(mutations) {
		t.Fatalf("kicks after reads/dry-run = %d, want %d (no kick on non-mutations)", kicks, len(mutations))
	}
}

// TestMutationNotifySkipsFailedWrites: a failed mutation leaves the index
// unchanged, so it must not kick.
func TestMutationNotifySkipsFailedWrites(t *testing.T) {
	ctx := context.Background()
	kicks := 0
	svc := WithMutationNotify(&fakeMutService{fail: true}, func() { kicks++ })

	_, _ = svc.Add(ctx, NewFinding{})
	_, _ = svc.Supersede(ctx, "f-1", "f-2", "r")
	_, _ = svc.Prune(ctx, false)
	if kicks != 0 {
		t.Fatalf("kicks after failed writes = %d, want 0", kicks)
	}
}

// TestMutationNotifyNilPassthrough: a nil notify must return the service
// unchanged (no wrapper allocation, no behavior change).
func TestMutationNotifyNilPassthrough(t *testing.T) {
	base := &fakeMutService{}
	if got := WithMutationNotify(base, nil); got != Service(base) {
		t.Fatalf("WithMutationNotify(svc, nil) must return svc unchanged")
	}
}
