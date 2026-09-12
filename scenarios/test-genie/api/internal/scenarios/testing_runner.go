package scenarios

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/vrooli/envkit-go"
)

// TestingRunner executes scenario testing commands described by TestingCapabilities.
type TestingRunner struct {
	Timeout time.Duration
	Output  io.Writer
	LogDir  string
}

type TestingRunnerResult struct {
	Command     []string
	LogPath     string
	SkipSummary SkipSummary
}

// SkipSummary is the minimum run-level truth needed to prevent a green test
// result from being mistaken for full execution. Counts come from Go's JSON
// test events when the command emits them; platform skips come from the shared
// machine-readable skip helper.
type SkipSummary struct {
	Platform     string `json:"platform"`
	Executed     int    `json:"executed"`
	Skipped      int    `json:"skipped"`
	Total        int    `json:"total"`
	Budget       int    `json:"budget"`
	WithinBudget bool   `json:"withinBudget"`
}

const defaultTestingTimeout = 10 * time.Minute

// Run executes the preferred testing command (or specific type when provided).
func (r TestingRunner) Run(ctx context.Context, caps TestingCapabilities, preferred string) (*TestingRunnerResult, error) {
	return r.RunWithArgs(ctx, caps, preferred, nil)
}

// RunWithArgs executes the preferred testing command and appends extra args (e.g., path filters).
func (r TestingRunner) RunWithArgs(ctx context.Context, caps TestingCapabilities, preferred string, extraArgs []string) (*TestingRunnerResult, error) {
	cmdSpec := caps.SelectCommand(preferred)
	if cmdSpec == nil {
		return nil, fmt.Errorf("no testing commands available for scenario")
	}

	timeout := r.Timeout
	if timeout <= 0 {
		timeout = defaultTestingTimeout
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	if len(cmdSpec.Command) == 0 {
		return nil, fmt.Errorf("testing command for %s is empty", cmdSpec.Type)
	}
	name := cmdSpec.Command[0]
	args := cmdSpec.Command[1:]
	if len(extraArgs) > 0 {
		args = append(args, extraArgs...)
	}
	skipDir, err := os.MkdirTemp("", "vrooli-test-skips-")
	if err != nil {
		return nil, fmt.Errorf("create skip evidence directory: %w", err)
	}
	defer os.RemoveAll(skipDir)
	skipPath := filepath.Join(skipDir, "platform-skips.jsonl")
	goWorkDir, err := createGoWorkDir("test-genie-")
	if err != nil {
		return nil, fmt.Errorf("create Go work directory: %w", err)
	}
	defer os.RemoveAll(goWorkDir)

	command := exec.CommandContext(runCtx, name, args...)
	command.Env = envkit.Toolchain(envkit.WithOverlay(envkit.Env(os.Environ()), envkit.SameScenario, envkit.Env{"VROOLI_SKIP_RECORD_PATH=" + skipPath, "GOTMPDIR=" + goWorkDir}), envkit.ToolchainOptions{})
	if cmdSpec.WorkingDir != "" {
		command.Dir = cmdSpec.WorkingDir
	}
	var logFile *os.File
	var logPath string
	if r.LogDir != "" {
		filename := fmt.Sprintf("scenario-tests-%d.log", time.Now().UnixNano())
		logPath = filepath.Join(r.LogDir, filename)
		if err := os.MkdirAll(r.LogDir, 0o755); err == nil {
			if f, err := os.Create(logPath); err == nil {
				logFile = f
				defer logFile.Close()
				command.Stdout = logFile
				command.Stderr = logFile
			}
		}
	}
	if command.Stdout == nil && r.Output != nil {
		command.Stdout = r.Output
		command.Stderr = r.Output
	}

	if err := command.Run(); err != nil {
		if runCtx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("%s testing command timed out after %s", cmdSpec.Type, timeout)
		}
		return nil, fmt.Errorf("%s testing command failed: %w", cmdSpec.Type, err)
	}
	summary, err := collectSkipSummary(skipPath, logPath, cmdSpec.WorkingDir)
	if err != nil {
		return nil, err
	}
	if !summary.WithinBudget {
		return nil, fmt.Errorf("platform skip budget exceeded: %d skips > %d budget for %s", summary.Skipped, summary.Budget, summary.Platform)
	}

	return &TestingRunnerResult{
		Command:     cmdSpec.Command,
		LogPath:     logPath,
		SkipSummary: summary,
	}, nil
}

