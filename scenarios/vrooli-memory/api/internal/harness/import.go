// Package harness imports native coding-agent memory stores without changing
// their source files. It is deliberately content-addressed and stateless.
package harness

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"vrooli-memory/internal/journal"
)

type (
	Importer struct {
		journal    *journal.Service
		claudeRoot string
		adapters   map[string]AdapterDescriptor
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
	home, _ := os.UserHomeDir()
	i := &Importer{journal: s, claudeRoot: claudeRoot, adapters: defaultAdapters(claudeRoot, home), active: make(map[string]string)}
	if len(databases) > 0 && databases[0] != nil {
		i.runs = newRunStore(databases[0])
	}
	return i
}

func (i *Importer) Import(ctx context.Context, runtime string, dryRun bool) (ImportResult, error) {
	result := ImportResult{Runtime: runtime}
	adapter, err := i.adapter(runtime)
	if err != nil {
		return result, err
	}
	items, err := adapter.discover()
	if err != nil {
		return result, err
	}
	for _, item := range items {
		if err := i.importItem(ctx, adapter, item, dryRun, &result); err != nil {
			return result, err
		}
	}
	if result.Seen == 0 {
		return result, fmt.Errorf("non-empty harness %q store yielded zero importable items", runtime)
	}
	return result, nil
}

// Start creates a durable import run and returns immediately. Repeated starts
// for a runtime join its active run, so retries cannot silently duplicate work.
func (i *Importer) Start(ctx context.Context, runtime string) (ImportRun, bool, error) {
	if i.runs == nil {
		return ImportRun{}, false, fmt.Errorf("durable import status store is not configured")
	}
	i.mu.Lock()
	defer i.mu.Unlock()
	if id := i.active[runtime]; id != "" {
		run, err := i.runs.get(ctx, id)
		return run, true, err
	}
	adapter, err := i.adapter(runtime)
	if err != nil {
		return ImportRun{}, false, err
	}
	items, err := adapter.discover()
	if err != nil {
		return ImportRun{}, false, err
	}
	run, err := i.runs.create(ctx, runtime, strings.Join(adapter.Locations, ","), len(items))
	if err != nil {
		return ImportRun{}, false, err
	}
	i.active[runtime] = run.ID
	go i.run(adapter, run.ID, items)
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

func (i *Importer) run(adapter AdapterDescriptor, id string, items []sourceItem) {
	ctx := context.Background()
	if err := i.runs.running(ctx, id); err != nil {
		return
	}
	result := ImportResult{Runtime: adapter.HarnessID}
	var failed int
	var lastError string
	for _, item := range items {
		err := i.importItem(ctx, adapter, item, false, &result)
		if err != nil {
			failed++
			lastError = err.Error()
		}
		// A checkpoint is committed after every source, including a failure.
		if err := i.runs.progress(ctx, id, result.Seen, result.Imported, result.Existing, failed, item.Path); err != nil {
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
			lastError = fmt.Sprintf("non-empty harness %q store yielded zero importable items", adapter.HarnessID)
		}
	}
	_ = i.runs.finish(ctx, id, status, result.Seen, result.Imported, result.Existing, failed, lastError)
	i.mu.Lock()
	if i.active[adapter.HarnessID] == id {
		delete(i.active, adapter.HarnessID)
	}
	i.mu.Unlock()
}

func (i *Importer) adapter(runtime string) (AdapterDescriptor, error) {
	adapter, ok := i.adapters[runtime]
	if !ok {
		return AdapterDescriptor{}, fmt.Errorf("unsupported harness %q", runtime)
	}
	return adapter, nil
}

func (i *Importer) importItem(ctx context.Context, adapter AdapterDescriptor, item sourceItem, dryRun bool, result *ImportResult) error {
	body := strings.TrimSpace(item.Body)
	if body == "" {
		return nil
	}
	result.Seen++
	result.Sources = append(result.Sources, item.Path)
	if dryRun {
		return nil
	}
	key := importKey(adapter.HarnessID, item.Path, body)
	if entry, found, err := i.journal.FindByImportKey(ctx, key); err != nil {
		return err
	} else if found {
		if entry.Import.Harness == "" || entry.Import.Path == "" || entry.Import.ImportedAt.IsZero() {
			if err := i.journal.RepairImportProvenance(ctx, entry.ID, journal.ImportProvenance{Harness: adapter.HarnessID, Path: item.Path}); err != nil {
				return fmt.Errorf("repair import provenance for %s: %w", item.Path, err)
			}
		}
		result.Existing++
		return nil
	}
	entry, err := i.journal.Append(ctx, journal.Entry{Body: body, Kind: "import", ImportKey: key, Attribution: journal.Attribution{ActorKind: "harness-import", SourceRuntime: adapter.Provenance.SourceRuntime}, Import: journal.ImportProvenance{Harness: adapter.HarnessID, Path: item.Path, ImportedAt: time.Now().UTC()}})
	if err != nil {
		return fmt.Errorf("import %s: %w", item.Path, err)
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
