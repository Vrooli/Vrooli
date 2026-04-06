package emitter

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

// EventPayload is the data sent to vrooli-events.
type EventPayload struct {
	EventID        string            `json:"eventId"`
	EventType      string            `json:"eventType"`
	SourceScenario string            `json:"sourceScenario"`
	TargetScenario string            `json:"targetScenario,omitempty"`
	Payload        json.RawMessage   `json:"payload,omitempty"`
	Metadata       map[string]string `json:"metadata,omitempty"`
}

// Config for the fire-and-forget emitter.
type Config struct {
	EventsURL      string        // vrooli-events ingest URL
	SourceScenario string        // calling scenario name
	BufferSize     int           // channel capacity (default 256)
	BatchSize      int           // max batch size (default 10)
	FlushInterval  time.Duration // max wait before flush (default 100ms)
	MaxRetries     int           // retry attempts (default 3)
	RetryBackoff   time.Duration // initial backoff (default 1s)
}

// Emitter sends events to vrooli-events without blocking the caller.
type Emitter struct {
	config  Config
	ch      chan EventPayload
	client  *http.Client
	dropped atomic.Int64
	sent    atomic.Int64
	wg      sync.WaitGroup
}

// NewEmitter creates a new Emitter that batches and sends events asynchronously.
func NewEmitter(cfg Config) *Emitter {
	if cfg.BufferSize <= 0 {
		cfg.BufferSize = 256
	}
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 10
	}
	if cfg.FlushInterval <= 0 {
		cfg.FlushInterval = 100 * time.Millisecond
	}
	if cfg.MaxRetries <= 0 {
		cfg.MaxRetries = 3
	}
	if cfg.RetryBackoff <= 0 {
		cfg.RetryBackoff = 1 * time.Second
	}

	e := &Emitter{
		config: cfg,
		ch:     make(chan EventPayload, cfg.BufferSize),
		client: &http.Client{Timeout: 5 * time.Second},
	}
	e.wg.Add(1)
	go e.drain()
	return e
}

// Emit queues an event for async delivery. Non-blocking; drops if buffer full.
func (e *Emitter) Emit(evt EventPayload) bool {
	select {
	case e.ch <- evt:
		return true
	default:
		e.dropped.Add(1)
		return false
	}
}

// Stats returns counters for sent and dropped events.
func (e *Emitter) Stats() (sent, dropped int64) {
	return e.sent.Load(), e.dropped.Load()
}

// Close flushes remaining events and stops the drain goroutine.
func (e *Emitter) Close() {
	close(e.ch)
	e.wg.Wait()
}

func (e *Emitter) drain() {
	defer e.wg.Done()
	batch := make([]EventPayload, 0, e.config.BatchSize)
	timer := time.NewTimer(e.config.FlushInterval)
	defer timer.Stop()

	flush := func() {
		if len(batch) == 0 {
			return
		}
		e.sendBatch(batch)
		batch = batch[:0]
		timer.Reset(e.config.FlushInterval)
	}

	for {
		select {
		case evt, ok := <-e.ch:
			if !ok {
				flush()
				return
			}
			batch = append(batch, evt)
			if len(batch) >= e.config.BatchSize {
				flush()
			}
		case <-timer.C:
			flush()
			timer.Reset(e.config.FlushInterval)
		}
	}
}

func (e *Emitter) sendBatch(batch []EventPayload) {
	for _, evt := range batch {
		e.sendOne(evt)
	}
}

func (e *Emitter) sendOne(evt EventPayload) {
	body, err := json.Marshal(evt)
	if err != nil {
		log.Printf("[emitter] marshal error: %v", err)
		e.dropped.Add(1)
		return
	}

	ctx := context.Background()

	for attempt := 0; attempt <= e.config.MaxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(e.config.RetryBackoff * time.Duration(1<<(attempt-1)))
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, e.config.EventsURL, bytes.NewReader(body))
		if err != nil {
			continue
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := e.client.Do(req)
		if err != nil {
			continue
		}
		resp.Body.Close()
		if resp.StatusCode < 400 {
			e.sent.Add(1)
			return
		}
	}
	e.dropped.Add(1)
}