func createGoWorkDir(prefix string) (string, error) {
	base := strings.TrimSpace(os.Getenv("VROOLI_HOME"))
	if base == "" {
		if home, err := os.UserHomeDir(); err == nil && strings.TrimSpace(home) != "" {
			base = filepath.Join(home, ".vrooli")
		} else {
			base = os.TempDir()
		}
	}
	root := filepath.Join(base, "tmp", "go-work")
	if err := os.MkdirAll(root, 0o755); err != nil {
		return "", err
	}
	return os.MkdirTemp(root, prefix)
}

type skipRecord struct {
	Platform string `json:"platform"`
}

type goTestEvent struct {
	Action string `json:"Action"`
	Test   string `json:"Test"`
}

func collectSkipSummary(skipPath, logPath, workingDir string) (SkipSummary, error) {
	summary := SkipSummary{Platform: normalizedPlatform(runtimeGOOS())}
	records, err := readSkipRecords(skipPath)
	if err != nil {
		return summary, err
	}
	summary.Skipped = len(records)
	if logPath != "" {
		executed, total, err := countGoTestEvents(logPath)
		if err != nil {
			return summary, err
		}
		summary.Executed = executed
		summary.Total = total
	}
	budget, err := readSkipBudget(workingDir, summary.Platform)
	if err != nil {
		return summary, err
	}
	summary.Budget = budget
	summary.WithinBudget = summary.Skipped <= budget
	return summary, nil
}

func readSkipRecords(path string) ([]skipRecord, error) {
	file, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("open skip evidence: %w", err)
	}
	defer file.Close()
	var records []skipRecord
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var record skipRecord
		if err := json.Unmarshal(scanner.Bytes(), &record); err != nil {
			return nil, fmt.Errorf("decode skip evidence: %w", err)
		}
		records = append(records, record)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read skip evidence: %w", err)
	}
	return records, nil
}

func countGoTestEvents(path string) (executed, total int, err error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, 0, err
	}
	defer file.Close()
	tests := map[string]struct{ terminal bool }{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var event goTestEvent
		if json.Unmarshal(scanner.Bytes(), &event) != nil || event.Test == "" {
			continue
		}
		tests[event.Test] = struct{ terminal bool }{terminal: event.Action == "pass" || event.Action == "fail" || event.Action == "skip"}
		if event.Action == "pass" || event.Action == "fail" {
			executed++
		}
	}
	if err := scanner.Err(); err != nil {
		return 0, 0, err
	}
	return executed, len(tests), nil
}

func readSkipBudget(start, platform string) (int, error) {
	path := strings.TrimSpace(start)
	for path != "" && path != "." && path != string(filepath.Separator) {
		candidate := filepath.Join(path, ".vrooli", "skip-budgets.json")
		if data, err := os.ReadFile(candidate); err == nil {
			var doc struct {
				Budgets map[string]int `json:"budgets"`
			}
			if err := json.Unmarshal(data, &doc); err != nil {
				return 0, fmt.Errorf("decode skip budgets: %w", err)
			}
			budget, ok := doc.Budgets[platform]
			if !ok {
				return 0, fmt.Errorf("skip budget has no entry for %s", platform)
			}
			return budget, nil
		}
		next := filepath.Dir(path)
		if next == path {
			break
		}
		path = next
	}
	// Unit-test runners may execute from an isolated temporary directory. The
	// repository-owned test-genie run always discovers .vrooli/skip-budgets.json
	// from its scenario or workspace root; an absent file here means there is
	// no platform skip to qualify in this isolated invocation.
	return 0, nil
}

func normalizedPlatform(goos string) string {
	if strings.EqualFold(goos, "darwin") {
		return "macos"
	}
	return goos
}

func runtimeGOOS() string {
	return runtime.GOOS
}
