// Package tailer contains the shared lifecycle and cursor machinery for
// append-only agent transcript adapters. Source implementations own native
// discovery, record/document decoding, and agent metadata extraction.
package tailer

import (
	"bufio"
	"context"
	"errors"
	"github.com/vrooli/cli-core/agenttranscript"
	"io"
	"log"
	"os"
	"strconv"
	"sync"
	"time"
)

// FileRef identifies one append-only transcript and its owning web-console
// session. SourceKey is intentionally the path by default, but sources may
// use a stable logical key when a file can move.
type FileRef struct {
	Path      string
	SessionID string
	SourceKey string
}

// Event is the normalized output of a source decoder. Commit controls the
// durable cursor: a source can consume a complete line while holding the
// checkpoint at the last complete turn boundary.
type Event struct {
	// Source optionally refines the capture classification; checkpoints remain
	// owned by the engine's configured source. Dispatch validates refinements.
	Source     string
	Role       string
	Text       string
	OccurredAt time.Time
	// Class is what the decoder knows natively about the row (kind, author,
	// agent thread). Nil lets the store classify the row from its text.
	Class   *agenttranscript.Classification
	Commit  bool
	Ignored bool
}

// Checkpoint is an opaque source cursor. Append-only JSONL sources normally
// store a decimal byte offset; reconciliation sources may store another
// source-defined representation.
type Checkpoint struct {
	Source    string
	SourceKey string
	SessionID string
	Cursor    string
	UpdatedAt time.Time
}

// CheckpointStore is implemented by the web-console durable transcript
// cursor store and by small in-memory stores in adapter tests.
type CheckpointStore interface {
	Get(context.Context, string, string) (Checkpoint, bool, error)
	Save(context.Context, Checkpoint) error
}

// Source supplies the source-specific part of a transcript adapter.
type Source interface {
	DiscoverFiles(context.Context) ([]FileRef, error)
	DecodeLine(path string, line []byte) ([]Event, error)
	CaptureAgentInfo(path, sessionID string)
}

// PositionedLineSource receives the byte offset of a complete record in its
// source file. Adapters use it when identical native records at different
// positions must retain distinct stable identities.
type PositionedLineSource interface {
	DecodeLineAt(path string, line []byte, byteOffset int64) ([]Event, error)
}

// EOFRecordSource marks formats whose files contain one complete record with
// no trailing newline. Sources must return false for an incomplete partial
// write so the tailer rewinds and retries from the record boundary.
type EOFRecordSource interface {
	CompleteEOFLine(path string, line []byte) bool
}

// DocumentSource handles native archives stored as one complete structured
// document rather than JSONL records. A false complete result leaves the
// cursor unchanged so a writer's partial document can be retried.
type DocumentSource interface {
	HandlesDocument(path string) bool
	DecodeDocument(path string, document []byte) (events []Event, complete bool, err error)
}

// SnapshotDocumentSource adapts a mutable, non-file transcript store such as
// a SQLite database. Version must change when the logical snapshot changes;
// ReadSnapshot returns one consistent serialized view. The ordinary source
// path remains unchanged for immutable native document files.
type SnapshotDocumentSource interface {
	UsesSnapshot(path string) bool
	SnapshotVersion(path string) (string, error)
	ReadSnapshot(path string) ([]byte, string, error)
}

// Config controls one engine. Zero intervals use safe defaults.
type Config struct {
	Name         string
	Source       Source
	Checkpoints  CheckpointStore
	PollInterval time.Duration
	TailInterval time.Duration
	StaleTimeout time.Duration
	// StopUndiscovered makes each successful discovery result authoritative
	// for reader lifetime. Use only for sources that return a complete snapshot.
	// Discovery errors retain existing readers; checkpoints are never removed.
	StopUndiscovered bool
	// Dispatch returns whether a durable projection accepted the event. A
	// rejected event keeps the source cursor behind so a later scan can retry it.
	Dispatch func(Event, string) bool
	Logger   *log.Logger
}

// Engine owns discovery, watcher lifecycle, byte-offset reads, and durable
// cursor updates for one transcript source.
type Engine struct {
	name             string
	source           Source
	checkpoints      CheckpointStore
	pollInterval     time.Duration
	tailInterval     time.Duration
	staleTimeout     time.Duration
	stopUndiscovered bool
	dispatch         func(Event, string) bool
	logger           *log.Logger

	stopOnce  sync.Once
	stopCh    chan struct{}
	pollWG    sync.WaitGroup
	tailWG    sync.WaitGroup
	captureWG sync.WaitGroup
	mu        sync.Mutex
	scanMu    sync.Mutex
	watchers  map[string]*watcher
	decodeMu  sync.Mutex
}

type watcher struct {
	refresh chan chan struct{}
	done    chan struct{}
	stopCh  chan struct{}
}

