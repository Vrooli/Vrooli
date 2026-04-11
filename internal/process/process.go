package process

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/config"
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

func HomeDir() (string, error) {
	return config.HomeDir()
}

func ScenarioProcessDir(home, name string) string {
	return filepath.Join(home, ".vrooli", "processes", "scenarios", name)
}

func ScenarioLogsDir(home, name string) string {
	return filepath.Join(home, ".vrooli", "logs", "scenarios", name)
}

func ScenarioLifecycleLogPath(home, name string) string {
	return filepath.Join(home, ".vrooli", "logs", name+".log")
}

func ScenarioStateDir(home string) string {
	return filepath.Join(home, ".vrooli", "state", "scenarios")
}

func ScenarioDegradedPath(home, name string) string {
	return filepath.Join(ScenarioProcessDir(home, name), "degraded.json")
}

func WriteScenarioRecord(home, name, step string, record Record) error {
	processDir := ScenarioProcessDir(home, name)
	if err := os.MkdirAll(processDir, 0o755); err != nil {
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
	if err := os.WriteFile(recordPath, data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", recordPath, err)
	}

	pidPath := filepath.Join(processDir, step+".pid")
	if err := os.WriteFile(pidPath, []byte(strconv.Itoa(record.PID)+"\n"), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", pidPath, err)
	}
	return nil
}

func RemoveScenarioRecord(home, name, step string) error {
	processDir := ScenarioProcessDir(home, name)
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
	processDir := filepath.Join(home, ".vrooli", "processes", "scenarios", name)
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
		if IsPIDRunning(record.PID) {
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
	processRoot := filepath.Join(home, ".vrooli", "processes", "scenarios")
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

func IsPIDRunning(pid int) bool {
	if pid <= 0 {
		return false
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return processIsAlive(process)
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
		values, err := readProcessEnvironment(record.PID)
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

func humanDuration(duration time.Duration) string {
	if duration < 0 {
		duration = 0
	}
	switch {
	case duration < time.Hour:
		return fmt.Sprintf("%.0fm", duration.Minutes())
	case duration < 24*time.Hour:
		return fmt.Sprintf("%.1fh", duration.Hours())
	default:
		return fmt.Sprintf("%.1fd", duration.Hours()/24)
	}
}
