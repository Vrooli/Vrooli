//go:build linux

package orchestration

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"agent-manager/internal/repository"
	"github.com/vrooli/cli-core/cliutil"
)

const maxLocalOwnerAncestry = 256

type processAncestryEntry struct {
	binding   cliutil.ProcessBinding
	parentPID int
}

// activeRunOwnsPeer reports whether an active run's persisted runner PID is the
// verified local peer or one of its process ancestors. This is a cooperative
// CLI boundary check, not a human-identity proof.
func activeRunOwnsPeer(ctx context.Context, runs repository.RunRepository, peerPID int) (bool, error) {
	if runs == nil {
		return false, errors.New("run registry unavailable")
	}
	ancestry, err := collectLocalPeerAncestry(ctx, peerPID, cliutil.ObserveProcessBinding, readLocalProcessStat)
	if err != nil {
		return false, err
	}
	pids := make([]int, 0, len(ancestry))
	for _, entry := range ancestry {
		pids = append(pids, entry.binding.PID)
	}
	if len(pids) == 0 {
		return false, errors.New("local peer ancestry is empty")
	}
	rows, err := runs.List(ctx, repository.RunListFilter{
		ListFilter:      repository.ListFilter{Limit: 1},
		NonTerminalOnly: true,
		RunnerPIDs:      pids,
	})
	if err != nil {
		return false, err
	}
	if len(rows) == 0 {
		return false, nil
	}
	for _, run := range rows {
		if run == nil || run.Status.IsTerminal() || !containsProcessID(pids, run.RunnerPID) {
			return false, errors.New("run registry did not honor the exact active runner filter")
		}
	}
	return true, nil
}

func containsProcessID(pids []int, candidate int) bool {
	for _, pid := range pids {
		if pid == candidate {
			return true
		}
	}
	return false
}

func collectLocalPeerAncestry(
	ctx context.Context,
	peerPID int,
	observe func(int) (cliutil.ProcessBinding, error),
	readStat func(int) ([]byte, error),
) ([]processAncestryEntry, error) {
	peer, err := observe(peerPID)
	if err != nil {
		return nil, fmt.Errorf("observe local peer: %w", err)
	}
	if peer.PID != peerPID || peer.UID != os.Getuid() || peer.Boot == "" || peer.Start == "" {
		return nil, errors.New("local peer process binding is incomplete")
	}

	ancestry := make([]processAncestryEntry, 0, 16)
	observedBindings := map[int]cliutil.ProcessBinding{peerPID: peer}
	seen := map[int]struct{}{peerPID: {}}
	currentPID := peerPID
	for range maxLocalOwnerAncestry {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		stat, err := readStat(currentPID)
		if err != nil {
			return nil, fmt.Errorf("read local peer ancestry: %w", err)
		}
		parentPID, start, err := parseProcessParentAndStart(stat)
		if err != nil {
			return nil, err
		}
		current, err := observe(currentPID)
		previous, previouslyObserved := observedBindings[currentPID]
		if err != nil || !previouslyObserved || current != previous || current.Start != start || current.Boot != peer.Boot {
			return nil, errors.New("local peer ancestry changed during verification")
		}
		ancestry = append(ancestry, processAncestryEntry{binding: current, parentPID: parentPID})
		if parentPID == 1 {
			return recheckLocalPeerAncestry(ctx, ancestry, observe, readStat)
		}
		if parentPID <= 1 {
			return nil, errors.New("local peer ancestry is incomplete")
		}
		if _, ok := seen[parentPID]; ok {
			return nil, errors.New("local peer ancestry contains a cycle")
		}
		// UID and cgroup boundaries are intentionally not stop conditions: an
		// Agent Manager runner can remain an ancestor across those boundaries.
		// Exact runner-PID matching in the active registry decides ownership.
		parent, err := observe(parentPID)
		if err != nil || parent.PID != parentPID || parent.Boot != peer.Boot || parent.Start == "" {
			return nil, fmt.Errorf("observe local peer parent: %w", err)
		}
		observedBindings[parentPID] = parent
		seen[parentPID] = struct{}{}
		currentPID = parentPID
	}
	return nil, errors.New("local peer ancestry exceeds verification bound")
}

func recheckLocalPeerAncestry(
	ctx context.Context,
	ancestry []processAncestryEntry,
	observe func(int) (cliutil.ProcessBinding, error),
	readStat func(int) ([]byte, error),
) ([]processAncestryEntry, error) {
	for _, entry := range ancestry {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		stat, err := readStat(entry.binding.PID)
		if err != nil {
			return nil, fmt.Errorf("recheck local peer ancestry: %w", err)
		}
		parentPID, start, err := parseProcessParentAndStart(stat)
		if err != nil || start != entry.binding.Start || parentPID != entry.parentPID {
			return nil, errors.New("local peer ancestry changed during verification")
		}
		current, err := observe(entry.binding.PID)
		if err != nil || current != entry.binding {
			return nil, errors.New("local peer process incarnation changed during verification")
		}
	}
	return ancestry, nil
}

func readLocalProcessStat(pid int) ([]byte, error) {
	return os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
}

func parseProcessParentAndStart(stat []byte) (int, string, error) {
	end := strings.LastIndex(string(stat), ") ")
	if end < 0 {
		return 0, "", errors.New("invalid process stat in local peer ancestry")
	}
	fields := strings.Fields(string(stat)[end+2:])
	if len(fields) < 20 || fields[0] == "Z" || fields[0] == "X" {
		return 0, "", errors.New("local peer ancestry process is not live")
	}
	parentPID, err := strconv.Atoi(fields[1])
	if err != nil {
		return 0, "", errors.New("invalid parent PID in local peer ancestry")
	}
	return parentPID, fields[19], nil
}
