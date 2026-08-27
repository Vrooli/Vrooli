package process

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/repocontractmeta"
	"github.com/vrooli/vrooli/internal/tuning"

	platform "github.com/vrooli/platform-go"
	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/config"
)

const (
	processParameterA = 2
	processParameterB = 24
)

type Record struct {
	PID        int       `json:"pid"`
	PGID       int       `json:"pgid,omitempty"`
	ProcessID  string    `json:"process_id,omitempty"`
	Phase      string    `json:"phase,omitempty"`
	Scenario   string    `json:"scenario,omitempty"`
	Step       string    `json:"step,omitempty"`
	Command    string    `json:"command,omitempty"`
	WorkingDir string    `json:"working_dir,omitempty"`
	LogFile    string    `json:"log_file,omitempty"`
	Port       int       `json:"port,omitempty"`
	PortKey    string    `json:"port_key,omitempty"`
	StartedAt  time.Time `json:"started_at"`
	Status     string    `json:"status,omitempty"`
}

type ScenarioRuntime struct {
	Name         string
	Records      []Record
	ProcessCount int
	StartedAt    *time.Time
	Runtime      string
}

var (
	isPIDRunningFn           = IsPIDRunning
	pidIsAliveFn             = platform.IsPIDRunning
	readProcessEnvironmentFn = platform.ReadProcessEnvironment
)

func HomeDir() (string, error) {
	home, err := config.HomeDir()
	// os.UserHomeDir may return the authenticated user's home alongside an
	// environment-related error. Process paths can safely use that concrete
	// value, and callers should not receive a misleading error with a usable
	// home directory.
	if strings.TrimSpace(home) != "" {
		return home, nil
	}
	return home, err
}

// ScenarioProcessDir resolves <home>/.vrooli/processes/scenarios/<name> from the
// runtime_home authority. Returns an error if the contract cannot be loaded.
func ScenarioProcessDir(home, name string) (string, error) {
	root, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyProcesses)
	if err != nil {
		return "", err
	}
	return filepath.Join(root, repocontractmeta.ScenarioDir, name), nil
}

// ScenarioLogsDir resolves <home>/.vrooli/logs/scenarios/<name>.
func ScenarioLogsDir(home, name string) (string, error) {
	root, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyLogs)
	if err != nil {
		return "", err
	}
	return filepath.Join(root, repocontractmeta.ScenarioDir, name), nil
}

// ScenarioLifecycleLogPath resolves <home>/.vrooli/logs/<name>.log.
func ScenarioLifecycleLogPath(home, name string) (string, error) {
	root, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyLogs)
	if err != nil {
		return "", err
	}
	return filepath.Join(root, name+".log"), nil
}

// ScenarioTestRunsDir resolves <home>/.vrooli/test-runs/<name> — the stable
// per-scenario root under which each test run gets its own <run-id>/ artifact
// directory (run.json + future Layer-2 phase logs/validator output).
func ScenarioTestRunsDir(home, name string) (string, error) {
	root, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyTestRuns)
	if err != nil {
		return "", err
	}
	return filepath.Join(root, name), nil
}

// ScenarioStateDir resolves <home>/.vrooli/state/scenarios.
func ScenarioStateDir(home string) (string, error) {
	root, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyState)
	if err != nil {
		return "", err
	}
	return filepath.Join(root, repocontractmeta.ScenarioDir), nil
}

func WriteScenarioRecord(home, name, step string, record Record) error {
	processDir, err := ScenarioProcessDir(home, name)
	if err != nil {
		return err
	}
	if _, err := config.EnsureOwnedDir(processDir); err != nil {
		return fmt.Errorf("create process dir %s: %w", processDir, err)
	}

	if strings.TrimSpace(record.Step) == "" {
		record.Step = step
	}
	if strings.TrimSpace(record.Scenario) == "" {
		record.Scenario = name
	}
	if record.StartedAt.IsZero() {
		record.StartedAt = time.Now().UTC()
	}

	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal process record: %w", err)
	}
	data = append(data, '\n')

	recordPath := filepath.Join(processDir, step+".json")
	if err := config.WriteOwnedFile(recordPath, data, tuning.PermFile); err != nil {
		return fmt.Errorf("write %s: %w", recordPath, err)
	}

	pidPath := filepath.Join(processDir, step+".pid")
	if err := config.WriteOwnedFile(pidPath, []byte(strconv.Itoa(record.PID)+"\n"), tuning.PermFile); err != nil {
		return fmt.Errorf("write %s: %w", pidPath, err)
	}
	return nil
}

func RemoveScenarioRecord(home, name, step string) error {
	processDir, err := ScenarioProcessDir(home, name)
	if err != nil {
		return err
	}
	recordPath := filepath.Join(processDir, step+".json")
	pidPath := filepath.Join(processDir, step+".pid")

	if err := os.Remove(recordPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove %s: %w", recordPath, err)
	}
	if err := os.Remove(pidPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove %s: %w", pidPath, err)
	}
	return nil
}

func ReadScenarioRecords(home, name string) ([]Record, error) {
	processDir, err := ScenarioProcessDir(home, name)
	if err != nil {
		return nil, err
	}
	files, err := filepath.Glob(filepath.Join(processDir, "*.json"))
	if err != nil {
		return nil, err
	}
	sort.Strings(files)

	records := make([]Record, 0, len(files))
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", file, err)
		}

		var record Record
		if err := json.Unmarshal(data, &record); err != nil {
			return nil, fmt.Errorf("parse %s: %w", file, err)
		}
		if record.Step == "" {
			record.Step = strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))
		}
		if record.Scenario == "" {
			record.Scenario = name
		}
		records = append(records, record)
	}

	return records, nil
}

