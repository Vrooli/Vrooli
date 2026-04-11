package maintenance

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/control"
	"github.com/vrooli/vrooli/internal/network"
	"github.com/vrooli/vrooli/internal/process"
)

type Controller struct {
	Root string
	Home string
}

type SystemProcess struct {
	PID     int    `json:"pid"`
	PPID    int    `json:"ppid"`
	Command string `json:"command"`
}

type LockInfo = network.LockInfo
type PortListener = network.PortListener

type PortDiagnostic struct {
	Port            int            `json:"port"`
	Scenario        string         `json:"scenario,omitempty"`
	InUse           bool           `json:"in_use"`
	Listeners       []PortListener `json:"listeners,omitempty"`
	Lock            *LockInfo      `json:"lock,omitempty"`
	OrphanCount     int            `json:"orphan_count"`
	Recommendations []string       `json:"recommendations,omitempty"`
}

var (
	listProcessesFn          = listProcesses
	listLocksFn              = network.ListLocks
	readLockFileFn           = network.ReadLockFile
	listPortListenersFn      = network.ListenersForPort
	killProcessFn            = killProcess
	looksLikeVrooliProcessFn = looksLikeVrooliProcess
)

func NewController(root, home string) *Controller {
	return &Controller{
		Root: filepath.Clean(root),
		Home: filepath.Clean(home),
	}
}

func (c *Controller) ListLocks() ([]LockInfo, error) {
	return listLocksFn(c.Home)
}

func (c *Controller) CleanStaleLocks() (control.StopReport, error) {
	locks, err := c.ListLocks()
	if err != nil {
		return control.StopReport{}, err
	}

	cleaned := make([]control.ResultItem, 0)
	failed := make([]control.ResultItem, 0)
	for _, lock := range locks {
		if !lock.Stale {
			continue
		}
		if err := os.Remove(lock.Path); err != nil && !os.IsNotExist(err) {
			failed = append(failed, control.Failed(fmt.Sprintf("%d", lock.Port), err))
			continue
		}
		cleaned = append(cleaned, control.Stopped(fmt.Sprintf("%d", lock.Port), "Removed stale lock"))
	}

	return control.StopReport{
		Stopped: cleaned,
		Failed:  failed,
		Message: control.StopSummary(len(cleaned), len(failed)),
	}, nil
}

func (c *Controller) ListOrphans() ([]SystemProcess, error) {
	processes, err := listProcessesFn()
	if err != nil {
		return nil, err
	}

	tracked, err := trackedProcessIDs(c.Home)
	if err != nil {
		return nil, err
	}
	parentByPID := make(map[int]int, len(processes))
	for _, item := range processes {
		parentByPID[item.PID] = item.PPID
	}

	orphans := make([]SystemProcess, 0)
	self := os.Getpid()
	for _, item := range processes {
		if item.PID <= 1 || item.PID == self {
			continue
		}
		if !looksLikeVrooliProcessFn(c.Root, item.Command) {
			continue
		}
		if ancestorTracked(item.PID, parentByPID, tracked) {
			continue
		}
		orphans = append(orphans, item)
	}

	sort.Slice(orphans, func(i, j int) bool {
		if orphans[i].Command == orphans[j].Command {
			return orphans[i].PID < orphans[j].PID
		}
		return orphans[i].Command < orphans[j].Command
	})
	return orphans, nil
}

func (c *Controller) KillOrphans() (control.StopReport, error) {
	orphans, err := c.ListOrphans()
	if err != nil {
		return control.StopReport{}, err
	}

	stopped := make([]control.ResultItem, 0, len(orphans))
	failed := make([]control.ResultItem, 0)
	for _, item := range orphans {
		if err := killProcessFn(item.PID, false); err != nil && !isMissingProcessError(err) {
			failed = append(failed, control.Failed(strconv.Itoa(item.PID), err))
			continue
		}
		time.Sleep(150 * time.Millisecond)
		if process.IsPIDRunning(item.PID) {
			if err := killProcessFn(item.PID, true); err != nil && !isMissingProcessError(err) {
				failed = append(failed, control.Failed(strconv.Itoa(item.PID), err))
				continue
			}
		}
		stopped = append(stopped, control.Stopped(strconv.Itoa(item.PID), item.Command))
	}

	return control.StopReport{
		Stopped: stopped,
		Failed:  failed,
		Message: control.StopSummary(len(stopped), len(failed)),
	}, nil
}

