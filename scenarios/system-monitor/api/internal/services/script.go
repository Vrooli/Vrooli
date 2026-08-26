package services

// DOC: docs/reference/api-endpoints.md#scripts

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/storage"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/apierrors"
	catalogpkg "github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/investigations"
)

// ScriptMeta holds parsed metadata from a script file header.
type ScriptMeta struct {
	ID            string
	Name          string
	Description   string
	Category      string
	Author        string
	CreatedAt     time.Time
	UpdatedAt     time.Time
	Enabled       bool
	ExecutionMode string
	RequiredTools []string
	SkipReason    string
	Query         string
	Platforms     []string
	Source        string
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
	ExecutionMode   string
	SkipReason      string
	Platforms       []string
	Source          string
}

// ScriptService discovers and executes investigation scripts from disk.
type ScriptService struct {
	scriptsDir string
	runner     CommandRunner
	native     NativeInvestigationRunner
	policyPath string
	catalog    *catalogpkg.Catalog
	stateDir   string
	runRepo    catalogpkg.Repository
}

// NewCatalogScriptService creates the production script service. The catalog
// is already loaded from the embedded product catalog plus the operator
// overlay, so no repository path is consulted at runtime.
func NewCatalogScriptService(catalog catalogpkg.Catalog, stateDir string, runner ...CommandRunner) *ScriptService {
	svc := NewScriptService("", runner...)
	svc.catalog = &catalog
	svc.stateDir = stateDir
	return svc
}

// NewScriptService creates a ScriptService that reads scripts from the given directory.
// An optional CommandRunner can be provided for testability; defaults to ExecCommandRunner.
func NewScriptService(scriptsDir string, runner ...CommandRunner) *ScriptService {
	r := CommandRunner(ExecCommandRunner{})
	if len(runner) > 0 && runner[0] != nil {
		r = runner[0]
	}
	return &ScriptService{
		scriptsDir: scriptsDir,
		runner:     r,
		policyPath: filepath.Join(filepath.Dir(scriptsDir), "execution-policy.json"),
	}
}

// SetNativeRunner installs the typed-query execution seam. It is separate from
// the command runner so tests can prove native investigations do not fork.
func (s *ScriptService) SetNativeRunner(r NativeInvestigationRunner) {
	s.native = r
}

func (s *ScriptService) SetRunRepository(repo catalogpkg.Repository) { s.runRepo = repo }

// ListScripts returns metadata for all scripts in the scripts directory.
func (s *ScriptService) ListScripts() ([]ScriptMeta, error) {
	if s.catalog != nil {
		entries := s.catalog.Entries()
		result := make([]ScriptMeta, 0, len(entries))
		for _, entry := range entries {
			meta := ScriptMeta{ID: entry.ID, Name: entry.Name, Description: entry.Description, Category: entry.Category, Enabled: entry.Enabled, ExecutionMode: string(entry.Mode), RequiredTools: append([]string(nil), entry.RequiredTools...), Query: entry.Query, Platforms: append([]string(nil), entry.Platforms...), Source: entry.Source, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
			if !entry.SupportsCurrentPlatform() {
				meta.SkipReason = "investigation is not available on " + runtime.GOOS
			}
			if meta.SkipReason == "" {
				meta.SkipReason = missingTools(entry.RequiredTools)
			}
			result = append(result, meta)
		}
		return result, nil
	}
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
		s.applyExecutionPolicy(&meta)
		scripts = append(scripts, meta)
	}
	return scripts, nil
}

// GetScript returns metadata and file content for a specific script.
func (s *ScriptService) GetScript(id string) (ScriptMeta, string, error) {
	if s.catalog != nil {
		entry, ok := s.catalog.Get(id)
		if !ok {
			return ScriptMeta{}, "", apierrors.NotFound("script", id)
		}
		meta := catalogEntryMeta(entry)
		if content, ok := s.catalog.Shell(id); ok {
			return meta, string(content), nil
		}
		return meta, "", nil
	}
	path, err := s.resolveScriptPath(id)
	if err != nil {
		return ScriptMeta{}, "", err
	}

	meta, err := parseScriptHeader(path)
	if err != nil {
		return ScriptMeta{}, "", apierrors.Internal("Script metadata could not be read", err)
	}
	s.applyExecutionPolicy(&meta)

	content, err := os.ReadFile(path)
	if err != nil {
		return ScriptMeta{}, "", apierrors.Internal("Script content could not be read", err)
	}

	return meta, string(content), nil
}

