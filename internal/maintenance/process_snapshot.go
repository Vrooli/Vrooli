package maintenance

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	platform "github.com/vrooli/platform-go"

	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/process"
	"github.com/vrooli/vrooli/internal/repocontractmeta"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
	"github.com/vrooli/vrooli/internal/shell"
	"github.com/vrooli/vrooli/internal/supervision"
)

const (
	processSnapshot = "\u2705"
)

const (
	processSnapshotCritical = "critical"
	ProcessHealthNormal     = "normal"
	processSnapshotValue    = processSnapshot
	processSnapshotWarning  = "warning"
)

const (
	processSnapshotParameterA      = 10
	processSnapshotParameterB      = 20
	processSnapshotParameterC      = 25
	processSnapshotParameterD      = 5
	processSnapshotParameterE      = 6
	initialProcessTableBufferBytes = 64 * 1024
	maxProcessTableLineBytes       = 4 * 1024 * 1024
)

type processTableEntry struct {
	PID        int
	PPID       int
	PGID       int
	SID        int // session id; used to keep dev-server worker subtrees out of the orphan list
	State      string
	Command    string
	Executable string // resolved /proc/<pid>/exe symlink target (empty on non-Linux or when unreadable)
	Cwd        string // resolved /proc/<pid>/cwd symlink target (empty on non-Linux or when unreadable)
	Cgroup     string // unified-hierarchy cgroup path (empty on non-Linux or when unreadable)
}

// agentSessionSlice is the user-manager slice every coding-agent session
// scope lives under. A process inside it is a session the launcher recorded
// as an editor lease; it is never an orphan, whatever its cwd or argv look
// like, because the registry's proof-of-death expiry owns its lifetime.
const agentSessionSlice = "/vrooli-agents.slice/"

func insideAgentSessionScope(entry processTableEntry) bool {
	return strings.Contains(entry.Cgroup, agentSessionSlice)
}

// ProcessSnapshot is the maintenance subsystem's canonical view of host-level
// Vrooli process health. API and CLI consumers should derive health and metrics
// from this snapshot rather than reimplementing process-table heuristics.
type ProcessSnapshot struct {
	TrackedProcesses int             `json:"tracked_processes"`
	RunningTracked   int             `json:"running_tracked"`
	ChildProcesses   int             `json:"child_processes"`
	TotalProcesses   int             `json:"total_processes"`
	ZombieProcesses  int             `json:"zombie_processes"`
	OrphanProcesses  int             `json:"orphan_processes"`
	Orphans          []SystemProcess `json:"orphans,omitempty"`
}

// HealthSnapshot is the derived health summary for a ProcessSnapshot.
type HealthSnapshot struct {
	ZombieCount   int    `json:"zombie_count"`
	ZombieStatus  string `json:"zombie_status"`
	ZombieEmoji   string `json:"zombie_emoji"`
	OrphanCount   int    `json:"orphan_count"`
	OrphanStatus  string `json:"orphan_status"`
	OrphanEmoji   string `json:"orphan_emoji"`
	OverallStatus string `json:"overall_status"`
}

var (
	listProcessTableFn           = listProcessTable
	processReadScenarioRecordsFn = process.ReadScenarioRecords
	runtimeProcessRefsFn         = runtimeTrackedProcessRefs
	ownershipIndexFn             = supervision.BuildHostIndex
)

// Snapshot gathers the current tracked-process and orphan-process state from the
// host and returns a single canonical maintenance view for downstream commands.
func (c *Controller) Snapshot() (ProcessSnapshot, error) {
	processTable, err := listProcessTableFn()
	if err != nil {
		return ProcessSnapshot{}, err
	}

	ownership, err := ownershipIndexFn(c.Home)
	if err != nil {
		return ProcessSnapshot{}, fmt.Errorf("build process ownership index: %w", err)
	}

	tracked, trackedSIDs, trackedCount, runningTracked, err := trackedProcessStats(c.Home, processTable, ownership)
	if err != nil {
		return ProcessSnapshot{}, err
	}

	orphans := collectOrphans(c.Root, c.Home, processTable, tracked, trackedSIDs)
	zombieCount := 0
	for _, entry := range processTable {
		if strings.HasPrefix(entry.State, "Z") {
			zombieCount++
		}
	}

	totalProcesses := len(processTable)
	childProcesses := totalProcesses - runningTracked
	if _, exists := processTable[os.Getpid()]; exists {
		childProcesses--
	}
	if childProcesses < 0 {
		childProcesses = 0
	}

	return ProcessSnapshot{
		TrackedProcesses: trackedCount,
		RunningTracked:   runningTracked,
		ChildProcesses:   childProcesses,
		TotalProcesses:   totalProcesses,
		ZombieProcesses:  zombieCount,
		OrphanProcesses:  len(orphans),
		Orphans:          orphans,
	}, nil
}