func (c *Controller) DiagnosePort(port int, scenarioName string) (PortDiagnostic, error) {
	listeners, err := listPortListenersFn(port)
	if err != nil {
		return PortDiagnostic{}, err
	}

	lockPath := network.LockPath(c.Home, port)
	var lock *LockInfo
	if info, err := os.Stat(lockPath); err == nil && !info.IsDir() {
		lockInfo, err := readLockFileFn(lockPath)
		if err != nil {
			return PortDiagnostic{}, err
		}
		lock = &lockInfo
	}

	orphans, err := c.ListOrphans()
	if err != nil {
		return PortDiagnostic{}, err
	}

	diagnostic := PortDiagnostic{
		Port:        port,
		Scenario:    strings.TrimSpace(scenarioName),
		InUse:       len(listeners) > 0,
		Listeners:   listeners,
		Lock:        lock,
		OrphanCount: len(orphans),
	}
	diagnostic.Recommendations = buildRecommendations(port, diagnostic)
	return diagnostic, nil
}

func buildRecommendations(port int, diagnostic PortDiagnostic) []string {
	recommendations := make([]string, 0, 4)
	if diagnostic.Lock != nil && diagnostic.Lock.Stale {
		recommendations = append(recommendations, fmt.Sprintf("Clean stale lock file %s", diagnostic.Lock.Path))
	}
	if diagnostic.InUse {
		recommendations = append(recommendations, fmt.Sprintf("Stop the process currently listening on port %d", port))
	}
	if diagnostic.OrphanCount > 0 {
		recommendations = append(recommendations, "Run `vrooli cleanup orphans` to terminate orphaned Vrooli processes")
	}
	if len(recommendations) == 0 {
		recommendations = append(recommendations, "No lock or listener conflict detected; inspect scenario logs for the failing service")
	}
	return recommendations
}

func trackedProcessIDs(home string) (map[int]struct{}, error) {
	tracked := make(map[int]struct{})
	processRoot := filepath.Join(home, ".vrooli", "processes", "scenarios")
	entries, err := os.ReadDir(processRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return tracked, nil
		}
		return nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		records, err := process.ReadScenarioRecords(home, entry.Name())
		if err != nil {
			return nil, err
		}
		for _, record := range records {
			if record.PID > 0 {
				tracked[record.PID] = struct{}{}
			}
			if record.PGID > 0 {
				tracked[record.PGID] = struct{}{}
			}
		}
	}
	return tracked, nil
}

func ancestorTracked(pid int, parentByPID map[int]int, tracked map[int]struct{}) bool {
	current := pid
	for depth := 0; depth < 32 && current > 1; depth++ {
		if _, ok := tracked[current]; ok {
			return true
		}
		next, ok := parentByPID[current]
		if !ok || next == current {
			return false
		}
		current = next
	}
	return false
}

func looksLikeVrooliProcess(root, command string) bool {
	lower := strings.ToLower(strings.TrimSpace(command))
	if lower == "" {
		return false
	}
	for _, excluded := range []string{
		"zombie-detector",
		"vrooli cleanup",
		"vrooli orphans",
		"vrooli diagnose-port",
		"vrooli status",
		"vrooli-autoheal",
	} {
		if strings.Contains(lower, excluded) {
			return false
		}
	}

	root = strings.ToLower(filepath.Clean(root))
	scenariosPath := strings.ToLower(filepath.Join(root, "scenarios"))
	return strings.Contains(lower, "vrooli") ||
		strings.Contains(lower, scenariosPath) ||
		strings.Contains(lower, "/scenarios/") ||
		strings.Contains(lower, "node_modules/.bin/vite") ||
		strings.Contains(lower, "ecosystem-manager") ||
		strings.Contains(lower, "picker-wheel")
}

func listProcesses() ([]SystemProcess, error) {
	cmd := exec.Command("ps", "-eo", "pid=,ppid=,command=")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	processes := make([]SystemProcess, 0)
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		pid, err := strconv.Atoi(fields[0])
		if err != nil {
			continue
		}
		ppid, err := strconv.Atoi(fields[1])
		if err != nil {
			continue
		}
		processes = append(processes, SystemProcess{
			PID:     pid,
			PPID:    ppid,
			Command: strings.Join(fields[2:], " "),
		})
	}
	return processes, scanner.Err()
}

func isMissingProcessError(err error) bool {
	if err == nil {
		return false
	}
	text := strings.ToLower(err.Error())
	return strings.Contains(text, "process already finished") ||
		strings.Contains(text, "no such process")
}
