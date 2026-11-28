package metering

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// LPBSGateway calls the authenticated LPBS reservation boundary. The caller
// supplies the access token; Switchboard never invents a user identity or a
// shared billing credential.
type LPBSGateway struct {
	BaseURL  string
	Token    string
	LimitKey string
	Client   *http.Client
}

func (g LPBSGateway) Reserve(ctx context.Context, _ string, estimate int64) (string, error) {
	var response struct {
		ReservationID string `json:"reservation_id"`
	}
	if err := g.post(ctx, "/api/v1/usage/reservations", map[string]any{"limit_key": g.LimitKey, "amount": estimate}, &response); err != nil {
		return "", err
	}
	if strings.TrimSpace(response.ReservationID) == "" {
		return "", fmt.Errorf("LPBS reservation response has no reservation_id")
	}
	return response.ReservationID, nil
}

func (g LPBSGateway) Finalize(ctx context.Context, id string, actual int64) error {
	return g.post(ctx, "/api/v1/usage/reservations/"+id+"/finalize", map[string]any{"actual_amount": actual}, nil)
}

func (g LPBSGateway) Release(ctx context.Context, id string) error {
	return g.post(ctx, "/api/v1/usage/reservations/"+id+"/release", nil, nil)
}

func (g LPBSGateway) post(ctx context.Context, path string, body any, out any) error {
	base := strings.TrimRight(strings.TrimSpace(g.BaseURL), "/")
	if base == "" || strings.TrimSpace(g.Token) == "" || strings.TrimSpace(g.LimitKey) == "" {
		return fmt.Errorf("LPBS gateway requires base URL, access token, and limit key")
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("encode LPBS request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+path, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("create LPBS request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+g.Token)
	req.Header.Set("Content-Type", "application/json")
	client := g.Client
	if client == nil {
		client = http.DefaultClient
	}
	res, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("call LPBS: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		data, _ := io.ReadAll(io.LimitReader(res.Body, 4096))
		return fmt.Errorf("LPBS returned HTTP %d: %s", res.StatusCode, strings.TrimSpace(string(data)))
	}
	if out != nil {
		if err := json.NewDecoder(res.Body).Decode(out); err != nil {
			return fmt.Errorf("decode LPBS response: %w", err)
		}
	}
	return nil
}
