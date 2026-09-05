package inapp

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"

	"github.com/google/uuid"

	"switchboard/internal/channels"
)

// OwnerAddress is the sender address the console stamps on its envelopes. The
// in-app surface is the owner's authenticated console, so its descriptor
// declares the owner default tier; this constant is the adapter's side of
// that contract.
const OwnerAddress = "owner"

type starter struct {
	mu    sync.Mutex
	start channels.StartFunc
}

func (a *Adapter) StartPath() string { return "/api/v1/channels/in-app/threads" }
func (a *Adapter) BindStart(fn channels.StartFunc) {
	a.starter.mu.Lock()
	a.starter.start = fn
	a.starter.mu.Unlock()
}

// StartHandler opens a new in-app conversation with one agent. The adapter
// mints the thread key; the injected StartFunc creates the binding and the
// durable thread so the conversation is listable before its first message.
func (a *Adapter) StartHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			AgentID string `json:"agent_id"`
		}
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&body); err != nil || strings.TrimSpace(body.AgentID) == "" {
			http.Error(w, "agent_id is required", http.StatusBadRequest)
			return
		}
		a.starter.mu.Lock()
		start := a.starter.start
		a.starter.mu.Unlock()
		if start == nil {
			http.Error(w, "in-app conversations cannot be started right now", http.StatusServiceUnavailable)
			return
		}
		started := channels.Started{ChannelID: a.ID(), ThreadKey: uuid.NewString(), Address: OwnerAddress}
		threadID, err := start(r.Context(), strings.TrimSpace(body.AgentID), started)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]string{"thread_id": threadID, "thread_key": started.ThreadKey, "channel_id": started.ChannelID, "address": started.Address})
	})
}

var _ channels.ThreadStarter = (*Adapter)(nil)
