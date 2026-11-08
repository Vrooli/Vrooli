package repository

import (
	"context"

	"github.com/google/uuid"
)

// RunProcessBinding is a candidate locator, never proof of process ownership.
// Callers must verify the current process incarnation and ancestry.
type RunProcessBinding struct {
	ID        uuid.UUID
	RunnerPID int
}

// ManagedRunProcessReader is the lightweight projection used by authorization
// ownership checks. It includes every nonterminal, non-attached run with a
// persisted runner PID above one, including resting and legacy execution modes.
type ManagedRunProcessReader interface {
	ListManagedRunProcesses(context.Context) ([]RunProcessBinding, error)
}
