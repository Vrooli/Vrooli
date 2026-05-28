package main

import (
	"context"
	"fmt"

	"swarm-manager/internal/backlog"
	"swarm-manager/internal/records"
)

// recordStubAdapter satisfies backlog.RecordStubCreator by delegating to the
// records.Service. Lives in main so neither package needs to import the other.
type recordStubAdapter struct {
	svc *records.Service
}

func newRecordStubAdapter(svc *records.Service) *recordStubAdapter {
	if svc == nil {
		return nil
	}
	return &recordStubAdapter{svc: svc}
}

// CreateBacklogStub builds a CreateStubInput from the terminal-status signal
// and forwards to records.Service.CreateStub. Scenario is left empty for now
// because the backlog item shape doesn't expose target_scenario in the hook
// signature; a future iteration can read the item back from the store, but
// keeping the hook synchronous and cheap is more important than a perfect
// scenario tag — the agent fills it in via `records edit` when narrative is
// filled.
func (a *recordStubAdapter) CreateBacklogStub(ctx context.Context, kind, name string, status backlog.BacklogStatus, decidedBy string) (string, error) {
	if a == nil || a.svc == nil {
		return "", nil
	}
	outcome := records.OutcomeShipped
	switch status {
	case backlog.StatusFailed:
		outcome = records.OutcomeAbandoned
	case backlog.StatusNeedsFollowup:
		// Followup items aren't yet done; don't write a stub.
		return "", nil
	}
	in := records.CreateStubInput{
		Kind:       records.RecordKind(kind),
		Scenario:   "swarm-manager", // see comment above; revised on narrative fill
		BacklogRef: fmt.Sprintf("%s/%s", kind, name),
		Outcome:    outcome,
		CreatedBy:  decidedBy,
	}
	r, err := a.svc.CreateStub(ctx, in)
	if err != nil {
		return "", err
	}
	return r.ID, nil
}
