// Package harness imports native coding-agent memory stores without changing
// their source files. It is deliberately content-addressed and stateless.
package harness

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"vrooli-memory/internal/journal"
)

type (
	Importer struct {
		journal    *journal.Service
		claudeRoot string
		runs       *runStore
		mu         sync.Mutex
		active     map[string]string
	}
	ImportResult struct {
		Runtime                  string
		Seen, Imported, Existing int
		Sources                  []string
	}
)

func NewImporter(s *journal.Service, claudeRoot string, databases ...*sql.DB) *Importer {
	i := &Importer{journal: s, claudeRoot: claudeRoot, active: make(map[string]string)}
	if len(databases) > 0 && databases[0] != nil {
		i.runs = newRunStore(databases[0])
	}
	return i
}

func (i *Importer) Import(ctx context.Context, runtime string, dryRun bool) (ImportResult, error) {
	if runtime != "claude-code" {
		return ImportResult{}, fmt.Errorf("unsupported harness %q", runtime)
	}
	result := ImportResult{Runtime: runtime}
	paths, err := i.sourcePaths()
	if err != nil {
		return result, err
	}
	for _, path := range paths {
		if err := i.importPath(ctx, runtime, path, dryRun, &result); err != nil {
			return result, err
		}
	}
	if result.Seen == 0 {
		return result, fmt.Errorf("non-empty import root %q yielded zero markdown items", i.claudeRoot)
	}
	return result, nil
}

// Start creates a durable import run and returns immediately. Repeated starts
// for a runtime join its active run, so retries cannot silently duplicate work.
func (i *Importer) Start(ctx context.Context, runtime string) (ImportRun, bool, error) {
	if runtime != "claude-code" {
		return ImportRun{}, false, fmt.Errorf("unsupported harness %q", runtime)
	}
	if i.runs == nil {
		return ImportRun{}, false, fmt.Errorf("durable import status store is not configured")
	}
	i.mu.Lock()
	defer i.mu.Unlock()
	if id := i.active[runtime]; id != "" {
		run, err := i.runs.get(ctx, id)
		return run, true, err
	}
	paths, err := i.sourcePaths()
	if err != nil {
		return ImportRun{}, false, err
	}
	run, err := i.runs.create(ctx, runtime, i.claudeRoot, len(paths))
	if err != nil {
		return ImportRun{}, false, err
	}
	i.active[runtime] = run.ID
	go i.run(runtime, run.ID, paths)
	return run, false, nil
}

func (i *Importer) Status(ctx context.Context, id, runtime string) (ImportRun, error) {
	if i.runs == nil {
		return ImportRun{}, fmt.Errorf("durable import status store is not configured")
	}
	if id != "" {
		return i.runs.get(ctx, id)
	}
	return i.runs.latest(ctx, runtime)
}

func (i *Importer) run(runtime, id string, paths []string) {
	ctx := context.Background()
	if err := i.runs.running(ctx, id); err != nil {
		return
	}
	result := ImportResult{Runtime: runtime}
	var failed int
	var lastError string
	for _, path := range paths {
		err := i.importPath(ctx, runtime, path, false, &result)
		if err != nil {
			failed++
			lastError = err.Error()
		}
		// A checkpoint is committed after every source, including a failure.
		if err := i.runs.progress(ctx, id, result.Seen, result.Imported, result.Existing, failed, path); err != nil {
			break
		}
	}
	status := ImportRunCompleted
	if failed > 0 {
		status = ImportRunCompletedWithErrors
	}
	if result.Seen == 0 {
		status = ImportRunFailed
		if lastError == "" {
			lastError = fmt.Sprintf("non-empty import root %q yielded zero markdown items", i.claudeRoot)
		}
	}
	_ = i.runs.finish(ctx, id, status, result.Seen, result.Imported, result.Existing, failed, lastError)
	i.mu.Lock()
	if i.active[runtime] == id {
		delete(i.active, runtime)
	}
	i.mu.Unlock()
}

func (i *Importer) sourcePaths() ([]string, error) {
	var paths []string
	err := filepath.WalkDir(i.claudeRoot, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() || !strings.HasSuffix(strings.ToLower(path), ".md") {
			return nil
		}
		paths = append(paths, path)
		return nil
	})
	sort.Strings(paths)
	return paths, err
}

func (i *Importer) importPath(ctx context.Context, runtime, path string, dryRun bool, result *ImportResult) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	body := strings.TrimSpace(string(b))
	if body == "" {
		return nil
	}
	result.Seen++
	result.Sources = append(result.Sources, path)
	if dryRun {
		return nil
	}
	key := importKey(runtime, path, body)
	if _, found, err := i.journal.FindByImportKey(ctx, key); err != nil {
		return err
	} else if found {
		result.Existing++
		return nil
	}
	entry, err := i.journal.Append(ctx, journal.Entry{Body: body, Kind: "import", ImportKey: key, Attribution: journal.Attribution{ActorKind: "harness-import", SourceRuntime: runtime}, Import: journal.ImportProvenance{Harness: runtime, Path: path, ImportedAt: time.Now().UTC()}})
	if err != nil {
		return fmt.Errorf("import %s: %w", path, err)
	}
	if entry.Existing {
		result.Existing++
	} else {
		result.Imported++
	}
	return nil
}

func importKey(runtime, path, body string) string {
	h := sha256.Sum256([]byte(runtime + "\x00" + path + "\x00" + strings.Join(strings.Fields(body), " ")))
	return hex.EncodeToString(h[:])
}
