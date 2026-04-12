package maintenance

import (
	"fmt"
	"os"
	"path/filepath"
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
	Port               int                        `json:"port"`
	Scenario           string                     `json:"scenario,omitempty"`
	InUse              bool                       `json:"in_use"`
	Listeners          []PortListener             `json:"listeners,omitempty"`
	ListenerInspection network.ListenerInspection `json:"listener_inspection"`
	Lock               *LockInfo                  `json:"lock,omitempty"`
	OrphanCount        int                        `json:"orphan_count"`
	Recommendations    []string                   `json:"recommendations,omitempty"`
}

var (
	listLocksFn              = network.ListLocks
	readLockFileFn           = network.ReadLockFile
	inspectPortListenersFn   = network.InspectPortListeners
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
	cleanedLocks, err := network.PruneStaleLocks(c.Home)
	if err != nil {
		return control.StopReport{}, err
	}

	cleaned := make([]control.ResultItem, 0, len(cleanedLocks))
	failed := make([]control.ResultItem, 0)
	for _, lock := range cleanedLocks {
		cleaned = append(cleaned, control.Stopped(fmt.Sprintf("%d", lock.Port), "Removed stale lock"))
	}

	return control.StopReport{
		Stopped: cleaned,
		Failed:  failed,
		Message: control.StopSummary(len(cleaned), len(failed)),
	}, nil
}

func (c *Controller) ListOrphans() ([]SystemProcess, error) {
	snapshot, err := c.Snapshot()
	if err != nil {
		return nil, err
	}
	return append([]SystemProcess(nil), snapshot.Orphans...), nil
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
	inspection, err := inspectPortListenersFn(port)
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

	snapshot, err := c.Snapshot()
	if err != nil {
		return PortDiagnostic{}, err
	}

	diagnostic := PortDiagnostic{
		Port:               port,
		Scenario:           strings.TrimSpace(scenarioName),
		InUse:              len(inspection.Listeners) > 0,
		Listeners:          inspection.Listeners,
		ListenerInspection: inspection.Inspection,
		Lock:               lock,
		OrphanCount:        snapshot.OrphanProcesses,
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
	if !diagnostic.ListenerInspection.Available {
		recommendations = append(recommendations, fmt.Sprintf("Listener inspection unavailable: %s", diagnostic.ListenerInspection.Reason))
	}
	if diagnostic.OrphanCount > 0 {
		recommendations = append(recommendations, "Run `vrooli cleanup orphans` to terminate orphaned Vrooli processes")
	}
	if len(recommendations) == 0 {
		recommendations = append(recommendations, "No lock or listener conflict detected; inspect scenario logs for the failing service")
	}
	return recommendations
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

func isMissingProcessError(err error) bool {
	if err == nil {
		return false
	}
	text := strings.ToLower(err.Error())
	return strings.Contains(text, "process already finished") ||
		strings.Contains(text, "no such process")
}
