package measures

import (
	"context"
	"sync"
	"time"

	"agent-manager/internal/outcomeledger"
)

// A window's inputs take seconds to read (transcript_usage holds millions of
// rows and the 30-day aggregate scans them), and the ledger job only refreshes
// every few minutes, so re-reading per request adds cost and no accuracy. A
// read within inputFreshFor reuses them; one within inputStaleFor reuses them
// and refreshes in the background, so a reader rarely pays for the scan. Windows
// that end now are matched by length, since their edges move with the clock.
const (
	inputFreshFor     = 2 * time.Minute
	inputStaleFor     = 15 * time.Minute
	inputSpanBucket   = 10 * time.Minute
	inputNowTolerance = time.Minute
	inputEntries      = 8
)

// cachedLedger is the ledger reader with the read model's input cache.
type cachedLedger struct {
	outcomeledger.Reader
	inputs *inputCache
}

type inputEntry struct {
	loaded     time.Time
	inputs     outcomeInputs
	refreshing bool
}

type inputKey struct {
	span     time.Duration
	from, to int64
}

type inputCache struct {
	mu      sync.Mutex
	entries map[inputKey]*inputEntry
	// loading serializes a first read per key, so parallel views scan once.
	loading map[inputKey]*sync.Mutex
}

func newInputCache() *inputCache {
	return &inputCache{entries: map[inputKey]*inputEntry{}, loading: map[inputKey]*sync.Mutex{}}
}

func inputKeyFor(now, from, to time.Time) inputKey {
	if gap := now.Sub(to); gap >= 0 && gap <= inputNowTolerance {
		return inputKey{span: to.Sub(from).Round(inputSpanBucket)}
	}
	return inputKey{from: from.UnixNano(), to: to.UnixNano()}
}

// get returns the window's inputs. Callers treat them as read-only.
func (c *inputCache) get(ctx context.Context, now, from, to time.Time, load func(context.Context) (outcomeInputs, error)) (outcomeInputs, error) {
	key := inputKeyFor(now, from, to)
	if inputs, ok := c.lookup(key, now, load); ok {
		return inputs, nil
	}
	c.mu.Lock()
	gate := c.loading[key]
	if gate == nil {
		gate = &sync.Mutex{}
		c.loading[key] = gate
	}
	c.mu.Unlock()
	gate.Lock()
	defer gate.Unlock()
	if inputs, ok := c.lookup(key, now, load); ok {
		return inputs, nil
	}
	inputs, err := load(ctx)
	if err != nil {
		return outcomeInputs{}, err
	}
	c.store(key, now, inputs)
	return inputs, nil
}

func (c *inputCache) lookup(key inputKey, now time.Time, load func(context.Context) (outcomeInputs, error)) (outcomeInputs, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	entry := c.entries[key]
	if entry == nil {
		return outcomeInputs{}, false
	}
	age := now.Sub(entry.loaded)
	if age > inputStaleFor {
		return outcomeInputs{}, false
	}
	if age > inputFreshFor && !entry.refreshing {
		entry.refreshing = true
		go func() {
			inputs, err := load(context.Background())
			if err != nil {
				c.mu.Lock()
				entry.refreshing = false
				c.mu.Unlock()
				return
			}
			c.store(key, now, inputs)
		}()
	}
	return entry.inputs, true
}

func (c *inputCache) store(key inputKey, now time.Time, inputs outcomeInputs) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[key] = &inputEntry{loaded: now, inputs: inputs}
	for len(c.entries) > inputEntries {
		var oldest inputKey
		var at time.Time
		for k, entry := range c.entries {
			if at.IsZero() || entry.loaded.Before(at) {
				oldest, at = k, entry.loaded
			}
		}
		delete(c.entries, oldest)
	}
}