// New constructs a shared transcript engine.
func New(cfg Config) *Engine {
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = time.Second
	}
	if cfg.TailInterval <= 0 {
		cfg.TailInterval = 500 * time.Millisecond
	}
	if cfg.StaleTimeout <= 0 {
		cfg.StaleTimeout = time.Hour
	}
	if cfg.Name == "" {
		cfg.Name = "transcript"
	}
	if cfg.Logger == nil {
		cfg.Logger = log.Default()
	}
	return &Engine{
		name:             cfg.Name,
		source:           cfg.Source,
		checkpoints:      cfg.Checkpoints,
		pollInterval:     cfg.PollInterval,
		tailInterval:     cfg.TailInterval,
		staleTimeout:     cfg.StaleTimeout,
		stopUndiscovered: cfg.StopUndiscovered,
		dispatch:         cfg.Dispatch,
		logger:           cfg.Logger,
		stopCh:           make(chan struct{}),
		watchers:         make(map[string]*watcher),
	}
}

// Start starts the source poller. A nil source is a no-op that still has
// well-defined shutdown semantics, which is useful for platform stubs.
func (e *Engine) Start() {
	if e == nil || e.source == nil {
		return
	}
	e.pollWG.Add(1)
	go func() {
		defer e.pollWG.Done()
		// Start the initial backfill promptly; the interval throttles recurring
		// discovery, not startup visibility of already-present history.
		e.Scan(context.Background())
		ticker := time.NewTicker(e.pollInterval)
		defer ticker.Stop()
		for {
			select {
			case <-e.stopCh:
				return
			case <-ticker.C:
				e.Scan(context.Background())
			}
		}
	}()
}

// Stop prevents new watchers and waits for all active file readers.
func (e *Engine) Stop() {
	if e == nil {
		return
	}
	e.stopOnce.Do(func() { close(e.stopCh) })
	e.pollWG.Wait()
	e.scanMu.Lock()
	defer e.scanMu.Unlock()
	e.tailWG.Wait()
	e.captureWG.Wait()
}

// Scan runs one discovery pass immediately. It is safe to use in tests and
// in hosts that trigger discovery from an external scheduler.
func (e *Engine) Scan(ctx context.Context) {
	if e == nil || e.source == nil {
		return
	}
	e.scanMu.Lock()
	defer e.scanMu.Unlock()
	select {
	case <-e.stopCh:
		return
	default:
	}
	files, err := e.source.DiscoverFiles(ctx)
	if err != nil {
		e.printf("discover files: %v", err)
		return
	}
	if e.stopUndiscovered {
		seen := make(map[string]bool, len(files))
		for _, ref := range files {
			if ref.Path != "" && ref.SessionID != "" {
				key := ref.SourceKey
				if key == "" {
					key = ref.Path
				}
				seen[key] = true
			}
		}
		var stopped []*watcher
		e.mu.Lock()
		for key, current := range e.watchers {
			if !seen[key] {
				delete(e.watchers, key)
				close(current.stopCh)
				stopped = append(stopped, current)
			}
		}
		e.mu.Unlock()
		for _, current := range stopped {
			<-current.done
		}
	}
	for _, ref := range files {
		if ref.Path == "" || ref.SessionID == "" {
			continue
		}
		key := ref.SourceKey
		if key == "" {
			key = ref.Path
		}
		e.mu.Lock()
		current, known := e.watchers[key]
		if !known {
			current = &watcher{refresh: make(chan chan struct{}), done: make(chan struct{}), stopCh: make(chan struct{})}
			e.watchers[key] = current
			e.tailWG.Add(1)
		}
		e.mu.Unlock()
		if known {
			ack := make(chan struct{})
			select {
			case current.refresh <- ack:
			case <-current.done:
			case <-e.stopCh:
			}
			select {
			case <-ack:
			case <-current.done:
			case <-e.stopCh:
			}
			continue
		}
		ready := make(chan struct{})
		e.captureWG.Add(1)
		go func() {
			defer e.captureWG.Done()
			e.capture(ref)
		}()
		go e.tail(ref, key, current, ready)
		// The first pass is bounded by one file open, seek, and currently
		// available append-only content. Waiting here makes an explicit Scan
		// deterministic for callers that use it as a one-shot reconciliation;
		// the normal poller still returns immediately after each file's initial
		// pass and never waits for future appends.
		<-ready
	}
}

func (e *Engine) capture(ref FileRef) {
	defer func() {
		if recovered := recover(); recovered != nil {
			e.printf("capture %s panicked: %v", ref.Path, recovered)
		}
	}()
	e.source.CaptureAgentInfo(ref.Path, ref.SessionID)
}

