// Package supervision provides shared, host-local supervision authorities.
package supervision

import (
	"errors"
	"fmt"
	"runtime"
	"sort"
	"time"
)

type OwnerKind string

const (
	OwnerKindResource OwnerKind = "resource"
	OwnerKindScenario OwnerKind = "scenario"
)

// Owner is a durable claim that Vrooli owns one process.
type Owner struct {
	Kind         OwnerKind `json:"kind"`
	Name         string    `json:"name"`
	PID          int       `json:"pid"`
	ArtifactPath string    `json:"artifact_path,omitempty"`
	StartedAt    time.Time `json:"started_at"`
}

type ProcessInfo struct {
	PID       int
	StartedAt time.Time
}

type OwnershipSource interface {
	Owners() ([]Owner, error)
}

type ProcessTableSource interface {
	Processes() (map[int]ProcessInfo, error)
}

// UnsupportedProcessEvidenceError reports a platform where the start-time
// evidence needed for PID-reuse protection is unavailable.
type UnsupportedProcessEvidenceError struct {
	Platform string
}

func (e *UnsupportedProcessEvidenceError) Error() string {
	return fmt.Sprintf("process start-time evidence is unsupported on %s", e.Platform)
}

func IsUnsupportedProcessEvidence(err error) bool {
	var target *UnsupportedProcessEvidenceError
	return errors.As(err, &target)
}

// recordClockAllowance covers ps's whole-second timestamps and the bounded
// delay between process creation and atomically persisting its ownership
// record. A materially older process still fails the PID-reuse guard.
const recordClockAllowance = 2 * time.Second

// Index is an immutable live-PID-to-owner view.
type Index struct {
	byPID map[int]Owner
}

func BuildIndex(processes ProcessTableSource, sources ...OwnershipSource) (*Index, error) {
	processTable, err := processes.Processes()
	if err != nil {
		return nil, err
	}
	owners := make([]Owner, 0)
	for _, source := range sources {
		if source == nil {
			continue
		}
		items, err := source.Owners()
		if err != nil {
			return nil, err
		}
		owners = append(owners, items...)
	}
	sort.Slice(owners, func(i, j int) bool {
		if owners[i].PID == owners[j].PID {
			if owners[i].Kind == owners[j].Kind {
				return owners[i].Name < owners[j].Name
			}
			return owners[i].Kind < owners[j].Kind
		}
		return owners[i].PID < owners[j].PID
	})
	index := &Index{byPID: make(map[int]Owner, len(owners))}
	for _, owner := range owners {
		if owner.PID <= 0 {
			continue
		}
		process, live := processTable[owner.PID]
		if !live {
			continue
		}
		if process.StartedAt.IsZero() {
			return nil, &UnsupportedProcessEvidenceError{Platform: runtime.GOOS}
		}
		if process.StartedAt.Add(recordClockAllowance).Before(owner.StartedAt) {
			continue
		}
		if existing, duplicate := index.byPID[owner.PID]; duplicate {
			return nil, fmt.Errorf("pid %d has conflicting owners %s/%s and %s/%s", owner.PID, existing.Kind, existing.Name, owner.Kind, owner.Name)
		}
		index.byPID[owner.PID] = owner
	}
	return index, nil
}

func (i *Index) Owner(pid int) (Owner, bool) {
	if i == nil {
		return Owner{}, false
	}
	owner, ok := i.byPID[pid]
	return owner, ok
}

func (i *Index) Owners() []Owner {
	if i == nil {
		return []Owner{}
	}
	owners := make([]Owner, 0, len(i.byPID))
	for _, owner := range i.byPID {
		owners = append(owners, owner)
	}
	sort.Slice(owners, func(a, b int) bool { return owners[a].PID < owners[b].PID })
	return owners
}

type NativeProcessTableSource struct{}

func (NativeProcessTableSource) Processes() (map[int]ProcessInfo, error) {
	return readNativeProcessTable()
}
