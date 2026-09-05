package retention

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"sync"
)

// recordingHandler captures slog records so a test can assert what an operator
// would actually see, rather than that a function was called.
type recordingHandler struct {
	mu      sync.Mutex
	records []string
}

func (h *recordingHandler) Enabled(context.Context, slog.Level) bool { return true }

func (h *recordingHandler) Handle(_ context.Context, r slog.Record) error {
	var b strings.Builder
	b.WriteString(r.Message)
	r.Attrs(func(a slog.Attr) bool {
		fmt.Fprintf(&b, " %s=%v", a.Key, a.Value)
		return true
	})
	h.mu.Lock()
	h.records = append(h.records, b.String())
	h.mu.Unlock()
	return nil
}

func (h *recordingHandler) WithAttrs([]slog.Attr) slog.Handler { return h }

func (h *recordingHandler) WithGroup(string) slog.Handler { return h }

func (h *recordingHandler) lines() []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]string(nil), h.records...)
}

func (h *recordingHandler) contains(substr string) bool {
	for _, line := range h.lines() {
		if strings.Contains(line, substr) {
			return true
		}
	}
	return false
}

func newRecordingLogger(h *recordingHandler) *slog.Logger { return slog.New(h) }

// discardLogger silences output for tests asserting on something other than
// logs.
func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