// HealthSnapshot converts raw process counts into the status vocabulary exposed
// by the API health and process-metrics endpoints.
func (s ProcessSnapshot) HealthSnapshot() HealthSnapshot {
	zombieStatus, zombieEmoji := interpretZombieStatus(s.ZombieProcesses)
	orphanStatus, orphanEmoji := interpretOrphanStatus(s.OrphanProcesses)
	overallStatus := scenarioruntime.HealthStatusHealthy
	switch {
	case zombieStatus == processSnapshotCritical || orphanStatus == processSnapshotCritical:
		overallStatus = processSnapshotCritical
	case zombieStatus == processSnapshotWarning || orphanStatus == processSnapshotWarning:
		overallStatus = processSnapshotWarning
	case zombieStatus == ProcessHealthNormal || orphanStatus == ProcessHealthNormal:
		overallStatus = ProcessHealthNormal
	}
	return HealthSnapshot{
		ZombieCount:   s.ZombieProcesses,
		ZombieStatus:  zombieStatus,
		ZombieEmoji:   zombieEmoji,
		OrphanCount:   s.OrphanProcesses,
		OrphanStatus:  orphanStatus,
		OrphanEmoji:   orphanEmoji,
		OverallStatus: overallStatus,
	}
}

func interpretZombieStatus(count int) (string, string) {
	switch {
	case count == 0:
		return scenarioruntime.HealthStatusHealthy, processSnapshot
	case count <= processSnapshotParameterD:
		return ProcessHealthNormal, processSnapshot
	case count <= processSnapshotParameterB:
		return processSnapshotWarning, "⚠️"
	default:
		return processSnapshotCritical, "🔴"
	}
}

func interpretOrphanStatus(count int) (string, string) {
	switch {
	case count == 0:
		return scenarioruntime.HealthStatusHealthy, processSnapshot
	case count <= processSnapshotParameterA:
		return ProcessHealthNormal, "✅"
	case count <= processSnapshotParameterC:
		return processSnapshotWarning, "⚠️"
	default:
		return processSnapshotCritical, "🔴"
	}
}

func trackedProcessStats(home string, processTable map[int]processTableEntry, ownership *supervision.Index) (map[int]struct{}, map[int]struct{}, int, int, error) {
	tracked := make(map[int]struct{})
	trackedSIDs := make(map[int]struct{})
	trackedCount := 0
	runningTracked := 0
	addTrackedPID := func(pid int) {
		if pid <= 0 {
			return
		}
		if _, exists := tracked[pid]; exists {
			return
		}
		tracked[pid] = struct{}{}
		trackedCount++
		if _, running := processTable[pid]; running {
			runningTracked++
		}
	}

	// Recorded ownership is authoritative. Only owners present in this same
	// process-table snapshot are added; stale and reused PIDs were already
	// removed by the ownership index's start-time guard.
	for _, owner := range ownership.Owners() {
		if _, running := processTable[owner.PID]; running {
			addTrackedPID(owner.PID)
		}
	}

	processesDir, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyProcesses)
	if err != nil {
		return nil, nil, 0, 0, err
	}
	processRoot := filepath.Join(processesDir, repocontractmeta.ScenarioDir)
	entries, err := os.ReadDir(processRoot)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, nil, 0, 0, err
		}
	} else {
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			records, err := processReadScenarioRecordsFn(home, entry.Name())
			if err != nil {
				return nil, nil, 0, 0, err
			}
			for _, record := range records {
				addTrackedPID(record.PID)
				if record.PGID > 0 {
					tracked[record.PGID] = struct{}{}
				}
			}
		}
	}

	refs, err := runtimeProcessRefsFn(home)
	if err != nil {
		return nil, nil, 0, 0, err
	}
	for _, ref := range refs {
		addTrackedPID(ref.PID)
		if ref.PGID > 0 {
			tracked[ref.PGID] = struct{}{}
		}
	}

	// Derive the set of session IDs owned by tracked processes. Subtree
	// workers (e.g. node tinypool / esbuild children of a tracked dev server)
	// often have their own PGID via setsid, so they miss the PGID match in
	// isTrackedOrAncestorTracked. The session ID is stable across the subtree
	// and lets us classify them as tracked even when the ppid chain is broken
	// by a reparented intermediate process.
	for pid := range tracked {
		entry, ok := processTable[pid]
		if !ok {
			continue
		}
		if entry.SID > 1 {
			trackedSIDs[entry.SID] = struct{}{}
		}
	}

	return tracked, trackedSIDs, trackedCount, runningTracked, nil
}