func (e *Engine) tail(ref FileRef, key string, current *watcher, ready chan struct{}) {
	if source, ok := e.source.(DocumentSource); ok && source.HandlesDocument(ref.Path) {
		e.tailDocument(ref, key, current, ready, source)
		return
	}
	defer e.tailWG.Done()
	readySignaled := false
	signalReady := func() {
		if ready != nil && !readySignaled {
			close(ready)
			readySignaled = true
		}
	}
	if ready != nil {
		defer signalReady()
	}
	defer close(current.done)
	defer func() {
		e.mu.Lock()
		if e.watchers[key] == current {
			delete(e.watchers, key)
		}
		e.mu.Unlock()
	}()

	f, err := os.Open(ref.Path)
	if err != nil {
		e.printf("open %s: %v", ref.Path, err)
		return
	}
	defer f.Close()

	offset := e.load(key)
	if stat, statErr := f.Stat(); statErr == nil && offset > stat.Size() {
		offset = 0
	}
	if _, err := f.Seek(offset, io.SeekStart); err != nil {
		e.printf("seek %s: %v", ref.Path, err)
		return
	}

	reader := bufio.NewReader(f)
	currentOffset := offset
	process := func() bool {
		processed := false
		for {
			lineStart := currentOffset
			line, readErr := reader.ReadBytes('\n')
			completeEOFRecord := false
			if readErr != nil {
				if !errors.Is(readErr, io.EOF) {
					e.printf("read %s: %v", ref.Path, readErr)
					return processed
				}
				if len(line) == 0 {
					return processed
				}
				if source, ok := e.source.(EOFRecordSource); ok && source.CompleteEOFLine(ref.Path, line) {
					completeEOFRecord = true
				} else {
					// ReadBytes consumed the partial suffix from its buffer. Rewind
					// so a later append can complete and decode the same record.
					if _, seekErr := f.Seek(lineStart, io.SeekStart); seekErr != nil {
						e.printf("rewind %s after incomplete record: %v", ref.Path, seekErr)
						return processed
					}
					reader = bufio.NewReader(f)
					return processed
				}
			}
			processed = true
			currentOffset += int64(len(line))
			e.decodeMu.Lock()
			var events []Event
			var decodeErr error
			if positioned, ok := e.source.(PositionedLineSource); ok {
				events, decodeErr = positioned.DecodeLineAt(ref.Path, line, lineStart)
			} else {
				events, decodeErr = e.source.DecodeLine(ref.Path, line)
			}
			e.decodeMu.Unlock()
			if decodeErr != nil {
				e.printf("decode %s: %v", ref.Path, decodeErr)
				// Do not let a later valid record move the durable cursor past
				// this undecodable record. Rewind and retry on the next poll so
				// transient source/catalog changes can recover without data loss.
				if _, seekErr := f.Seek(lineStart, io.SeekStart); seekErr != nil {
					e.printf("rewind %s after decode failure: %v", ref.Path, seekErr)
					return processed
				}
				reader = bufio.NewReader(f)
				currentOffset = lineStart
				return processed
			}
			commit := len(events) == 0
			projectionOK := true
			for _, event := range events {
				if event.Commit {
					commit = true
				}
				if !event.Ignored && e.dispatch != nil && event.Text != "" {
					if !e.dispatch(event, ref.SessionID) {
						projectionOK = false
					}
				}
			}
			if commit && projectionOK {
				if !e.save(key, ref.SessionID, currentOffset) {
					if _, seekErr := f.Seek(lineStart, io.SeekStart); seekErr != nil {
						e.printf("rewind %s after checkpoint failure: %v", ref.Path, seekErr)
						return processed
					}
					reader = bufio.NewReader(f)
					currentOffset = lineStart
					return processed
				}
			} else if !projectionOK {
				// Dispatch consumed the decoded line but did not durably project
				// it. Rewind both the file and buffered reader so a refresh can
				// retry the same line without requiring a process restart.
				if _, seekErr := f.Seek(lineStart, io.SeekStart); seekErr != nil {
					e.printf("rewind %s after dispatch failure: %v", ref.Path, seekErr)
					return processed
				}
				reader = bufio.NewReader(f)
				currentOffset = lineStart
				return processed
			}
			if completeEOFRecord {
				return processed
			}
		}
	}
	process()
	signalReady()

	ticker := time.NewTicker(e.tailInterval)
	defer ticker.Stop()
	stale := time.NewTimer(e.staleTimeout)
	defer stale.Stop()
	for {
		select {
		case <-e.stopCh:
			return
		case <-current.stopCh:
			return
		case ack := <-current.refresh:
			process()
			close(ack)
		case <-ticker.C:
			if process() {
				if !stale.Stop() {
					select {
					case <-stale.C:
					default:
					}
				}
				stale.Reset(e.staleTimeout)
			}
		case <-stale.C:
			if _, statErr := os.Stat(ref.Path); statErr != nil {
				return
			}
			stale.Reset(e.staleTimeout)
		}
	}
}

