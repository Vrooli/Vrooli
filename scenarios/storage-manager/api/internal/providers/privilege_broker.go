package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"runtime"
	"strings"
	"time"

	platformgo "github.com/vrooli/platform-go"
	"storage-manager/internal/cleanup"
)

type privilegeBrokerRequest struct {
	Version   string         `json:"version"`
	RequestID string         `json:"request_id"`
	Action    string         `json:"action"`
	Subject   map[string]any `json:"subject"`
	Journal   map[string]any `json:"journal,omitempty"`
	Docker    map[string]any `json:"docker,omitempty"`
}

type privilegeBrokerResponse struct {
	Status  string `json:"status"`
	Changed bool   `json:"changed"`
	Code    string `json:"code"`
}

type privilegeBrokerClient struct {
	socket string
	client *net.Dialer
}

// NewPrivilegeBrokerClient returns the production broker transport. An absent
// broker fails closed; conditional providers never fall back to sudo or direct
// privileged commands.
func NewPrivilegeBrokerClient() cleanup.BrokerActionClient {
	return &privilegeBrokerClient{socket: privilegeBrokerSocketPath(), client: &net.Dialer{Timeout: 2 * time.Second}}
}

func (c *privilegeBrokerClient) Do(ctx context.Context, action string, subject map[string]any) (cleanup.BrokerActionResult, error) {
	if c == nil || strings.TrimSpace(c.socket) == "" {
		return cleanup.BrokerActionResult{}, fmt.Errorf("privilege broker unavailable")
	}
	if runtime.GOOS == "windows" {
		// The broker's Windows named-pipe protocol is deliberately not guessed
		// through a Unix dialer. Until its typed transport exists, fail closed.
		return cleanup.BrokerActionResult{}, fmt.Errorf("privilege broker named-pipe transport is unavailable")
	}
	req := privilegeBrokerRequest{Version: "v1", RequestID: fmt.Sprintf("storage-manager-%d", time.Now().UnixNano()), Action: action, Subject: map[string]any{}}
	if value, ok := subject["journal"].(map[string]any); ok {
		req.Journal = value
	}
	if value, ok := subject["docker"].(map[string]any); ok {
		req.Docker = value
	}
	conn, err := c.client.DialContext(ctx, "unix", c.socket)
	if err != nil {
		return cleanup.BrokerActionResult{}, fmt.Errorf("dial privilege broker: %w", err)
	}
	defer conn.Close()
	if err := json.NewEncoder(conn).Encode(req); err != nil {
		return cleanup.BrokerActionResult{}, fmt.Errorf("send privilege broker request: %w", err)
	}
	var response privilegeBrokerResponse
	if err := json.NewDecoder(conn).Decode(&response); err != nil {
		return cleanup.BrokerActionResult{}, fmt.Errorf("read privilege broker response: %w", err)
	}
	if response.Status != "completed" {
		return cleanup.BrokerActionResult{}, fmt.Errorf("privilege broker rejected %s: %s", action, response.Code)
	}
	return cleanup.BrokerActionResult{Changed: response.Changed}, nil
}

func privilegeBrokerSocketPath() string {
	return platformgo.PrivilegeBrokerSocketPath()
}
