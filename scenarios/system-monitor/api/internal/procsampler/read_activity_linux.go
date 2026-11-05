//go:build linux

package procsampler

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

const readActivityProcessLimit = 4096

// InvestigateReadActivity performs an explicitly requested, bounded pair of
// reads. Normal scheduler collection does not read /proc/<pid>/io.
func InvestigateReadActivity(ctx context.Context) (ReadActivity, error) {
	fs := osProcFS{}
	first, err := sampleReadActivity(ctx, fs, time.Now, readActivityProcessLimit, false)
	if err != nil {
		return ReadActivity{}, err
	}
	timer := time.NewTimer(2 * time.Second)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ReadActivity{}, ctx.Err()
	case <-timer.C:
	}
	last, err := sampleReadActivity(ctx, fs, time.Now, readActivityProcessLimit, true)
	if err != nil {
		return ReadActivity{}, err
	}
	return summarizeReadActivity(first, last, 20), nil
}

func sampleReadActivity(ctx context.Context, fs procFS, now func() time.Time, limit int, withDetails bool) (readActivitySnapshot, error) {
	pids, err := fs.PIDs()
	if err != nil {
		return readActivitySnapshot{}, fmt.Errorf("enumerate processes: %w", err)
	}
	sort.Ints(pids)
	snapshot := readActivitySnapshot{points: make(map[int]readActivityPoint), enumerated: len(pids), observedAt: now()}
	markReadError := func(err error) {
		snapshot.unreadable++
		if errors.Is(err, os.ErrPermission) {
			snapshot.permissionDenied++
		}
		if errors.Is(err, os.ErrNotExist) {
			snapshot.disappeared++
		}
	}
	if len(pids) > limit {
		pids, snapshot.truncated = pids[:limit], true
	}
	for _, pid := range pids {
		if err := ctx.Err(); err != nil {
			return readActivitySnapshot{}, err
		}
		before, err := fs.ReadFile(pid, "stat")
		if err != nil {
			markReadError(err)
			continue
		}
		process, ok := parseStat(string(before))
		if !ok || process.StartTimeTicks == 0 {
			snapshot.unreadable++
			continue
		}
		ioData, err := fs.ReadFile(pid, "io")
		if err != nil {
			markReadError(err)
			continue
		}
		readAt := now()
		requested, storage, err := parseReadCounters(string(ioData))
		if err != nil {
			snapshot.unreadable++
			continue
		}
		after, err := fs.ReadFile(pid, "stat")
		if err != nil {
			markReadError(err)
			continue
		}
		current, ok := parseStat(string(after))
		if !ok || current.StartTimeTicks != process.StartTimeTicks {
			snapshot.unreadable++
			continue
		}
		process.PID = pid
		// Only cheap host paths are inspected; command lines are used for owner
		// attribution but are never returned in the investigation result.
		if withDetails {
			if cmd, err := fs.ReadFile(pid, "cmdline"); err == nil {
				if len(cmd) > 4096 {
					cmd = cmd[:4096]
				}
				process.Cmdline = renderCmdline(cmd)
			}
			if cwd, err := fs.Readlink(pid, "cwd"); err == nil {
				process.Cwd = cwd
			}
			identity, err := fs.ReadFile(pid, "stat")
			if err != nil {
				markReadError(err)
				continue
			}
			verified, ok := parseStat(string(identity))
			if !ok || verified.StartTimeTicks != process.StartTimeTicks {
				snapshot.unreadable++
				continue
			}
		}
		snapshot.points[pid] = readActivityPoint{process: process, requested: requested, storage: storage, observedAt: readAt}
	}
	return snapshot, nil
}

func parseReadCounters(raw string) (uint64, uint64, error) {
	var requested, storage uint64
	var haveRequested, haveStorage bool
	for _, line := range strings.Split(raw, "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 || (fields[0] != "rchar:" && fields[0] != "read_bytes:") {
			continue
		}
		value, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			return 0, 0, err
		}
		if fields[0] == "rchar:" {
			requested, haveRequested = value, true
		} else {
			storage, haveStorage = value, true
		}
	}
	if !haveRequested || !haveStorage {
		return 0, 0, errors.New("required read counters unavailable")
	}
	return requested, storage, nil
}
