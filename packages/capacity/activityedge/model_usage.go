package activityedge

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/vrooli/vrooli/packages/capacity/modelretention"
)

// modelBodyLimit bounds how much of a request body the edge buffers to read
// its model name. Larger bodies are proxied unread and hold the
// unidentified-request fence instead, which still blocks every eviction.
const modelBodyLimit = 64 << 20

// modelUsageFlushInterval is how often observed requests reach the durable
// usage file storage-manager reads.
const modelUsageFlushInterval = 30 * time.Second

// modelGate records per-model last requests and holds the per-model eviction
// fence for the length of each request. It is enabled only for model stores
// (ForResource sets Handlers.ModelStore).
type modelGate struct {
	fence    modelretention.Fence
	recorder *modelretention.Recorder
	logf     func(string, ...any)
	stop     context.CancelFunc
	done     chan struct{}
	once     sync.Once
}

func (h *Handlers) startModelGate() {
	resource := strings.TrimSpace(h.ModelStore)
	if resource == "" {
		return
	}
	paths, err := modelretention.StorePaths(resource)
	if err != nil {
		h.errorf("activity-edge model usage disabled: %v\n", err)
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	gate := &modelGate{
		fence:    modelretention.Fence{Dir: paths.FenceDir},
		recorder: modelretention.NewRecorder(resource, paths.UsageFile, h.now),
		logf:     func(format string, args ...any) { h.errorf("activity-edge "+format+"\n", args...) },
		stop:     cancel,
		done:     make(chan struct{}),
	}
	go func() {
		defer close(gate.done)
		gate.recorder.Run(ctx, modelUsageFlushInterval, gate.logf)
	}()
	h.trackerMu.Lock()
	h.models = gate
	h.trackerMu.Unlock()
}

func (h *Handlers) stopModelGate() {
	h.trackerMu.Lock()
	gate := h.models
	h.trackerMu.Unlock()
	if gate == nil {
		return
	}
	gate.once.Do(func() {
		gate.stop()
		<-gate.done
	})
}

func (h *Handlers) modelGateFor() *modelGate {
	h.trackerMu.Lock()
	defer h.trackerMu.Unlock()
	return h.models
}

// enter reads the request's model, restores the body, holds the model's
// fence until release, and attributes a use to the connection's caller. A fenced model returns modelretention.ErrModelFenced;
// fence infrastructure failures fail open like every other edge dependency.
func (g *modelGate) enter(r *http.Request) (*http.Request, func(), error) {
	model, use, body, err := peekModel(r.Body)
	if err != nil {
		return r, func() {}, err
	}
	r.Body = body
	release, err := g.fence.EnterRequest(model)
	if errors.Is(err, modelretention.ErrModelFenced) {
		return r, func() {}, err
	}
	if err != nil {
		g.logf("model fence unavailable, proxying unfenced: %v", err)
		release = func() {}
	}
	// Only POST runs a model. DELETE /api/delete names a model and holds its
	// fence, but deleting is not use.
	use = use && r.Method == http.MethodPost
	if use {
		at := g.recorder.Now()
		g.recorder.Record(model)
		peerFrom(r.Context()).observe(func(caller string) { g.recorder.RecordCaller(model, caller, at) })
	}
	return r, func() {
		if use {
			// A long generation is use until it ends.
			g.recorder.Touch(model)
		}
		release()
	}, nil
}

func peekModel(body io.ReadCloser) (string, bool, io.ReadCloser, error) {
	if body == nil || body == http.NoBody {
		return "", false, body, nil
	}
	buffered, err := io.ReadAll(io.LimitReader(body, modelBodyLimit+1))
	if err != nil {
		return "", false, body, err
	}
	if len(buffered) > modelBodyLimit {
		return "", false, readCloser{Reader: io.MultiReader(bytes.NewReader(buffered), body), Closer: body}, nil
	}
	_ = body.Close()
	model, use := modelretention.RequestModel(buffered)
	return model, use, io.NopCloser(bytes.NewReader(buffered)), nil
}

type readCloser struct {
	io.Reader
	io.Closer
}

func (g *modelGate) reject(w http.ResponseWriter, err error) {
	if errors.Is(err, modelretention.ErrModelFenced) {
		http.Error(w, "model is being evicted from disk; retry after it is reacquired", http.StatusServiceUnavailable)
		return
	}
	http.Error(w, "read request body", http.StatusBadRequest)
}
