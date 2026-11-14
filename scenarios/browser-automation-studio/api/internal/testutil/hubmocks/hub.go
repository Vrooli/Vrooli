package hubmocks

import (
	"sync"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	ws "github.com/vrooli/browser-automation-studio/websocket"
	bastimeline "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/timeline"
)

// Hub is a concurrency-safe HubInterface fake that records envelope broadcasts
// and can model frame subscribers for service and handler tests.
type Hub struct {
	mu                        sync.RWMutex
	ClientCount               int
	ExecutionFrameSubscribers map[string]bool
	RecordingSubscribers      map[string]bool
	envelopes                 []any
}

func New() *Hub {
	return &Hub{
		ExecutionFrameSubscribers: make(map[string]bool),
		RecordingSubscribers:      make(map[string]bool),
	}
}

var _ ws.HubInterface = (*Hub)(nil)

func (h *Hub) ServeWS(*websocket.Conn, *uuid.UUID) {}

func (h *Hub) BroadcastEnvelope(event any) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.envelopes = append(h.envelopes, event)
}

func (h *Hub) BroadcastEnvelopeCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.envelopes)
}

func (h *Hub) LastBroadcastEnvelope() any {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if len(h.envelopes) == 0 {
		return nil
	}
	return h.envelopes[len(h.envelopes)-1]
}

func (h *Hub) ResetBroadcastEnvelopes() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.envelopes = nil
}

func (h *Hub) BroadcastTimelineEntry(sessionID string, _ *bastimeline.TimelineEntry) ws.BroadcastResult {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if h.RecordingSubscribers[sessionID] {
		return ws.BroadcastResult{SubscriberCount: 1, SentCount: 1}
	}
	return ws.BroadcastResult{}
}

func (*Hub) BroadcastBinaryFrame(string, []byte) {}

func (h *Hub) HasRecordingFrameSubscribers(sessionID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.RecordingSubscribers[sessionID]
}

func (*Hub) BroadcastPerfStats(string, any)     {}
func (*Hub) BroadcastPageEvent(string, any)     {}
func (*Hub) BroadcastPageSwitch(string, string) {}

func (h *Hub) HasExecutionFrameSubscribers(executionID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.ExecutionFrameSubscribers[executionID]
}

func (*Hub) BroadcastExecutionFrame(string, *ws.ExecutionFrame) {}
func (*Hub) BroadcastExportProgress(*ws.ExportProgress)         {}

func (h *Hub) GetClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.ClientCount
}

func (*Hub) Run()                     {}
func (*Hub) CloseExecution(uuid.UUID) {}