func LiveRecords(records []Record) []Record {
	live := make([]Record, 0, len(records))
	for _, record := range records {
		if isPIDRunningFn(record.PID) {
			live = append(live, record)
		}
	}

	sort.Slice(live, func(i, j int) bool {
		if live[i].Step == live[j].Step {
			return live[i].PID < live[j].PID
		}
		return live[i].Step < live[j].Step
	})
	return live
}

func SummarizeScenario(name string, records []Record) ScenarioRuntime {
	live := LiveRecords(records)
	runtime := ScenarioRuntime{
		Name:         name,
		Records:      live,
		ProcessCount: len(live),
		Runtime:      "N/A",
	}
	if len(live) == 0 {
		return runtime
	}

	var startedAt *time.Time
	for _, record := range live {
		if record.StartedAt.IsZero() {
			continue
		}
		if startedAt == nil || record.StartedAt.Before(*startedAt) {
			timestamp := record.StartedAt
			startedAt = &timestamp
		}
	}
	runtime.StartedAt = startedAt
	if startedAt != nil {
		runtime.Runtime = humanDuration(time.Since(*startedAt))
	}
	return runtime
}

func DiscoverRunningScenarios(home string, valid func(string) bool) ([]ScenarioRuntime, error) {
	processesRoot, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyProcesses)
	if err != nil {
		return nil, err
	}
	processRoot := filepath.Join(processesRoot, repocontractmeta.ScenarioDir)
	entries, err := os.ReadDir(processRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	runtimes := make([]ScenarioRuntime, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if valid != nil && !valid(name) {
			continue
		}

		records, err := ReadScenarioRecords(home, name)
		if err != nil {
			return nil, err
		}
		runtime := SummarizeScenario(name, records)
		if runtime.ProcessCount > 0 {
			runtimes = append(runtimes, runtime)
		}
	}

	sort.Slice(runtimes, func(i, j int) bool {
		return runtimes[i].Name < runtimes[j].Name
	})
	return runtimes, nil
}

// IsPIDRunning reports process liveness straight from the PID. It must never
// route through os.FindProcess: on Windows FindProcess opens the process and
// fails with ERROR_ACCESS_DENIED for other-user/elevated PIDs, which would
// short-circuit a live process into false-dead before the platform primitive
// (whose access-denied⇒alive mapping exists for exactly that case) ever runs.
// False-dead evidence expires valid registry claims downstream.
func IsPIDRunning(pid int) bool {
	if pid <= 0 {
		return false
	}
	return pidIsAliveFn(pid)
}

func ReadEnvironmentPorts(records []Record, keys []string) map[string]int {
	allowed := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		if key != "" {
			allowed[key] = struct{}{}
		}
	}
	if len(allowed) == 0 {
		return map[string]int{}
	}

	ports := make(map[string]int)
	for _, record := range records {
		if record.PID <= 0 {
			continue
		}
		values, err := readProcessEnvironmentFn(record.PID)
		if err != nil {
			continue
		}
		for key := range allowed {
			if _, exists := ports[key]; exists {
				continue
			}
			if raw, ok := values[key]; ok {
				if port, err := strconv.Atoi(raw); err == nil {
					ports[key] = port
				}
			}
		}
	}

	return ports
}

func ReadEnvironment(pid int) (map[string]string, error) {
	return readProcessEnvironmentFn(pid)
}

func parseEnvironmentEntries(data []byte) map[string]string {
	values := make(map[string]string)
	for _, entry := range bytes.Split(data, []byte{0}) {
		if len(entry) == 0 {
			continue
		}
		parts := bytes.SplitN(entry, []byte{'='}, processParameterA)
		if len(parts) != 2 || len(parts[0]) == 0 {
			continue
		}
		values[string(parts[0])] = string(parts[1])
	}
	return values
}

func humanDuration(duration time.Duration) string {
	if duration < 0 {
		duration = 0
	}
	switch {
	case duration < time.Hour:
		return fmt.Sprintf("%.0fm", duration.Minutes())
	case duration < tuning.DailyRetentionWindow():
		return fmt.Sprintf("%.1fh", duration.Hours())
	default:
		return fmt.Sprintf("%.1fd", duration.Hours()/processParameterB)
	}
}

// InitiatorInfo names the process that initiated an operation, durably enough
// to answer "who did this?" after that process is gone.
//
// A bare PID cannot answer it. On a host where short-lived CLI invocations
// start work continuously, the initiator has usually exited before anyone
// asks, and PIDs are reused. Argv says what ran; the parent says who ran it
// (a shell, an agent, a service); the scope says where it belonged and, unlike
// the PID, survives the process.
type InitiatorInfo struct {
	PID        int
	Argv       string
	ParentPID  int
	ParentArgv string
	Scope      string
}

// Initiator captures the current process as the initiator of an operation.
// Every field is best-effort: a host that cannot report one leaves it empty
// rather than failing the operation the record describes.
func Initiator() InitiatorInfo {
	info := InitiatorInfo{
		PID: os.Getpid(),
		// os.Args, not the platform reader: our own argv is always available
		// and needs no syscall.
		Argv:      strings.Join(os.Args, " "),
		ParentPID: os.Getppid(),
	}
	if info.ParentPID > 0 {
		if argv, err := processCommandLineFn(info.ParentPID); err == nil {
			info.ParentArgv = argv
		}
	}
	if scope, err := processScopeFn(info.PID); err == nil {
		info.Scope = scope
	}
	return info
}

// Platform seams, overridable in tests.
var (
	processCommandLineFn = platform.ProcessCommandLine
	processScopeFn       = platform.ProcessScope
)
