package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/vrooli/secrets-manager-native-host/protocol"
)

const (
	nativeHostBaseURL   = "SECRETS_MANAGER_NATIVE_HOST_URL"
	nativeHostTransport = "SECRETS_MANAGER_NATIVE_HOST_TOKEN"
)

type httpAuthority struct {
	baseURL        string
	transportToken string
	client         *http.Client
	mu             sync.Mutex
	sessions       map[string]httpAuthoritySession
}

type httpAuthoritySession struct {
	OwnerToken string
	Workspace  string
	Origin     string
}

func newHTTPAuthorityFromEnv() protocol.Authority {
	baseURL := strings.TrimRight(strings.TrimSpace(os.Getenv(nativeHostBaseURL)), "/")
	transportToken := strings.TrimSpace(os.Getenv(nativeHostTransport))
	parsed, err := url.Parse(baseURL)
	if err != nil || parsed.Host == "" || parsed.User != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || transportToken == "" {
		return nil
	}
	return &httpAuthority{baseURL: baseURL, transportToken: transportToken, client: &http.Client{Timeout: 10 * time.Second}, sessions: make(map[string]httpAuthoritySession)}
}

func (a *httpAuthority) Enroll(ctx context.Context, request protocol.EnrollmentRequest, ownerToken string) error {
	if strings.TrimSpace(ownerToken) == "" {
		return protocol.ErrUnauthorized
	}
	return a.call(ctx, ownerToken, request.WorkspaceID, "/api/v1/native-host/enroll", map[string]string{
		"extension_id": request.ExtensionID,
		"origin":       request.Origin,
	}, nil)
}

func (a *httpAuthority) Metadata(ctx context.Context, request protocol.MetadataRequest, ownerToken string) ([]protocol.MetadataRecord, error) {
	if strings.TrimSpace(ownerToken) == "" {
		return nil, protocol.ErrUnauthorized
	}
	input := map[string]any{
		"extension_id": request.ExtensionID,
		"workspace_id": request.WorkspaceID,
		"origin":       request.Scope.Origin,
		"tab_id":       request.Scope.TabID,
		"frame_id":     request.Scope.FrameID,
		"document_id":  request.Scope.DocumentID,
	}
	var response struct {
		Items []protocol.MetadataRecord `json:"items"`
	}
	if err := a.call(ctx, ownerToken, request.WorkspaceID, "/api/v1/native-host/metadata", input, &response); err != nil {
		return nil, err
	}
	return response.Items, nil
}

func (a *httpAuthority) Save(ctx context.Context, request protocol.SaveRequest, ownerToken string) (protocol.MetadataRecord, error) {
	return a.write(ctx, "/api/v1/native-host/save", map[string]any{
		"extension_id": request.ExtensionID, "workspace_id": request.WorkspaceID, "vault_id": request.VaultID,
		"origin": request.Scope.Origin, "tab_id": request.Scope.TabID, "frame_id": request.Scope.FrameID,
		"document_id": request.Scope.DocumentID, "name": request.Name, "username": request.Username,
		"uri": request.URI, "password": request.Password, "totp": request.TOTP,
	}, ownerToken, request.WorkspaceID)
}

func (a *httpAuthority) Update(ctx context.Context, request protocol.UpdateRequest, ownerToken string) (protocol.MetadataRecord, error) {
	return a.write(ctx, "/api/v1/native-host/update", map[string]any{
		"extension_id": request.ExtensionID, "workspace_id": request.WorkspaceID, "vault_id": request.VaultID,
		"origin": request.Scope.Origin, "tab_id": request.Scope.TabID, "frame_id": request.Scope.FrameID,
		"document_id": request.Scope.DocumentID, "item_id": request.ItemID, "item_revision": request.Revision,
		"name": request.Name, "username": request.Username, "uri": request.URI, "password": request.Password, "totp": request.TOTP,
	}, ownerToken, request.WorkspaceID)
}

func (a *httpAuthority) write(ctx context.Context, path string, input any, ownerToken, workspace string) (protocol.MetadataRecord, error) {
	var response protocol.MetadataRecord
	if err := a.call(ctx, ownerToken, workspace, path, input, &response); err != nil {
		return protocol.MetadataRecord{}, err
	}
	return response, nil
}

