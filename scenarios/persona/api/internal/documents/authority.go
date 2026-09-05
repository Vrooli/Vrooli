package documents

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type unavailableAuthority struct{}

func (unavailableAuthority) Check(context.Context) error { return ErrDocumentAuthorityUnavailable }
func (unavailableAuthority) Release(context.Context, string, string) (string, error) {
	return "", ErrDocumentAuthorityUnavailable
}
func NewUnavailableAuthority() Authority { return unavailableAuthority{} }

// HTTPAuthority is deliberately metadata-only: its release request contains
// document and handoff identifiers, never document content.
type HTTPAuthority struct {
	BaseURL string
	Client  *http.Client
}

func (a HTTPAuthority) Check(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(a.BaseURL, "/")+"/health", nil)
	if err != nil {
		return err
	}
	client := a.Client
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("document-manager health status %d", resp.StatusCode)
	}
	return nil
}

func (a HTTPAuthority) Release(ctx context.Context, documentID, handoffID string) (string, error) {
	payload, err := json.Marshal(map[string]string{"document_id": documentID, "handoff_id": handoffID})
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(a.BaseURL, "/")+"/api/v1/custody/release", bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	client := a.Client
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("document-manager release status %d", resp.StatusCode)
	}
	var response struct {
		ReleaseID string `json:"release_id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", err
	}
	if strings.TrimSpace(response.ReleaseID) == "" {
		return "", errors.New("document-manager returned no release id")
	}
	return response.ReleaseID, nil
}
