package main

import (
	"context"
	"fmt"
	"strings"

	"swarm-manager/internal/backlog"
	"swarm-manager/internal/pathutil"
	"swarm-manager/internal/records"
)

// recordCaptureAdapter satisfies backlog.RecordCreator by delegating to the
// records.Service. Lives in main so neither package needs to import the other.
type recordCaptureAdapter struct {
	svc *records.Service
}

func newRecordCaptureAdapter(svc *records.Service) *recordCaptureAdapter {
	if svc == nil {
		return nil
	}
	return &recordCaptureAdapter{svc: svc}
}

// CreateBacklogRecord writes a FILLED, immediately-indexed record from a backlog
// item's terminal decision — the write-side of the recursive-learning loop. The
// item's own human-authored title/description become the record's
// trigger/approach (so a hit carries the lesson, not just an id), the acceptance
// globs derive the target scenario, the milestone links it back, and the
// decision maps to an outcome.
//
// Unlike the empty stub this hook used to create (which `records edit` was meant
// to fill, but nothing ever did), the record is born non-stub and indexed at
// birth via records.Service.Create. That also means it is immutable: enrichment
// goes through `records supersede`, not `records edit` (which would hit
// ErrStubLocked).
func (a *recordCaptureAdapter) CreateBacklogRecord(ctx context.Context, req backlog.BacklogRecordRequest) (string, error) {
	if a == nil || a.svc == nil {
		return "", nil
	}
	outcome := records.OutcomeShipped
	switch req.Status {
	case backlog.StatusFailed:
		outcome = records.OutcomeAbandoned
	case backlog.StatusNeedsFollowup:
		// Followup items aren't done yet — nothing to capture.
		return "", nil
	}

	// Derive the target scenario from the item's acceptance globs (work usually
	// lives under scenarios/<name>/...); fall back to swarm-manager when the
	// globs name no scenario (e.g. root tooling).
	scenario := "swarm-manager"
	if names := pathutil.ScenariosFromGlobs(req.AcceptanceAllow); len(names) > 0 {
		scenario = names[0]
	}

	trigger := strings.TrimSpace(req.Title)
	approach := strings.TrimSpace(req.Description)
	// records.Service.Create requires at least one of trigger/approach/ruled_out.
	// A bare item (no title AND no description) still earns a searchable floor
	// record via a templated approach the agent is expected to enrich.
	if trigger == "" && approach == "" {
		approach = fmt.Sprintf(
			"Auto-captured on review-decide (%s) for backlog %s/%s. Enrich via `swarm-manager records supersede`.",
			outcome, req.Kind, req.Name)
	}

	r, err := a.svc.Create(ctx, records.CreateInput{
		Kind:         records.RecordKind(req.Kind),
		Scenario:     scenario,
		BacklogRef:   fmt.Sprintf("%s/%s", req.Kind, req.Name),
		MilestoneID: strings.TrimSpace(req.Milestone),
		Trigger:      trigger,
		Approach:     approach,
		Outcome:      outcome,
		CreatedBy:    req.DecidedBy,
	})
	if err != nil {
		return "", err
	}
	return r.ID, nil
}