func (a *httpAuthority) Unlock(ctx context.Context, request protocol.UnlockRequest, ownerToken string) error {
	if strings.TrimSpace(ownerToken) == "" {
		return protocol.ErrUnauthorized
	}
	input := map[string]any{
		"extension_id":  request.ExtensionID,
		"workspace_id":  request.WorkspaceID,
		"grant_id":      request.GrantID,
		"origin":        request.Scope.Origin,
		"tab_id":        request.Scope.TabID,
		"frame_id":      request.Scope.FrameID,
		"document_id":   request.Scope.DocumentID,
		"item_id":       request.Scope.ItemID,
		"item_revision": request.Scope.Revision,
		"field":         request.Field,
	}
	if err := a.call(ctx, ownerToken, request.WorkspaceID, "/api/v1/native-host/unlock", input, nil); err != nil {
		return err
	}
	a.mu.Lock()
	a.sessions[request.SessionID] = httpAuthoritySession{OwnerToken: ownerToken, Workspace: request.WorkspaceID, Origin: request.Scope.Origin}
	a.mu.Unlock()
	return nil
}

func (a *httpAuthority) Fill(ctx context.Context, request protocol.FillRequest) (string, error) {
	a.mu.Lock()
	session, ok := a.sessions[request.SessionID]
	a.mu.Unlock()
	if !ok {
		return "", protocol.ErrUnauthorized
	}
	input := map[string]any{
		"extension_id":  request.ExtensionID,
		"workspace_id":  request.WorkspaceID,
		"grant_id":      request.GrantID,
		"origin":        request.Scope.Origin,
		"tab_id":        request.Scope.TabID,
		"frame_id":      request.Scope.FrameID,
		"document_id":   request.Scope.DocumentID,
		"item_id":       request.Scope.ItemID,
		"item_revision": request.Scope.Revision,
		"field":         request.Field,
	}
	var response struct {
		Credential string `json:"credential"`
	}
	if err := a.call(ctx, session.OwnerToken, session.Workspace, "/api/v1/native-host/fill", input, &response); err != nil {
		return "", err
	}
	if response.Credential == "" {
		return "", errors.New("native authority returned an empty credential")
	}
	return response.Credential, nil
}

func (a *httpAuthority) Revoke(ctx context.Context, request protocol.FillRequest) error {
	a.mu.Lock()
	session, ok := a.sessions[request.SessionID]
	delete(a.sessions, request.SessionID)
	a.mu.Unlock()
	if !ok {
		return protocol.ErrUnauthorized
	}
	return a.call(ctx, session.OwnerToken, session.Workspace, "/api/v1/native-host/revoke", map[string]string{
		"extension_id": request.ExtensionID,
		"origin":       session.Origin,
	}, nil)
}

func (a *httpAuthority) call(ctx context.Context, ownerToken, workspace, path string, input any, output any) error {
	payload, err := json.Marshal(input)
	if err != nil {
		return fmt.Errorf("native authority request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.baseURL+path, strings.NewReader(string(payload)))
	if err != nil {
		return fmt.Errorf("native authority request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+ownerToken)
	req.Header.Set("X-Workspace-ID", workspace)
	req.Header.Set("X-Secrets-Manager-Native-Host-Token", a.transportToken)
	response, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("native authority unavailable: %w", protocol.ErrAuthorityUnavailable)
	}
	defer response.Body.Close()
	body, readErr := io.ReadAll(io.LimitReader(response.Body, 64*1024+1))
	if readErr != nil || len(body) > 64*1024 {
		return fmt.Errorf("native authority unavailable: %w", protocol.ErrAuthorityUnavailable)
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		if response.StatusCode == http.StatusUnauthorized || response.StatusCode == http.StatusForbidden {
			return protocol.ErrUnauthorized
		}
		if response.StatusCode >= 500 || response.StatusCode == http.StatusNotFound {
			return protocol.ErrAuthorityUnavailable
		}
		return fmt.Errorf("native authority rejected request: status %d", response.StatusCode)
	}
	if output != nil && len(body) > 0 {
		if err := json.Unmarshal(body, output); err != nil {
			return fmt.Errorf("native authority returned invalid response: %w", protocol.ErrAuthorityUnavailable)
		}
	}
	return nil
}