func collectOrphans(root, home string, processTable map[int]processTableEntry, tracked, trackedSIDs map[int]struct{}) []SystemProcess {
	orphans := make([]SystemProcess, 0)
	self := os.Getpid()
	memo := make(map[int]bool)
	visiting := make(map[int]bool)

	for pid, entry := range processTable {
		if pid <= 1 || pid == self {
			continue
		}
		if !looksLikeVrooliProcessFn(root, home, entry) {
			continue
		}
		if insideAgentSessionScope(entry) {
			continue
		}
		// Never classify transient `vrooli` CLI invocations as orphans: they
		// are legitimate short-lived user commands (e.g. `vrooli scenario
		// restart <name>`) that don't register a process record, and
		// SIGTERM'ing a sibling vrooli invocation during `cleanup orphans`
		// would disrupt in-flight user work.
		if isVrooliCLIExecutable(home, entry.Executable) {
			continue
		}
		if isControlPlaneAPIExecutable(entry) {
			continue
		}
		if isTrackedOrAncestorTracked(pid, tracked, trackedSIDs, processTable, memo, visiting) {
			continue
		}
		orphans = append(orphans, SystemProcess{
			PID:     entry.PID,
			PPID:    entry.PPID,
			Command: entry.Command,
		})
	}

	sort.Slice(orphans, func(i, j int) bool {
		if orphans[i].Command == orphans[j].Command {
			return orphans[i].PID < orphans[j].PID
		}
		return orphans[i].Command < orphans[j].Command
	})
	return orphans
}

func isTrackedOrAncestorTracked(pid int, tracked, trackedSIDs map[int]struct{}, processTable map[int]processTableEntry, memo map[int]bool, visiting map[int]bool) bool {
	if _, ok := tracked[pid]; ok {
		memo[pid] = true
		return true
	}
	if value, ok := memo[pid]; ok {
		return value
	}
	entry, ok := processTable[pid]
	if !ok {
		memo[pid] = false
		return false
	}
	if entry.PGID > 0 {
		if _, ok := tracked[entry.PGID]; ok {
			memo[pid] = true
			return true
		}
	}
	if entry.SID > 1 {
		if _, ok := trackedSIDs[entry.SID]; ok {
			memo[pid] = true
			return true
		}
	}
	if entry.PPID == 0 || entry.PPID == 1 {
		memo[pid] = false
		return false
	}
	if visiting[pid] {
		memo[pid] = false
		return false
	}
	visiting[pid] = true
	trackedAncestor := isTrackedOrAncestorTracked(entry.PPID, tracked, trackedSIDs, processTable, memo, visiting)
	visiting[pid] = false
	memo[pid] = trackedAncestor
	return trackedAncestor
}

func listProcessTable() (map[int]processTableEntry, error) {
	output, err := shell.Output(shell.Spec{
		Name: "ps",
		// `-axo` and `sess` are accepted by both procps (Linux) and the
		// BSD-derived macOS ps. The previous `-eo ... sid=` form is Linux-only
		// and made every Mac health/cleanup probe fail before parsing a row.
		Args: []string{"-axo", "pid=,ppid=,pgid=,sess=,state=,command="},
	})
	if err != nil {
		return nil, fmt.Errorf("inspect process table: %w", err)
	}

	processTable := make(map[int]processTableEntry)
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	// Command lines can legitimately approach the host ARG_MAX (for example a
	// generated test invocation). Scanner's 64 KiB default must not make the
	// entire ownership classifier unavailable because one process has a long
	// argv.
	scanner.Buffer(make([]byte, initialProcessTableBufferBytes), maxProcessTableLineBytes)
	for scanner.Scan() {
		entry, ok := parseProcessTableLine(scanner.Text())
		if !ok {
			continue
		}
		processTable[entry.PID] = entry
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan process table: %w", err)
	}
	return processTable, nil
}

func parseProcessTableLine(line string) (processTableEntry, bool) {
	line = strings.TrimSpace(line)
	if line == "" {
		return processTableEntry{}, false
	}
	fields := strings.Fields(line)
	if len(fields) < processSnapshotParameterE {
		return processTableEntry{}, false
	}
	pid, err := strconv.Atoi(fields[0])
	if err != nil {
		return processTableEntry{}, false
	}
	ppid, err := strconv.Atoi(fields[1])
	if err != nil {
		return processTableEntry{}, false
	}
	pgid, err := strconv.Atoi(fields[2])
	if err != nil {
		return processTableEntry{}, false
	}
	sid, err := strconv.Atoi(fields[3])
	if err != nil {
		return processTableEntry{}, false
	}
	exe, _ := os.Readlink(fmt.Sprintf("/proc/%d/exe", pid))
	cwd, _ := os.Readlink(fmt.Sprintf("/proc/%d/cwd", pid))
	cgroup, _ := platform.ProcessScope(pid)
	return processTableEntry{
		PID:        pid,
		PPID:       ppid,
		PGID:       pgid,
		SID:        sid,
		State:      fields[4],
		Command:    strings.Join(fields[5:], " "),
		Executable: exe,
		Cwd:        cwd,
		Cgroup:     cgroup,
	}, true
}

// readProcessEntryFn reads a single process's table entry. It is overridable in
// tests and is used by KillOrphans to re-validate an orphan right before
// sending a signal, guarding against the PID being recycled between the
// snapshot and the kill. The implementation is per-platform
// (process_entry_{linux,darwin,other}.go); platforms without an
// implementation report not-found so the kill guard fails safe.
var readProcessEntryFn = readProcessEntry