func (e *Engine) tailDocument(ref FileRef, key string, current *watcher, ready chan struct{}, source DocumentSource) {
	defer e.tailWG.Done()
	readySignaled := false
	signalReady := func() {
		if ready != nil && !readySignaled {
			close(ready)
			readySignaled = true
		}
	}
	if ready != nil {
		defer signalReady()
	}
	defer close(current.done)
	defer func() {
		e.mu.Lock()
		if e.watchers[key] == current {
			delete(e.watchers, key)
		}
		e.mu.Unlock()
	}()
	var cursor int64
	var checkpointVersion string
	if e.checkpoints != nil {
		cp, ok, err := e.checkpoints.Get(context.Background(), e.name, key)
		if err == nil && ok {
			checkpointVersion = cp.Cursor
			cursor, _ = strconv.ParseInt(cp.Cursor, 10, 64)
		}
	}
	snapshotSource, supportsSnapshots := source.(SnapshotDocumentSource)
	snapshots := supportsSnapshots && snapshotSource.UsesSnapshot(ref.Path)
	process := func() bool {
		var document []byte
		version := ""
		if snapshots {
			before, err := snapshotSource.SnapshotVersion(ref.Path)
			if err != nil || before == "" || before == checkpointVersion {
				return false
			}
			document, version, err = snapshotSource.ReadSnapshot(ref.Path)
			if err != nil || version == "" || version != before {
				return false
			}
		} else {
			before, err := os.Stat(ref.Path)
			if err != nil || !before.Mode().IsRegular() || before.Size() <= cursor {
				return false
			}
			document, err = os.ReadFile(ref.Path)
			if err != nil {
				e.printf("read document %s: %v", ref.Path, err)
				return false
			}
			after, err := os.Stat(ref.Path)
			if err != nil || before.Size() != after.Size() || !before.ModTime().Equal(after.ModTime()) || int64(len(document)) != after.Size() {
				return false
			}
			version = strconv.FormatInt(int64(len(document)), 10)
		}
		e.decodeMu.Lock()
		events, complete, decodeErr := source.DecodeDocument(ref.Path, document)
		e.decodeMu.Unlock()
		if decodeErr != nil {
			e.printf("decode document %s: %v", ref.Path, decodeErr)
			return false
		}
		if !complete {
			return false
		}
		for _, event := range events {
			if event.Ignored || event.Text == "" || e.dispatch == nil {
				continue
			}
			if !e.dispatch(event, ref.SessionID) {
				return false
			}
		}
		if e.checkpoints != nil {
			if err := e.checkpoints.Save(context.Background(), Checkpoint{Source: e.name, SourceKey: key, SessionID: ref.SessionID, Cursor: version, UpdatedAt: time.Now().UTC()}); err != nil {
				e.printf("save document checkpoint %s: %v", ref.Path, err)
				return false
			}
		}
		checkpointVersion = version
		if !snapshots {
			cursor = int64(len(document))
		}
		return true
	}
	process()
	signalReady()
	ticker := time.NewTicker(e.tailInterval)
	defer ticker.Stop()
	stale := time.NewTimer(e.staleTimeout)
	defer stale.Stop()
	for {
		select {
		case <-e.stopCh:
			return
		case <-current.stopCh:
			return
		case ack := <-current.refresh:
			process()
			close(ack)
		case <-ticker.C:
			if process() {
				if !stale.Stop() {
					select {
					case <-stale.C:
					default:
					}
				}
				stale.Reset(e.staleTimeout)
			}
		case <-stale.C:
			if _, statErr := os.Stat(ref.Path); statErr != nil {
				return
			}
			stale.Reset(e.staleTimeout)
		}
	}
}

func (e *Engine) load(key string) int64 {
	if e.checkpoints == nil {
		return 0
	}
	cp, ok, err := e.checkpoints.Get(context.Background(), e.name, key)
	if err != nil || !ok {
		return 0
	}
	parsed, scanErr := strconv.ParseInt(cp.Cursor, 10, 64)
	if scanErr != nil || parsed < 0 {
		return 0
	}
	return parsed
}

func (e *Engine) save(key, sessionID string, offset int64) bool {
	if e.checkpoints == nil {
		return true
	}
	if err := e.checkpoints.Save(context.Background(), Checkpoint{
		Source: e.name, SourceKey: key, SessionID: sessionID,
		Cursor: strconv.FormatInt(offset, 10), UpdatedAt: time.Now().UTC(),
	}); err != nil {
		e.printf("save checkpoint %s: %v", key, err)
		return false
	}
	return true
}

func (e *Engine) printf(format string, args ...any) {
	if e.logger != nil {
		e.logger.Printf(e.name+": "+format, args...)
	}
}
