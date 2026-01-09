package main

import (
	"bufio"
	"encoding/json"
	"os"
	"sync"
	"time"
)

// clientRegistry manages WebSocket clients attached to a session.
type clientRegistry struct {
	mu      sync.Mutex
	clients map[*wsClient]struct{}
}

func newClientRegistry() *clientRegistry {
	return &clientRegistry{clients: make(map[*wsClient]struct{})}
}

func (c *clientRegistry) add(client *wsClient) {
	if client == nil {
		return
	}
	c.mu.Lock()
	c.clients[client] = struct{}{}
	c.mu.Unlock()
}

func (c *clientRegistry) remove(client *wsClient) {
	if client == nil {
		return
	}
	c.mu.Lock()
	delete(c.clients, client)
	c.mu.Unlock()
}

func (c *clientRegistry) broadcast(msg websocketEnvelope) {
	c.mu.Lock()
	for client := range c.clients {
		client.enqueue(msg)
	}
	c.mu.Unlock()
}

func (c *clientRegistry) closeAll() {
	c.mu.Lock()
	for client := range c.clients {
		client.close()
	}
	c.clients = make(map[*wsClient]struct{})
	c.mu.Unlock()
}

func (c *clientRegistry) count() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.clients)
}

// transcriptManager handles serialized transcript writes and syncing to disk.
type transcriptManager struct {
	mu       sync.Mutex
	file     *os.File
	writer   *bufio.Writer
	lastSync time.Time
	buffered int
}

func newTranscriptManager(file *os.File) *transcriptManager {
	if file == nil {
		return &transcriptManager{}
	}
	return &transcriptManager{
		file:   file,
		writer: bufio.NewWriter(file),
	}
}

func (tm *transcriptManager) write(entry transcriptEntry) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	if tm.writer == nil {
		return
	}
	encoded, err := json.Marshal(entry)
	if err != nil {
		return
	}
	if _, err := tm.writer.Write(encoded); err == nil {
		_, _ = tm.writer.WriteString("\n")
		tm.buffered += len(encoded) + 1
	}
}

func (tm *transcriptManager) flush(force bool, syncInterval time.Duration, minBytes int) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	if tm.writer == nil {
		return
	}
	if err := tm.writer.Flush(); err != nil {
		return
	}
	if tm.file == nil {
		return
	}
	if !force {
		if minBytes > 0 && tm.buffered < minBytes {
			return
		}
		if !tm.lastSync.IsZero() && time.Since(tm.lastSync) < syncInterval {
			return
		}
	}
	if err := tm.file.Sync(); err == nil {
		tm.lastSync = time.Now().UTC()
		tm.buffered = 0
	}
}

func (tm *transcriptManager) close() {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	if tm.writer != nil {
		_ = tm.writer.Flush()
	}
	if tm.file != nil {
		_ = tm.file.Close()
		tm.file = nil
	}
	tm.buffered = 0
	tm.writer = nil
}

// outputReplayBuffer maintains a rolling buffer of recent output for reconnects.
type outputReplayBuffer struct {
	mu         sync.RWMutex
	chunks     []bufferedOutput
	maxChunks  int
	maxBytes   int
	bytesTotal int
	truncated  bool
}

func newOutputReplayBuffer(maxChunks, maxBytes int) *outputReplayBuffer {
	return &outputReplayBuffer{
		chunks:    make([]bufferedOutput, 0, maxChunks),
		maxChunks: maxChunks,
		maxBytes:  maxBytes,
	}
}

func (b *outputReplayBuffer) append(payload outputPayload, size int) {
	b.mu.Lock()
	b.chunks = append(b.chunks, bufferedOutput{payload: payload, size: size})
	b.bytesTotal += size

	dropped := false
	for (b.maxChunks > 0 && len(b.chunks) > b.maxChunks) || (b.maxBytes > 0 && b.bytesTotal > b.maxBytes) {
		if len(b.chunks) == 0 {
			break
		}
		removed := b.chunks[0]
		b.chunks = b.chunks[1:]
		b.bytesTotal -= removed.size
		if b.bytesTotal < 0 {
			b.bytesTotal = 0
		}
		dropped = true
	}
	if dropped {
		b.truncated = true
	}
	b.mu.Unlock()
}

func (b *outputReplayBuffer) snapshot() ([]outputPayload, bool) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	payloads := make([]outputPayload, len(b.chunks))
	for i, entry := range b.chunks {
		payloads[i] = entry.payload
	}
	return payloads, b.truncated
}

func (b *outputReplayBuffer) len() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.chunks)
}
