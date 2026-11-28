package monetization

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

const maxQuotaReservationResponse = 32 << 10

// QuotaReservationClient uses LPBS's atomic count-based reservation API. The
// authority resolves the user's tier and accounts for pending holds in one
// transaction; callers cannot select a stronger tier.
type QuotaReservationClient struct {
	BaseURL    string
	HTTPClient *http.Client
}

// QuotaHold tracks one remotely reserved count until it is committed or
// released. It stores the short-lived bearer only in process memory.
type QuotaHold struct {
	client QuotaReservationClient
	token  string
	id     string
	amount int64
	mu     sync.Mutex
	done   bool
}

// Reserve atomically holds count-based capacity for a meter. A zero window
// uses LPBS's default bounded reservation lifetime.
func (c QuotaReservationClient) Reserve(ctx context.Context, accessToken, meter string, amount, windowSeconds int64) (string, error) {
	body, err := json.Marshal(map[string]any{"limit_key": strings.TrimSpace(meter), "amount": amount, "window_seconds": windowSeconds})
	if err != nil {
		return "", err
	}
	var response struct {
		ReservationID string `json:"reservation_id"`
	}
	if err := c.request(ctx, http.MethodPost, "/api/v1/usage/reservations", accessToken, body, &response); err != nil {
		return "", err
	}
	if strings.TrimSpace(response.ReservationID) == "" {
		return "", fmt.Errorf("LPBS quota reservation omitted reservation_id")
	}
	return response.ReservationID, nil
}

// Begin reserves capacity and returns a one-shot hold for the caller to
// commit after its local operation record is durably accepted.
func (c QuotaReservationClient) Begin(ctx context.Context, accessToken, meter string, amount, windowSeconds int64) (*QuotaHold, error) {
	id, err := c.Reserve(ctx, accessToken, meter, amount, windowSeconds)
	if err != nil {
		return nil, err
	}
	return &QuotaHold{client: c, token: strings.TrimSpace(accessToken), id: id, amount: amount}, nil
}

// Commit converts this hold into usage. A successful repeat is idempotent in
// the local client, while a failed/uncertain transport leaves the hold open.
func (h *QuotaHold) Commit(ctx context.Context, amount int64) error {
	if h == nil {
		return fmt.Errorf("LPBS quota hold is required")
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.done {
		return nil
	}
	if amount < 0 || amount > h.amount {
		return fmt.Errorf("quota commit amount exceeds its reservation")
	}
	if err := h.client.Finalize(ctx, h.token, h.id, amount); err != nil {
		return err
	}
	h.done = true
	return nil
}

// Release frees an uncommitted hold. Release is idempotent in the local client.
func (h *QuotaHold) Release(ctx context.Context) error {
	if h == nil {
		return nil
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.done {
		return nil
	}
	if err := h.client.Release(ctx, h.token, h.id); err != nil {
		return err
	}
	h.done = true
	return nil
}

// Finalize turns a held count into committed usage.
func (c QuotaReservationClient) Finalize(ctx context.Context, accessToken, reservationID string, amount int64) error {
	body, err := json.Marshal(map[string]int64{"actual_amount": amount})
	if err != nil {
		return err
	}
	return c.request(ctx, http.MethodPost, "/api/v1/usage/reservations/"+url.PathEscape(strings.TrimSpace(reservationID))+"/finalize", accessToken, body, nil)
}

// Release frees a hold when the reserved operation was not accepted.
func (c QuotaReservationClient) Release(ctx context.Context, accessToken, reservationID string) error {
	return c.request(ctx, http.MethodPost, "/api/v1/usage/reservations/"+url.PathEscape(strings.TrimSpace(reservationID))+"/release", accessToken, nil, nil)
}

func (c QuotaReservationClient) request(ctx context.Context, method, path, accessToken string, body []byte, result any) error {
	base := strings.TrimRight(strings.TrimSpace(c.BaseURL), "/")
	token := strings.TrimSpace(accessToken)
	if token == "" {
		return fmt.Errorf("LPBS quota reservation access token is required")
	}
	if err := validateQuotaAuthority(base); err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(ctx, method, base+path, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create LPBS quota reservation request: %w", err)
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Authorization", "Bearer "+token)
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	client := c.HTTPClient
	if client == nil {
		client = &http.Client{}
	}
	copy := *client
	copy.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	response, err := copy.Do(request)
	if err != nil {
		return fmt.Errorf("LPBS quota reservation unavailable: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("LPBS quota reservation returned status %d", response.StatusCode)
	}
	if result == nil {
		_, err = io.Copy(io.Discard, io.LimitReader(response.Body, maxQuotaReservationResponse))
		return err
	}
	responseBody, err := io.ReadAll(io.LimitReader(response.Body, maxQuotaReservationResponse+1))
	if err != nil {
		return fmt.Errorf("read LPBS quota reservation response: %w", err)
	}
	if len(responseBody) > maxQuotaReservationResponse {
		return fmt.Errorf("LPBS quota reservation response is too large")
	}
	if err := json.Unmarshal(responseBody, result); err != nil {
		return fmt.Errorf("decode LPBS quota reservation response: %w", err)
	}
	return nil
}

func validateQuotaAuthority(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return fmt.Errorf("LPBS quota authority URL is invalid")
	}
	if strings.EqualFold(u.Scheme, "https") {
		return nil
	}
	if !strings.EqualFold(u.Scheme, "http") {
		return fmt.Errorf("LPBS quota authority must use HTTPS or loopback HTTP")
	}
	host := strings.TrimSuffix(strings.ToLower(u.Hostname()), ".")
	if host == "localhost" {
		return nil
	}
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		return fmt.Errorf("LPBS quota authority must use HTTPS or loopback HTTP")
	}
	return nil
}
