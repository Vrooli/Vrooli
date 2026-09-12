package channels

import (
	"context"
	"net/http"
)

// Started describes a conversation an adapter has opened on request. The
// address is what the adapter will stamp on inbound envelopes for this
// conversation, so the caller can create a binding that resolves.
type Started struct {
	ChannelID string `json:"channel_id"`
	ThreadKey string `json:"thread_key"`
	Address   string `json:"address"`
}

// StartFunc is injected into a ThreadStarter by the HTTP module. It creates
// the binding and durable thread for a freshly started conversation and
// returns the thread id.
type StartFunc func(ctx context.Context, agentID string, started Started) (threadID string, err error)

// ThreadStarter is an optional adapter capability: the adapter can open a new
// conversation with an agent on demand (a chat surface that has no external
// sender to arrive first). Core code mounts every starter generically; which
// adapters implement it is the adapter's decision, not a channel-id branch.
type ThreadStarter interface {
	StartPath() string
	StartHandler() http.Handler
	BindStart(StartFunc)
}
