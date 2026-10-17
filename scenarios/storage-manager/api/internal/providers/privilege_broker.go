package providers

import (
	"context"
	"fmt"
	"runtime"
	"time"

	platformgo "github.com/vrooli/platform-go"
	"storage-manager/internal/cleanup"
)

type privilegeBrokerRequest struct {
	Version   string         `json:"version"`
	RequestID string         `json:"request_id"`
	Action    string         `json:"action"`
	Subject   map[string]any `json:"subject"`
	Log       map[string]any `json:"log,omitempty"`
	Journal   map[string]any `json:"journal,omitempty"`
	Docker    map[string]any `json:"docker,omitempty"`
}

type privilegeBrokerResponse struct {
	Version   string `json:"version"`
	RequestID string `json:"request_id"`
	Action    string `json:"action"`
	Status    string `json:"status"`
	Changed   bool   `json:"changed"`
	Code      string `json:"code"`
}

type privilegeBrokerClient struct {
	exchange func(context.Context, any, any) error
}

// NewPrivilegeBrokerClient returns the production broker transport. An absent
// broker fails closed; conditional providers never fall back to sudo or direct
// privileged commands.
func NewPrivilegeBrokerClient() cleanup.BrokerActionClient {
	transport := platformgo.PrivilegeBrokerTransport{SocketPath: platformgo.PrivilegeBrokerSocketPath(), Timeout: 2 * time.Minute}
	return &privilegeBrokerClient{exchange: transport.Exchange}
}

func (c *privilegeBrokerClient) Do(ctx context.Context, action string, subject map[string]any) (cleanup.BrokerActionResult, error) {
	if c == nil || c.exchange == nil {
		return cleanup.BrokerActionResult{}, fmt.Errorf("privilege broker unavailable")
	}
	if runtime.GOOS == "windows" {
		// The broker's Windows named-pipe protocol is deliberately not guessed
		// through a Unix dialer. Until its typed transport exists, fail closed.
		return cleanup.BrokerActionResult{}, fmt.Errorf("privilege broker named-pipe transport is unavailable")
	}
	req := privilegeBrokerRequest{Version: "v1", RequestID: fmt.Sprintf("storage-manager-%d", time.Now().UnixNano()), Action: action, Subject: map[string]any{}}
	var key string
	switch action {
	case "log.rotate.force":
		key = "log"
	case "journald.vacuum":
		key = "journal"
	case "docker.prune.unused-images", "docker.prune.unused-volumes":
		key = "docker"
	default:
		return cleanup.BrokerActionResult{}, fmt.Errorf("storage privilege broker action is not allowed: %s", action)
	}
	value, ok := subject[key].(map[string]any)
	if !ok || len(subject) != 1 {
		return cleanup.BrokerActionResult{}, fmt.Errorf("storage privilege broker requires only the %s subject", key)
	}
	switch key {
	case "log":
		req.Log = value
	case "journal":
		req.Journal = value
	case "docker":
		req.Docker = value
	}
	var response privilegeBrokerResponse
	if err := c.exchange(ctx, req, &response); err != nil {
		return cleanup.BrokerActionResult{}, fmt.Errorf("read privilege broker response: %w", err)
	}
	if response.Version != req.Version || response.RequestID != req.RequestID || response.Action != req.Action {
		return cleanup.BrokerActionResult{}, fmt.Errorf("privilege broker response identity mismatch")
	}
	if response.Status != "completed" {
		return cleanup.BrokerActionResult{}, fmt.Errorf("privilege broker rejected %s: %s", action, response.Code)
	}
	return cleanup.BrokerActionResult{Changed: response.Changed}, nil
}