// UpdateScript persists the complete source for an existing script and returns
// the parsed metadata and source that will be served to subsequent readers.
func (s *ScriptService) UpdateScript(id string, content string) (ScriptMeta, string, error) {
	if s.catalog != nil {
		return ScriptMeta{}, "", apierrors.Validation("id", "built-in catalog entries are immutable; create an operator entry in the state overlay")
	}
	path, err := s.resolveScriptPath(id)
	if err != nil {
		return ScriptMeta{}, "", err
	}
	if strings.TrimSpace(content) == "" {
		return ScriptMeta{}, "", apierrors.Validation("content", "Script content is required")
	}
	if err := storage.WriteFileAtomic(path, []byte(content), 0o644); err != nil {
		return ScriptMeta{}, "", apierrors.Internal("Script content could not be saved", err)
	}
	return s.GetScript(id)
}

// ExecuteScript runs a script with a timeout, capturing stdout/stderr/exit code.
// If contentOverride is non-empty, it is written to a temp file and executed instead.
func (s *ScriptService) ExecuteScript(ctx context.Context, id string, contentOverride string) (ScriptExecution, error) {
	if s.catalog != nil {
		return s.executeCatalogScript(ctx, id, contentOverride)
	}
	execID := uuid.New().String()
	startedAt := time.Now()

	result := ScriptExecution{
		ScriptID:    id,
		ExecutionID: execID,
		StartedAt:   startedAt,
	}
	meta := ScriptMeta{ID: id, ExecutionMode: "shell"}
	s.applyExecutionPolicy(&meta)
	result.ExecutionMode = meta.ExecutionMode
	if meta.SkipReason != "" {
		result.Status = "skipped"
		result.SkipReason = meta.SkipReason
		result.Stderr = meta.SkipReason
		result.CompletedAt = time.Now()
		result.DurationSeconds = result.CompletedAt.Sub(startedAt).Seconds()
		return result, nil
	}
	if meta.ExecutionMode == "native" {
		if s.native == nil {
			result.Status = "skipped"
			result.SkipReason = "native query runner is unavailable"
			result.Stderr = result.SkipReason
			result.CompletedAt = time.Now()
			return result, nil
		}
		stdout, err := s.native.RunNative(ctx, meta.Query)
		result.CompletedAt = time.Now()
		result.DurationSeconds = result.CompletedAt.Sub(startedAt).Seconds()
		result.Stdout = string(stdout)
		if err != nil {
			result.Status = "failed"
			result.Stderr = err.Error()
			result.ExitCode = 1
			return result, nil
		}
		result.Status = "completed"
		result.ExitCode = 0
		return result, nil
	}

	var scriptPath string
	var cleanup func()

	if contentOverride != "" {
		// Write content override to a temp file
		tmpFile, err := os.CreateTemp("", "script-exec-*.sh")
		if err != nil {
			return result, apierrors.Internal("Failed to prepare script for execution", err)
		}
		cleanup = func() { _ = storage.RemoveFile(tmpFile.Name()) }

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

func catalogEntryMeta(entry catalogpkg.Entry) ScriptMeta {
	now := time.Now().UTC()
	meta := ScriptMeta{ID: entry.ID, Name: entry.Name, Description: entry.Description, Category: entry.Category, Enabled: entry.Enabled, ExecutionMode: string(entry.Mode), RequiredTools: append([]string(nil), entry.RequiredTools...), Query: entry.Query, Platforms: append([]string(nil), entry.Platforms...), Source: entry.Source, CreatedAt: now, UpdatedAt: now}
	if !entry.SupportsCurrentPlatform() {
		meta.SkipReason = "investigation is not available on " + runtime.GOOS
	}
	if meta.SkipReason == "" {
		meta.SkipReason = missingTools(entry.RequiredTools)
	}
	return meta
}

func missingTools(tools []string) string {
	missing := make([]string, 0)
	for _, tool := range tools {
		if _, err := exec.LookPath(tool); err != nil {
			missing = append(missing, tool)
		}
	}
	if len(missing) == 0 {
		return ""
	}
	return "required tools unavailable: " + strings.Join(missing, ", ")
}

func (s *ScriptService) executeCatalogScript(ctx context.Context, id, contentOverride string) (result ScriptExecution, err error) {
	entry, ok := s.catalog.Get(id)
	if !ok {
		return ScriptExecution{}, apierrors.NotFound("script", id)
	}
	meta := catalogEntryMeta(entry)
	started := time.Now().UTC()
	result = ScriptExecution{ScriptID: id, ExecutionID: uuid.NewString(), StartedAt: started, ExecutionMode: string(entry.Mode), SkipReason: meta.SkipReason}
	defer func() {
		if s.runRepo == nil {
			return
		}
		run := catalogpkg.Run{ID: result.ExecutionID, EntryID: result.ScriptID, ExecutionMode: result.ExecutionMode, Status: result.Status, SkipReason: result.SkipReason, ExitCode: result.ExitCode, TimedOut: result.TimedOut, StartedAt: result.StartedAt, CompletedAt: result.CompletedAt, DurationSeconds: result.DurationSeconds, HostOS: runtime.GOOS, HostArch: runtime.GOARCH, ResultJSON: result.Stdout, StderrTail: result.Stderr}
		if saveErr := s.runRepo.SaveRun(ctx, run); saveErr != nil && err == nil {
			err = fmt.Errorf("persist investigation run: %w", saveErr)
		}
	}()
	if meta.SkipReason != "" {
		result.Status, result.Stderr, result.CompletedAt = "skipped", meta.SkipReason, time.Now().UTC()
		result.DurationSeconds = result.CompletedAt.Sub(started).Seconds()
		return result, nil
	}
	if entry.Mode == catalogpkg.ModeNative {
		if s.native == nil {
			result.Status, result.SkipReason = "skipped", "native query runner is unavailable"
			result.Stderr = result.SkipReason
			result.CompletedAt = time.Now().UTC()
			return result, nil
		}
		output, err := s.native.RunNative(ctx, entry.Query)
		result.CompletedAt, result.Stdout = time.Now().UTC(), string(output)
		result.DurationSeconds = result.CompletedAt.Sub(started).Seconds()
		if err != nil {
			result.Status, result.Stderr, result.ExitCode = "failed", err.Error(), 1
			return result, nil
		}
		if !json.Valid(output) {
			result.Status, result.Stderr, result.ExitCode = "failed", "native investigation returned invalid JSON", 1
			return result, nil
		}
		result.Status = "completed"
		return result, nil
	}

	content := contentOverride
	if content == "" {
		var exists bool
		contentBytes, exists := s.catalog.Shell(id)
		if !exists {
			return ScriptExecution{}, apierrors.Internal("embedded shell investigation is missing", fmt.Errorf("%s", id))
		}
		content = string(contentBytes)
	}
	tmp, err := os.CreateTemp("", "system-monitor-investigation-*.sh")
	if err != nil {
		return ScriptExecution{}, apierrors.Internal("prepare shell investigation", err)
	}
	tmpPath := tmp.Name()
	defer func() { _ = storage.RemoveFile(tmpPath) }()
	if _, err = tmp.WriteString(content); err != nil {
		_ = tmp.Close()
		return ScriptExecution{}, apierrors.Internal("write shell investigation", err)
	}
	if err = tmp.Close(); err != nil {
		return ScriptExecution{}, apierrors.Internal("close shell investigation", err)
	}
	_ = os.Chmod(tmpPath, 0o700)
	execCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	stdout, stderr, exitCode, runErr := s.runner.Run(execCtx, "bash", []string{tmpPath}, filepath.Dir(tmpPath))
	result.CompletedAt, result.Stdout, result.Stderr, result.ExitCode = time.Now().UTC(), stdout, stderr, exitCode
	result.DurationSeconds = result.CompletedAt.Sub(started).Seconds()
	if execCtx.Err() == context.DeadlineExceeded {
		result.Status, result.TimedOut, result.ExitCode = "failed", true, -1
		return result, nil
	}
	if runErr != nil {
		result.Status = "failed"
		return result, nil
	}
	if !json.Valid([]byte(strings.TrimSpace(stdout))) {
		result.Status, result.Stderr, result.ExitCode = "failed", "shell investigation returned stdout that is not one JSON document", 1
		return result, nil
	}
	result.Status = "completed"
	return result, nil
}

type scriptExecutionPolicy struct {
	Mode          string   `json:"mode"`
	Query         string   `json:"query"`
	RequiredTools []string `json:"required_tools"`
}

type scriptExecutionPolicyFile struct {
	Entries map[string]scriptExecutionPolicy `json:"entries"`
}

func (s *ScriptService) applyExecutionPolicy(meta *ScriptMeta) {
	if meta.ExecutionMode == "" {
		meta.ExecutionMode = "shell"
	}
	data, err := os.ReadFile(s.policyPath)
	if err != nil {
		return
	}
	var file scriptExecutionPolicyFile
	if json.Unmarshal(data, &file) != nil {
		return
	}
	policy, ok := file.Entries[meta.ID]
	if !ok {
		return
	}
	if policy.Mode != "" {
		meta.ExecutionMode = policy.Mode
	}
	meta.RequiredTools = append([]string(nil), policy.RequiredTools...)
	meta.Query = policy.Query
	if meta.ExecutionMode == "native" && meta.Query == "" {
		meta.SkipReason = "native investigation has no declared query"
	}
	missing := make([]string, 0)
	for _, tool := range meta.RequiredTools {
		if _, err := exec.LookPath(tool); err != nil {
			missing = append(missing, tool)
		}
	}
	if len(missing) > 0 {
		meta.SkipReason = "required tools unavailable: " + strings.Join(missing, ", ")
	}
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
