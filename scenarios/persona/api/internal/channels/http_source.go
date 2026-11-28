package channels

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// HTTPSource is the production adapter seam. It forwards references to the
// owning channel and never accepts or persists a credential value.
type HTTPSource struct {
	BaseURL string
	Client  *http.Client
}

func (s HTTPSource) Check(ctx context.Context, channel Channel) error {
	var response map[string]any
	return s.post(ctx, "/v1/health", map[string]string{"persona_id": channel.PersonaID, "channel_id": channel.ID, "address": channel.Address, "credential_ref": channel.CredentialRef}, &response)
}

func (s HTTPSource) Retrieve(ctx context.Context, channel Channel, purpose string) (Code, error) {
	var response struct {
		Code      string `json:"code"`
		ExpiresAt string `json:"expires_at"`
		Adapter   string `json:"adapter"`
	}
	if err := s.post(ctx, "/v1/code", map[string]string{"persona_id": channel.PersonaID, "channel_id": channel.ID, "address": channel.Address, "credential_ref": channel.CredentialRef, "purpose": purpose}, &response); err != nil {
		return Code{}, err
	}
	expires, err := time.Parse(time.RFC3339, response.ExpiresAt)
	if err != nil {
		return Code{}, fmt.Errorf("parse adapter expiry: %w", err)
	}
	return Code{Value: response.Code, ExpiresAt: expires.UTC(), Adapter: response.Adapter}, nil
}

func (s HTTPSource) Send(ctx context.Context, channel Channel, in MessageInput) (Message, error) {
	var response struct {
		MessageID string `json:"message_id"`
	}
	if err := s.post(ctx, "/v1/send", map[string]string{"persona_id": channel.PersonaID, "channel_id": channel.ID, "from_address": channel.Address, "recipient": in.Recipient, "subject": in.Subject, "body": in.Body}, &response); err != nil {
		return Message{}, err
	}
	return Message{ID: response.MessageID, FromAddress: channel.Address}, nil
}

func (s HTTPSource) post(ctx context.Context, path string, payload map[string]string, response any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(s.BaseURL, "/")+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	client := s.Client
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return fmt.Errorf("channel adapter status %d", res.StatusCode)
	}
	return json.NewDecoder(res.Body).Decode(response)
}
