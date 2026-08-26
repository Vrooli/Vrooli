// Package tailer contains the shared lifecycle and cursor machinery for
// append-only agent transcript adapters. Source implementations only own
// discovery, line decoding, and agent metadata extraction.
package tailer

import (
	"bufio"
	"context"
	"errors"
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
	Role    string
	Text    string
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

// RunPollLoop is the small lifecycle seam used by legacy adapters while they
// are migrated to Engine. It keeps timer ownership and stop semantics in this
// package, so adapters cannot accidentally diverge on shutdown behavior.
func RunPollLoop(stop <-chan struct{}, interval time.Duration, scan func()) {
	if interval <= 0 {
		interval = time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			scan()
		}
	}
}

// Config controls one engine. Zero intervals use safe defaults.
type Config struct {
	Name         string
	Source       Source
	Checkpoints  CheckpointStore
	PollInterval time.Duration
	TailInterval time.Duration
	StaleTimeout time.Duration
	Dispatch     func(Event, string)
	Logger       *log.Logger
}

// Engine owns discovery, watcher lifecycle, byte-offset reads, and durable
// cursor updates for one transcript source.
type Engine struct {
	name         string
	source       Source
	checkpoints  CheckpointStore
	pollInterval time.Duration
	tailInterval time.Duration
	staleTimeout time.Duration
	dispatch     func(Event, string)
	logger       *log.Logger

	stopOnce  sync.Once
	stopCh    chan struct{}
	pollWG    sync.WaitGroup
	tailWG    sync.WaitGroup
	captureWG sync.WaitGroup
	mu        sync.Mutex
	watchers  map[string]*watcher
	decodeMu  sync.Mutex
}

type watcher struct {
	refresh chan chan struct{}
	done    chan struct{}
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
		name:         cfg.Name,
		source:       cfg.Source,
		checkpoints:  cfg.Checkpoints,
		pollInterval: cfg.PollInterval,
		tailInterval: cfg.TailInterval,
		staleTimeout: cfg.StaleTimeout,
		dispatch:     cfg.Dispatch,
		logger:       cfg.Logger,
		stopCh:       make(chan struct{}),
		watchers:     make(map[string]*watcher),
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
	e.tailWG.Wait()
	e.captureWG.Wait()
}

// Scan runs one discovery pass immediately. It is safe to use in tests and
// in hosts that trigger discovery from an external scheduler.
func (e *Engine) Scan(ctx context.Context) {
	if e == nil || e.source == nil {
		return
	}
	files, err := e.source.DiscoverFiles(ctx)
	if err != nil {
		e.printf("discover files: %v", err)
		return
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
			current = &watcher{refresh: make(chan chan struct{}), done: make(chan struct{})}
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
			line, readErr := reader.ReadBytes('\n')
			if readErr != nil {
				if !errors.Is(readErr, io.EOF) {
					e.printf("read %s: %v", ref.Path, readErr)
				}
				return processed
			}
			processed = true
			currentOffset += int64(len(line))
			e.decodeMu.Lock()
			events, decodeErr := e.source.DecodeLine(ref.Path, line)
			e.decodeMu.Unlock()
			if decodeErr != nil {
				e.printf("decode %s: %v", ref.Path, decodeErr)
				continue
			}
			commit := len(events) == 0
			for _, event := range events {
				if event.Commit {
					commit = true
				}
				if !event.Ignored && e.dispatch != nil && event.Text != "" {
					e.dispatch(event, ref.SessionID)
				}
			}
			if commit {
				e.save(key, ref.SessionID, currentOffset)
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

func (e *Engine) save(key, sessionID string, offset int64) {
	if e.checkpoints == nil {
		return
	}
	if err := e.checkpoints.Save(context.Background(), Checkpoint{
		Source: e.name, SourceKey: key, SessionID: sessionID,
		Cursor: strconv.FormatInt(offset, 10), UpdatedAt: time.Now().UTC(),
	}); err != nil {
		e.printf("save checkpoint %s: %v", key, err)
	}
}

func (e *Engine) printf(format string, args ...any) {
	if e.logger != nil {
		e.logger.Printf(e.name+": "+format, args...)
	}
}
