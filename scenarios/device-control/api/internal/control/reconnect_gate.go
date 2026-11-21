package control

import (
	"sync"
	"time"
)

// reconnectGate bounds automatic endpoint reconnects during inventory. A failed
// attempt (device asleep, address unreachable) blocks the next one for
// reconnectBackoff so a powered-off TV cannot tax every inventory read.
type reconnectGate struct {
	mu       sync.Mutex
	nextTry  map[string]time.Time
	inFlight map[string]bool
}

const reconnectBackoff = 60 * time.Second

// allow reserves the attempt for id; the caller must follow with record.
// A nil gate never allows a reconnect.
func (g *reconnectGate) allow(id string, now time.Time) bool {
	if g == nil {
		return false
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.inFlight[id] || now.Before(g.nextTry[id]) {
		return false
	}
	if g.inFlight == nil {
		g.inFlight = map[string]bool{}
	}
	g.inFlight[id] = true
	return true
}

func (g *reconnectGate) record(id string, now time.Time, ok bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	delete(g.inFlight, id)
	if g.nextTry == nil {
		g.nextTry = map[string]time.Time{}
	}
	if ok {
		delete(g.nextTry, id)
		return
	}
	g.nextTry[id] = now.Add(reconnectBackoff)
}
