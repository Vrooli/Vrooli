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
	"strconv"
	"strings"
	"sync"

	"connectrpc.com/connect"
	sourcejournal "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/journal"
	sourceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/journal/journal_v1connect"
)

type (
	Importer struct {
		journal           sourceconnect.JournalServiceClient
		claudeRoot        string
		adapters          map[string]AdapterDescriptor
		projectionTargets map[string]struct{}
		runs              *runStore
		mu                sync.Mutex
		active            map[string]string
	}
	ImportResult struct {
		Runtime                  string
		Seen, Imported, Existing int
		Sources                  []string
		Cursor                   string
		Bounded                  bool
	}
)

func NewImporter(s sourceconnect.JournalServiceClient, claudeRoot string, projectionTargets []string, databases ...*sql.DB) *Importer {
	home, _ := os.UserHomeDir()
	targets := make(map[string]struct{}, len(projectionTargets))
	for _, target := range projectionTargets {
		targets[normalizedAbsolutePath(target)] = struct{}{}
	}
	i := &Importer{journal: s, claudeRoot: claudeRoot, adapters: defaultAdapters(claudeRoot, home), projectionTargets: targets, active: make(map[string]string)}
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
	items, managedOnly, err := adapter.discover(i.projectionTargets)
	if err != nil {
		return result, err
	}
	if adapter.HarnessID == "swarm-manager-records" && !dryRun && i.runs != nil {
		return i.importSwarmBatch(ctx, adapter, items)
	}
	for _, item := range items {
		if err := i.importItem(ctx, adapter, item, dryRun, &result); err != nil {
			return result, err
		}
	}
	if result.Seen == 0 {
		// A store whose only content is this service's own projection is
		// correctly empty. Reporting it as a failure would turn the one-way
		// projection guarantee into a permanent red on the operator surface.
		if managedOnly {
			return result, nil
		}
		return result, fmt.Errorf("non-empty harness %q store yielded zero importable items", runtime)
	}
	return result, nil
}

const swarmImportBatchSize = 128

// importSwarmBatch makes the large append-only record tree converge across
// maintenance ticks. The cursor advances only after an item has been handed to
// source-ledger; a timeout before that checkpoint leaves the item eligible for
// the next pass, while source-ledger's content key makes a replay harmless.
func (i *Importer) importSwarmBatch(ctx context.Context, adapter AdapterDescriptor, items []sourceItem) (ImportResult, error) {
	result := ImportResult{Runtime: adapter.HarnessID, Bounded: true}
	root := strings.Join(adapter.Locations, ",")
	cursor, err := i.runs.cursor(ctx, adapter.HarnessID, root)
	if err != nil {
		return result, err
	}
	start := 0
	for start < len(items) && items[start].Path <= cursor {
		start++
	}
	if start == len(items) {
		if err := i.runs.setCursor(ctx, adapter.HarnessID, root, ""); err != nil {
			return result, err
		}
		return result, nil
	}
	end := start + swarmImportBatchSize
	if end > len(items) {
		end = len(items)
	}
	for _, item := range items[start:end] {
		if err := i.importItem(ctx, adapter, item, false, &result); err != nil {
			if ctx.Err() != nil {
				return result, nil
			}
			return result, err
		}
		if err := i.runs.setCursor(ctx, adapter.HarnessID, root, item.Path); err != nil {
			return result, err
		}
		result.Cursor = item.Path
	}
	if end == len(items) {
		if err := i.runs.setCursor(ctx, adapter.HarnessID, root, ""); err != nil {
			return result, err
		}
		result.Cursor = ""
	}
	return result, nil
}

