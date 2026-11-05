package apply

import (
	"architecture-cartographer/internal/conflicts"
)

// OperationsFromConflict derives the deterministic operation list a
// resolved conflict implies. Used by the planner; v0.1 supports the
// move_file fix kind (the only one needed by the day-one detectors).
//
// The function is pure — given the same conflict, returns the same
// operations. Apply execution (Phase 13) consumes these to enact the
// real file changes.
func OperationsFromConflict(c conflicts.Conflict) []Operation {
	var ops []Operation
	for _, fix := range c.SuggestedFixes {
		switch fix.Kind {
		case conflicts.FixKindMoveFile:
			ops = append(ops, Operation{
				ID:                  "op:" + c.ID + ":" + fix.ID,
				Kind:                OperationKindMoveFile,
				Payload:             fix.Payload,
				ResolvesConflictIDs: []string{c.ID},
			})
		case conflicts.FixKindReassignDomain:
			ops = append(ops, Operation{
				ID:                  "op:" + c.ID + ":" + fix.ID,
				Kind:                OperationKindRewriteImport,
				Payload:             fix.Payload,
				ResolvesConflictIDs: []string{c.ID},
			})
		}
	}
	return ops
}
