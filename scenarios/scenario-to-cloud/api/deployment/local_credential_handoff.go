package deployment

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/vrooli/api-core/discovery"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
	"scenario-to-cloud/domain"
)

type localRemoteProfile struct {
	ID  int64  `json:"id"`
	Tag string `json:"tag"`
}

type localRemoteProfileList struct {
	Profiles []localRemoteProfile `json:"profiles"`
}

// localCredentialHandoff reconciles deployment-generated credentials into a
// local consumer over service discovery. It deliberately has no target/VPS
// transport and never persists or logs the credential value.
func (o *Orchestrator) localCredentialHandoff(ctx context.Context, manifest domain.CloudManifest, deploymentID string, values map[string]string) error {
	for _, handoff := range manifest.LocalCredentialHandoffs {
		if err := reconcileLocalRemoteProfile(ctx, manifest, handoff, values); err != nil {
			return fmt.Errorf("consumer %q: %w", handoff.ConsumerScenario, err)
		}
	}
	return nil
}

func reconcileLocalRemoteProfile(ctx context.Context, manifest domain.CloudManifest, handoff domain.LocalCredentialHandoff, values map[string]string) error {
	if err := validateHandoff(handoff); err != nil {
		return err
	}
	secret := resolveHandoffSecret(manifest, handoff.SecretID, values)
	if secret == "" {
		return fmt.Errorf("deployment credential %q was not materialized", handoff.SecretID)
	}
	identity, err := credentialauthority.ParseIdentity(handoff.AuthDescriptor.LogicalID)
	if err != nil {
		return fmt.Errorf("parse local auth descriptor: %w", err)
	}
	authority, err := credentialauthority.Default()
	if err != nil {
		return fmt.Errorf("load local credential authority: %w", err)
	}
	serviceToken, err := authority.Require(identity, handoff.AuthDescriptor.Field)
	if err != nil {
		return fmt.Errorf("resolve local auth descriptor: %w", err)
	}
	base, err := discovery.ResolveScenarioURLDefault(ctx, handoff.ConsumerScenario)
	if err != nil {
		return fmt.Errorf("discover local consumer: %w", err)
	}
	endpoint, err := url.JoinPath(strings.TrimRight(base, "/"), handoff.ConsumerPath)
	if err != nil {
		return fmt.Errorf("build local handoff endpoint: %w", err)
	}
	listReq, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("build profile lookup request: %w", err)
	}
	listReq.Header.Set("Authorization", "Bearer "+serviceToken)
	listResp, err := http.DefaultClient.Do(listReq)
	if err != nil {
		return fmt.Errorf("lookup local remote profile: %w", err)
	}
	defer listResp.Body.Close()
	if listResp.StatusCode != http.StatusOK {
		return fmt.Errorf("lookup local remote profile returned %s", listResp.Status)
	}
	var listed localRemoteProfileList
	if err := json.NewDecoder(listResp.Body).Decode(&listed); err != nil {
		return fmt.Errorf("decode local remote profile list: %w", err)
	}

	payload := map[string]any{
		"tag": handoff.ProfileTag, "label": handoff.ProfileLabel,
		"api_base": handoff.APIBase, "auth_mode": "service",
		"remote_service_secret": secret,
	}
	method, target := http.MethodPost, endpoint
	for _, profile := range listed.Profiles {
		if profile.Tag == handoff.ProfileTag {
			method = http.MethodPut
			target, err = url.JoinPath(endpoint, fmt.Sprint(profile.ID))
			if err != nil {
				return fmt.Errorf("build profile update endpoint: %w", err)
			}
			break
		}
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode local remote profile: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, method, target, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build local profile reconcile request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+serviceToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("reconcile local remote profile: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("reconcile local remote profile returned %s", resp.Status)
	}
	return nil
}

func resolveHandoffSecret(manifest domain.CloudManifest, secretID string, values map[string]string) string {
	if value := strings.TrimSpace(values[secretID]); value != "" {
		return value
	}
	if manifest.Secrets == nil {
		return ""
	}
	for _, plan := range manifest.Secrets.BundleSecrets {
		if plan.ID != secretID && plan.Target.Name != secretID && (plan.Descriptor == nil || plan.Descriptor.Field != secretID) {
			continue
		}
		if value := strings.TrimSpace(values[plan.ID]); value != "" {
			return value
		}
		if value := strings.TrimSpace(values[plan.Target.Name]); value != "" {
			return value
		}
	}
	return ""
}

func validateHandoff(handoff domain.LocalCredentialHandoff) error {
	if !strings.HasPrefix(handoff.ConsumerPath, "/api/v1/admin/remote-profiles") || strings.Contains(handoff.ConsumerPath, "..") {
		return fmt.Errorf("consumer path is not an approved local profile endpoint")
	}
	parsed, err := url.Parse(handoff.APIBase)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return fmt.Errorf("deployment API base is invalid")
	}
	return nil
}
