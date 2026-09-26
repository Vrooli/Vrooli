// Package autoheal provides a thin HTTP client to the vrooli-autoheal
// scenario for the forensics page's autoheal panel.
//
// Contract: this client never returns an error. Timeouts, refused
// connections, non-2xx responses, malformed JSON — all degrade to
// {available: false, reason: ...}. The forensics endpoint composes its
// summary even when autoheal is offline.
package autoheal

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/vrooli/api-core/discovery"
)

// ForensicsRelevantCheck is the subset of an autoheal check we surface.
type ForensicsRelevantCheck struct {
	CheckID   string                 `json:"checkId"`
	Status    string                 `json:"status"`
	Message   string                 `json:"message,omitempty"`
	Category  string                 `json:"category,omitempty"`
	Details   map[string]interface{} `json:"details,omitempty"`
	LastRunAt string                 `json:"lastRunAt,omitempty"`
}

// Envelope is the autoheal panel data block.
type Envelope struct {
	Available bool                     `json:"available"`
	Reason    string                   `json:"reason,omitempty"`
	Checks    []ForensicsRelevantCheck `json:"checks,omitempty"`
}

// forensicsCheckIDs are the four forensics-specific autoheal checks; any
// other system-* check is also surfaced.
var forensicsCheckIDs = map[string]bool{
	"system-pstore-evidence": true,
	"system-boot-history":    true,
	"system-mce-recent":      true,
	"system-pm-runtime-hog":  true,
}

// Config controls Client behavior.
type Config struct {
	BaseURL string        // optional; falls back to env / discovery
	Timeout time.Duration // default 2s
}

// Client is an autoheal HTTP client.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient builds a Client.
func NewClient(cfg Config) *Client {
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 2 * time.Second
	}
	return &Client{
		baseURL:    cfg.BaseURL,
		httpClient: &http.Client{Timeout: timeout},
	}
}

// Forensics returns the forensics-relevant subset of autoheal checks.
// Always returns a non-nil envelope; never errors.
func (c *Client) Forensics(ctx context.Context) Envelope {
	if c == nil {
		return Envelope{Reason: "autoheal client not configured"}
	}

	url, err := c.resolveURL(ctx)
	if err != nil {
		return Envelope{Reason: "resolve autoheal url: " + err.Error()}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url+"/api/v1/status", nil)
	if err != nil {
		return Envelope{Reason: "build request: " + err.Error()}
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return Envelope{Reason: "autoheal unreachable: " + err.Error()}
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Envelope{Reason: fmt.Sprintf("autoheal returned status %d", resp.StatusCode)}
	}

	var payload struct {
		Checks []struct {
			CheckID   string                 `json:"checkId"`
			Status    string                 `json:"status"`
			Message   string                 `json:"message"`
			Category  string                 `json:"category"`
			Details   map[string]interface{} `json:"details"`
			LastRunAt string                 `json:"lastRunAt"`
		} `json:"checks"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return Envelope{Reason: "autoheal payload malformed: " + err.Error()}
	}

	env := Envelope{Available: true}
	for _, ch := range payload.Checks {
		if !forensicsCheckIDs[ch.CheckID] && !strings.HasPrefix(ch.CheckID, "system-") {
			continue
		}
		env.Checks = append(env.Checks, ForensicsRelevantCheck{
			CheckID:   ch.CheckID,
			Status:    ch.Status,
			Message:   ch.Message,
			Category:  ch.Category,
			Details:   ch.Details,
			LastRunAt: ch.LastRunAt,
		})
	}
	return env
}

func (c *Client) resolveURL(ctx context.Context) (string, error) {
	if c.baseURL != "" {
		return strings.TrimRight(c.baseURL, "/"), nil
	}
	if env := os.Getenv("VROOLI_AUTOHEAL_API_URL"); env != "" {
		return strings.TrimRight(env, "/"), nil
	}
	url, err := discovery.ResolveScenarioURLDefault(ctx, "vrooli-autoheal")
	if err != nil {
		return "", err
	}
	return strings.TrimRight(url, "/"), nil
}
