package handoffs

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// HTTPRelay is optional delivery only. Its payload is a handoff summary and
// checkpoint metadata; it never carries document bytes or channel secrets.
type HTTPRelay struct {
	BaseURL string
	Client  *http.Client
}

func (r HTTPRelay) Deliver(ctx context.Context, h Handoff) error {
	payload, err := json.Marshal(map[string]any{"handoff_id": h.ID, "persona_id": h.PersonaID, "title": h.Title, "human_action": h.HumanAction, "deadline": h.Deadline.UTC().Format(time.RFC3339), "state": h.State})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(r.BaseURL, "/")+"/api/v1/notifications/handoffs", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	client := r.Client
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	response, err := client.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("notification-hub status %d", response.StatusCode)
	}
	return nil
}
