// Package integration contains narrow cross-scenario clients. It does not
// duplicate either scenario's database or business rules.
package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"compute-manager/internal/clock"
	"compute-manager/internal/meter"
)

type LPBSCredits struct {
	BaseURL string
	Token   func(context.Context) (string, error)
	Client  *http.Client
	Now     func() time.Time
}

func (c *LPBSCredits) ReserveCredits(ctx context.Context, user, limit string, amount float64, window time.Duration) (meter.Reservation, error) {
	var response struct {
		ReservationID string `json:"reservation_id"`
	}
	if err := c.call(ctx, user, http.MethodPost, "/api/v1/usage/reservations", map[string]any{"limit_key": limit, "amount": int64(amount), "window_seconds": int64(window.Seconds())}, &response); err != nil {
		return meter.Reservation{}, err
	}
	if response.ReservationID == "" {
		return meter.Reservation{}, fmt.Errorf("landing-page-business-suite returned no reservation id")
	}
	now := c.Now
	if now == nil {
		now = clock.System{}.Now
	}
	return meter.Reservation{ID: response.ReservationID, ExpiresAt: now().Add(window)}, nil
}

func (c *LPBSCredits) ReleaseReservation(ctx context.Context, id string) error {
	return c.call(ctx, "", http.MethodPost, "/api/v1/usage/reservations/"+id+"/release", map[string]any{}, nil)
}

func (c *LPBSCredits) FinalizeReservation(ctx context.Context, id string, amount float64) error {
	return c.call(ctx, "", http.MethodPost, "/api/v1/usage/reservations/"+id+"/finalize", map[string]any{"actual_amount": int64(amount)}, nil)
}

func (c *LPBSCredits) call(ctx context.Context, user, method, path string, payload any, output any) error {
	if strings.TrimSpace(c.BaseURL) == "" {
		return fmt.Errorf("landing-page-business-suite base URL is not configured")
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, method, strings.TrimRight(c.BaseURL, "/")+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	if c.Token != nil {
		token, e := c.Token(ctx)
		if e != nil {
			return fmt.Errorf("resolve cross-scenario token: %w", e)
		}
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
	}
	if user != "" {
		req.Header.Set("X-Vrooli-Identity", user)
	}
	req.Header.Set("Content-Type", "application/json")
	client := c.Client
	if client == nil {
		client = &http.Client{}
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("landing-page-business-suite request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusPaymentRequired {
		return meter.ErrInsufficientCredits
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("landing-page-business-suite returned %s", resp.Status)
	}
	if output != nil {
		return json.NewDecoder(resp.Body).Decode(output)
	}
	return nil
}
