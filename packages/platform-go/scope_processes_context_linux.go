//go:build linux

package platform

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

type scopeUnitState struct {
	load, active string
}

// The output cap bounds a complete systemd scope inventory. Truncation is an
// incomplete snapshot and cannot be used to prove any requested unit missing.
const maxScopeUnitListOutputBytes = 16 << 20

var queryScopeStatesBatch = queryScopeStatesBatchCommand

func queryScopeStatesBatchCommand(ctx context.Context, units []string) (map[string]scopeUnitState, error) {
	if len(units) == 0 {
		return map[string]scopeUnitState{}, nil
	}
	cmd := exec.CommandContext(ctx, "systemctl", "--user", "list-units", "--all", "--type=scope", "--plain", "--no-legend", "--no-pager", "--full")
	cmd.Env = userManagerEnv(nil)
	var output scopeUnitListOutput
	output.limit = maxScopeUnitListOutputBytes
	cmd.Stdout = &output
	cmd.Stderr = io.Discard
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("list user scope units: %w", err)
	}
	if output.overflow {
		return nil, fmt.Errorf("list user scope units exceeded %d bytes", maxScopeUnitListOutputBytes)
	}
	return parseScopeUnitList(output.String(), units)
}

type scopeUnitListOutput struct {
	bytes.Buffer
	limit    int
	overflow bool
}

func (b *scopeUnitListOutput) Write(p []byte) (int, error) {
	remaining := b.limit - b.Len()
	if remaining <= 0 {
		b.overflow = true
		return len(p), nil
	}
	if len(p) > remaining {
		b.overflow = true
		_, _ = b.Buffer.Write(p[:remaining])
		return len(p), nil
	}
	return b.Buffer.Write(p)
}

// parseScopeUnitList accepts only a complete `list-units --all --type=scope`
// snapshot. An exact launcher-owned unit in the snapshot must be loaded and
// inactive to prove absence. A requested unit omitted from the complete list
// is not-found/inactive; substring and escaped-name matches are never used.
func parseScopeUnitList(output string, requested []string) (map[string]scopeUnitState, error) {
	wanted := make(map[string]struct{}, len(requested))
	states := make(map[string]scopeUnitState, len(requested))
	for _, unit := range requested {
		if unit == "" {
			return nil, fmt.Errorf("scope unit list request contains an empty unit name")
		}
		if _, exists := wanted[unit]; exists {
			return nil, fmt.Errorf("scope unit list request contains duplicate unit %s", unit)
		}
		wanted[unit] = struct{}{}
	}
	for lineNo, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 4 || !strings.HasSuffix(fields[0], ".scope") {
			return nil, fmt.Errorf("parse scope unit list line %d: expected unit, load, active, and sub-state", lineNo+1)
		}
		unit := fields[0]
		if _, match := wanted[unit]; !match {
			continue
		}
		if _, exists := states[unit]; exists {
			return nil, fmt.Errorf("systemd returned duplicate exact scope unit %s", unit)
		}
		states[unit] = scopeUnitState{load: fields[1], active: fields[2]}
	}
	for _, unit := range requested {
		if _, found := states[unit]; !found {
			states[unit] = scopeUnitState{load: "not-found", active: "inactive"}
		}
	}
	return states, nil
}

func scopeProcessesContext(ctx context.Context, ref ScopeRef) ([]int, error) {
	observations := scopeProcessesBatchContext(ctx, []ScopeRef{ref})
	return observations[0].PIDs, observations[0].Err
}

func scopeProcessesBatchContext(ctx context.Context, refs []ScopeRef) []ScopeProcessObservation {
	observations := make([]ScopeProcessObservation, len(refs))
	byUnit := map[string][]int{}
	for i, ref := range refs {
		if err := ctx.Err(); err != nil {
			observations[i].Err = err
			continue
		}
		pids, err := scopeProcesses(ref)
		if err == nil {
			observations[i].PIDs = pids
			continue
		}
		if !errors.Is(err, os.ErrNotExist) {
			observations[i].Err = err
			continue
		}
		unit, validateErr := validateGoneScope(ref)
		if validateErr != nil {
			observations[i].Err = validateErr
			continue
		}
		byUnit[unit] = append(byUnit[unit], i)
	}
	units := make([]string, 0, len(byUnit))
	for unit := range byUnit {
		units = append(units, unit)
	}
	if len(units) == 0 {
		return observations
	}
	sort.Strings(units)
	states, err := queryScopeStatesBatch(ctx, units)
	for _, unit := range units {
		for _, i := range byUnit[unit] {
			if err != nil {
				observations[i].Err = fmt.Errorf("scope gone state unavailable: %w", err)
				continue
			}
			state, exists := states[unit]
			if !exists {
				observations[i].Err = fmt.Errorf("scope gone state unavailable: manager omitted exact unit %s", unit)
				continue
			}
			if state.active != "inactive" || (state.load != "not-found" && state.load != "loaded") {
				observations[i].Err = fmt.Errorf("scope cgroup absent but unit state is load=%s active=%s", state.load, state.active)
			}
		}
	}
	return observations
}

func validateGoneScope(ref ScopeRef) (string, error) {
	dir, err := readableCgroup(ref)
	if err != nil {
		return "", err
	}
	if _, err = os.Stat(dir); err == nil {
		return "", fmt.Errorf("scope directory exists but membership is unavailable: %s", ref.Path)
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	if _, err = freezableCgroup(ref); err != nil {
		return "", err
	}
	unit := ref.Name
	if !strings.HasSuffix(unit, scopeSuffix) {
		unit += scopeSuffix
	}
	if ref.Name == "" || filepath.Base(dir) != unit {
		return "", errors.New("gone scope name/path identity mismatch")
	}
	return unit, nil
}
