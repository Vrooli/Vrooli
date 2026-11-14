// Responsibility: authenticated LPBS model discovery and device-key custody.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"web-console/handlers/modelselection"
)

// codingKey is issued by LPBS. Web Console never creates a billing credential
// or interprets a browser-supplied account as authority to spend.
type codingKey struct {
	ID        string `json:"id"`
	Key       string `json:"key"`
	ExpiresAt int64  `json:"expires_at"`
}

type codingModel struct {
	ID                  string `json:"id"`
	Name                string `json:"name"`
	ContextWindowTokens int64  `json:"context_window_tokens"`
}

// codingAuthority talks only to the authenticated landing-page door. HTTP
// failures retain their status so funding refusal cannot become a retry.
type codingAuthorityRefusal int

func (e codingAuthorityRefusal) Error() string {
	return fmt.Sprintf("coding authority refused the operation (HTTP %d)", int(e))
}

type codingAuthority struct {
	BaseURL string
	Client  *http.Client
}

func (a codingAuthority) request(ctx context.Context, method, path, token string, input, output any) error {
	var body io.Reader
	if input != nil {
		data, err := json.Marshal(input)
		if err != nil {
			return err
		}
		body = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, strings.TrimRight(a.BaseURL, "/")+"/api/v1/ai"+path, body)
	if err != nil {
		return errors.New("invalid coding authority URL")
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	client := a.Client
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	response, err := client.Do(req)
	if err != nil {
		return errors.New("coding authority unavailable")
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return codingAuthorityRefusal(response.StatusCode)
	}
	if output == nil {
		_, err = io.Copy(io.Discard, io.LimitReader(response.Body, 1<<20))
		return err
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(output); err != nil {
		return errors.New("invalid coding authority response")
	}
	return nil
}

func (a codingAuthority) models(ctx context.Context, access, harness string) ([]codingModel, string, error) {
	var response struct {
		Data   []codingModel `json:"data"`
		Reason string        `json:"reason"`
	}
	err := a.request(ctx, http.MethodGet, "/models?harness="+url.QueryEscape(harness), access, nil, &response)
	return response.Data, response.Reason, err
}

const codingSessionCap int64 = 5000

func (s *Server) codingAuthorityDevice(ctx context.Context, account string, cap int64) (codingAuthority, codingKey, error) {
	if s == nil || s.subscriptionResolver == nil || strings.TrimSpace(account) == "" || cap <= 0 {
		return codingAuthority{}, codingKey{}, errors.New("choose a linked billing account to use Vrooli credits")
	}
	access, err := s.subscriptionResolver.Resolve(ctx)
	if err != nil {
		return codingAuthority{}, codingKey{}, errors.New("sign in to Vrooli to use credits")
	}
	authority := codingAuthority{BaseURL: s.subscriptionResolver.LPBSBaseURL, Client: s.subscriptionResolver.HTTPClient}
	var device codingKey
	err = authority.request(ctx, http.MethodPost, "/keys/device", access.AccessToken, map[string]any{
		"business_account_id": account, "app_bundle_key": "web-console", "max_credits": cap,
	}, &device)
	if err == nil && (device.ID == "" || device.Key == "" || device.ExpiresAt <= time.Now().Unix()) {
		err = errors.New("coding authority returned an invalid device credential")
	}
	return authority, device, err
}

// GetAccountRoleModelChoices composes resource-owned own-account choices with
// the door's current eligible paid choices. No local boolean can approve a
// harness, supplier or model. Issuance verifies account membership in LPBS.
func (s *Server) GetAccountRoleModelChoices(ctx context.Context, agent, target, command, account string) modelselection.Result {
	result := s.GetRoleModelChoices(ctx, agent, target, command)
	if target != "" && target != "local" || result.CommandLiteral {
		return result
	}
	authority, device, err := s.codingAuthorityDevice(ctx, account, codingSessionCap)
	if err != nil {
		result.Detail = err.Error()
		return result
	}
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = authority.revoke(cleanup, device.Key, device.ID)
	}()
	models, reason, err := authority.models(ctx, device.Key, agent)
	if err != nil {
		result.Detail = err.Error()
		return result
	}
	if len(models) == 0 {
		result.Detail = reason
		return result
	}
	for _, model := range models {
		if !safeNativeModelArg.MatchString(model.ID) {
			continue
		}
		label := model.Name
		if label == "" {
			label = model.ID
		}
		result.Choices = append(result.Choices, modelselection.Choice{
			Model: model.ID, DisplayName: label, Source: "vrooli-credits", Provider: "vrooli",
			LaunchModel: model.ID, SourceLabel: "Vrooli credits", Metered: true, ContextWindowTokens: model.ContextWindowTokens, PolicyCurrent: true,
		})
	}
	return result
}

func (a codingAuthority) revoke(ctx context.Context, authority, id string) error {
	err := a.request(ctx, http.MethodDelete, "/keys/"+url.PathEscape(id), authority, nil, nil)
	// A known issued parent that no longer authenticates cannot authorize any child.
	var refusal codingAuthorityRefusal
	if errors.As(err, &refusal) && int(refusal) == http.StatusUnauthorized {
		return nil
	}
	return err
}