// IsEmptyStoreError identifies an expected discovery result for a declared
// adapter. Dry-run callers report this as an honest zero-source observation;
// durable imports still surface it as a failed run so operators can see that
// no work was performed.
func IsEmptyStoreError(err error) bool {
	if err == nil {
		return false
	}
	message := err.Error()
	return strings.Contains(message, "store is not present") || strings.Contains(message, "yielded zero importable items")
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
	items, _, err := adapter.discover(i.projectionTargets)
	if err != nil {
		return ImportRun{}, false, err
	}
	run, err := i.runs.create(ctx, runtime, strings.Join(adapter.Locations, ","), len(items))
	if err != nil {
		return ImportRun{}, false, err
	}
	i.active[runtime] = run.ID
	// The durable run must survive the request that created it, but it should
	// retain any safe context values already established at the API boundary.
	// WithoutCancel is the explicit lifecycle hand-off; a fresh Background here
	// would silently discard that provenance and trips the goroutine-context
	// safety check.
	go i.run(context.WithoutCancel(ctx), adapter, run.ID, items)
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

func (i *Importer) run(ctx context.Context, adapter AdapterDescriptor, id string, items []sourceItem) {
	if err := i.runs.running(ctx, id); err != nil {
		return
	}
	result := ImportResult{Runtime: adapter.HarnessID}
	var failed int
	var lastError string
	type itemResult struct {
		result ImportResult
		path   string
		err    error
	}
	jobs := make(chan sourceItem)
	completed := make(chan itemResult, importWorkerCount())
	var workers sync.WaitGroup
	for range cap(completed) {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for item := range jobs {
				itemResult := ImportResult{Runtime: adapter.HarnessID}
				completed <- struct {
					result ImportResult
					path   string
					err    error
				}{result: itemResult, path: item.Path, err: i.importItem(ctx, adapter, item, false, &itemResult)}
			}
		}()
	}
	go func() {
		defer close(jobs)
		for _, item := range items {
			select {
			case jobs <- item:
			case <-ctx.Done():
				return
			}
		}
		workers.Wait()
		close(completed)
	}()
	for item := range completed {
		result.Seen += item.result.Seen
		result.Imported += item.result.Imported
		result.Existing += item.result.Existing
		if item.err != nil {
			failed++
			lastError = item.err.Error()
		}
		// A checkpoint is committed after every source, including a failure.
		if err := i.runs.progress(ctx, id, result.Seen, result.Imported, result.Existing, failed, item.path); err != nil {
			lastError = err.Error()
			failed++
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

// importWorkerCount bounds concurrent inference during a durable import. SQLite
// remains the serialization point for the final immutable append, while
// classification and the three derived embeddings can use the gateway in
// parallel. Operators may reduce this when sharing a constrained gateway.
func importWorkerCount() int {
	const defaultWorkers = 4
	raw := strings.TrimSpace(os.Getenv("VROOLI_MEMORY_IMPORT_CONCURRENCY"))
	if raw == "" {
		return defaultWorkers
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 {
		return defaultWorkers
	}
	if n > 16 {
		return 16
	}
	return n
}

func (i *Importer) adapter(runtime string) (AdapterDescriptor, error) {
	adapter, ok := i.adapters[runtime]
	if !ok {
		return AdapterDescriptor{}, fmt.Errorf("unsupported harness %q", runtime)
	}
	return adapter, nil
}

func (i *Importer) importItem(ctx context.Context, adapter AdapterDescriptor, item sourceItem, dryRun bool, result *ImportResult) error {
	body := strings.TrimSpace(stripManagedWakeBlock(item.Body))
	if body == "" {
		return nil
	}
	result.Seen++
	result.Sources = append(result.Sources, item.Path)
	if dryRun {
		return nil
	}
	key := importKey(adapter.HarnessID, item.Path, body)
	kind := "import"
	if adapter.HarnessID == "swarm-manager-records" {
		kind = "work-record"
	}
	request := &sourcejournal.AppendEntryRequest{Body: body, Kind: kind, Scope: "agent-memory", ImportProvenance: &sourcejournal.ImportProvenance{Runtime: adapter.HarnessID, SourceLocator: item.Path, ContentHash: key}}
	if kind == "work-record" {
		request.Trigger = "harness import"
		request.Approach = "import durable harness record into source-ledger"
		request.Evidence = item.Path
		request.Outcome = body
	}
	entry, err := i.journal.AppendEntry(ctx, connect.NewRequest(request))
	if err != nil {
		return fmt.Errorf("import %s: %w", item.Path, err)
	}
	if entry.Msg.GetExisting() {
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

// Capture appends a native harness write using the same content-addressed key
// as a later store sweep. A hook and import therefore converge on one entry.
func (i *Importer) Capture(ctx context.Context, runtime, path, body string) (*sourcejournal.Entry, error) {
	_, err := i.adapter(runtime)
	if err != nil {
		return nil, err
	}
	body = strings.TrimSpace(body)
	path = strings.TrimSpace(path)
	if body == "" {
		return nil, fmt.Errorf("capture requires content")
	}
	// Native memory tools often identify the destination implicitly and omit a
	// filesystem path. Use one stable logical locator so those writes still
	// reach the ledger and replaying the same body remains idempotent.
	if path == "" {
		path = "native-memory:" + runtime
	}
	resp, err := i.journal.AppendEntry(ctx, connect.NewRequest(&sourcejournal.AppendEntryRequest{Body: body, Kind: "capture", Scope: "agent-memory", ImportProvenance: &sourcejournal.ImportProvenance{Runtime: runtime, SourceLocator: path, ContentHash: importKey(runtime, path, body)}}))
	if err != nil {
		return nil, err
	}
	return resp.Msg.GetEntry(), nil
}
