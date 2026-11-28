package backlog

import (
	"fmt"

	"swarm-manager/internal/backlogstatus"
)

// TransitionStatus is the single mutation point for a persisted backlog
// item's lifecycle. Workflow handlers still establish their own authority,
// evidence and side effects; this function owns the legal status edge.
func TransitionStatus(item *BacklogItem, next BacklogStatus, owner backlogstatus.TransitionOwner) error {
	if item == nil {
		return fmt.Errorf("transition backlog status: item is nil")
	}
	status, err := backlogstatus.Transition(string(item.Status), string(next), owner)
	if err != nil {
		return err
	}
	item.Status = BacklogStatus(status)
	return nil
}
