package services
// DOC: docs/reference/api-endpoints.md#scripts

import (
	"bufio"
	"context"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"system-monitor-api/internal/apierrors"
)

// ScriptMeta holds parsed metadata from a script file header.
type ScriptMeta struct {
	ID          string
	Name        string
	Description string
	Category    string
	Author      string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Enabled     bool
}

// ScriptExecution holds the result of running a script.
type ScriptExecution struct {
	ScriptID        string
	ExecutionID     string
	Status          string // "completed" or "failed"
	StartedAt       time.Time
	CompletedAt     time.Time
	Stdout          string
	Stderr          string
	ExitCode        int
	TimedOut        bool
	DurationSeconds float64
}

// ScriptService discovers and executes investigation scripts from disk.
type ScriptService struct {
	scriptsDir string
	runner     CommandRunner
}

// NewScriptService creates a ScriptService that reads scripts from the given directory.
// An optional CommandRunner can be provided for testability; defaults to ExecCommandRunner.
func NewScriptService(scriptsDir string, runner ...CommandRunner) *ScriptService {
	r := CommandRunner(ExecCommandRunner{})
	if len(runner) > 0 && runner[0] != nil {
		r = runner[0]
	}
	return &ScriptService{scriptsDir: scriptsDir, runner: r}
}

// ListScripts returns metadata for all scripts in the scripts directory.
func (s *ScriptService) ListScripts() ([]ScriptMeta, error) {
	entries, err := os.ReadDir(s.scriptsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, apierrors.Internal("Unable to load investigation scripts", err)
	}

	var scripts []ScriptMeta
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sh") {
			continue
		}
		path := filepath.Join(s.scriptsDir, e.Name())
		meta, err := parseScriptHeader(path)
		if err != nil {
			continue // skip unparseable scripts
		}
		scripts = append(scripts, meta)
	}
	return scripts, nil
}

// GetScript returns metadata and file content for a specific script.
func (s *ScriptService) GetScript(id string) (ScriptMeta, string, error) {
	path, err := s.resolveScriptPath(id)
	if err != nil {
		return ScriptMeta{}, "", err
	}

	meta, err := parseScriptHeader(path)
	if err != nil {
		return ScriptMeta{}, "", apierrors.Internal("Script metadata could not be read", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return ScriptMeta{}, "", apierrors.Internal("Script content could not be read", err)
	}

	return meta, string(content), nil
}

// ExecuteScript runs a script with a timeout, capturing stdout/stderr/exit code.
// If contentOverride is non-empty, it is written to a temp file and executed instead.
func (s *ScriptService) ExecuteScript(ctx context.Context, id string, contentOverride string) (ScriptExecution, error) {
	execID := uuid.New().String()
	startedAt := time.Now()

	result := ScriptExecution{
		ScriptID:    id,
		ExecutionID: execID,
		StartedAt:   startedAt,
	}

	var scriptPath string
	var cleanup func()

	if contentOverride != "" {
		// Write content override to a temp file
		tmpFile, err := os.CreateTemp("", "script-exec-*.sh")
		if err != nil {
			return result, apierrors.Internal("Failed to prepare script for execution", err)
		}
		cleanup = func() { os.Remove(tmpFile.Name()) }

		if _, err := tmpFile.WriteString(contentOverride); err != nil {
			cleanup()
			return result, apierrors.Internal("Failed to prepare script for execution", err)
		}
		tmpFile.Close()
		if err := os.Chmod(tmpFile.Name(), 0o700); err != nil {
			cleanup()
			return result, apierrors.Internal("Failed to set script permissions", err)
		}
		scriptPath = tmpFile.Name()
	} else {
		resolved, err := s.resolveScriptPath(id)
		if err != nil {
			return result, err
		}
		scriptPath = resolved
	}
	if cleanup != nil {
		defer cleanup()
	}

	// Execute with 60s timeout
	execCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	stdout, stderr, exitCode, err := s.runner.Run(execCtx, "bash", []string{scriptPath}, s.scriptsDir)
	result.CompletedAt = time.Now()
	result.DurationSeconds = result.CompletedAt.Sub(startedAt).Seconds()
	result.Stdout = stdout
	result.Stderr = stderr

	if execCtx.Err() == context.DeadlineExceeded {
		result.TimedOut = true
		result.ExitCode = -1
		result.Status = "failed"
	} else if err != nil {
		result.ExitCode = exitCode
		result.Status = "failed"
	} else {
		result.ExitCode = 0
		result.Status = "completed"
	}

	return result, nil
}

// resolveScriptPath maps a script ID (filename stem) to its full path.
func (s *ScriptService) resolveScriptPath(id string) (string, error) {
	// The ID is the filename without .sh extension
	path := filepath.Join(s.scriptsDir, id+".sh")
	if _, err := os.Stat(path); err != nil {
		return "", apierrors.NotFound("script", id)
	}
	return path, nil
}

// parseScriptHeader reads structured header comments from a script file.
func parseScriptHeader(path string) (ScriptMeta, error) {
	f, err := os.Open(path)
	if err != nil {
		return ScriptMeta{}, err
	}
	defer f.Close()

	base := filepath.Base(path)
	id := strings.TrimSuffix(base, ".sh")

	meta := ScriptMeta{
		ID:      id,
		Name:    id,
		Enabled: true,
	}

	info, err := os.Stat(path)
	if err == nil {
		meta.UpdatedAt = info.ModTime()
		meta.CreatedAt = info.ModTime()
	}

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "#") {
			// Stop at first non-comment, non-empty line (allow shebang and blank lines)
			if strings.TrimSpace(line) != "" && !strings.HasPrefix(line, "#!/") {
				break
			}
			continue
		}

		line = strings.TrimPrefix(line, "#")
		line = strings.TrimSpace(line)

		if idx := strings.Index(line, ":"); idx > 0 {
			key := strings.TrimSpace(line[:idx])
			value := strings.TrimSpace(line[idx+1:])

			switch strings.ToUpper(key) {
			case "NAME":
				meta.Name = value
			case "DESCRIPTION":
				meta.Description = value
			case "CATEGORY":
				meta.Category = value
			case "AUTHOR":
				meta.Author = value
			case "CREATED":
				if t, err := time.Parse("2006-01-02", value); err == nil {
					meta.CreatedAt = t
				}
			case "LAST_MODIFIED":
				if t, err := time.Parse("2006-01-02", value); err == nil {
					meta.UpdatedAt = t
				}
			}
		}
	}

	return meta, nil
}
