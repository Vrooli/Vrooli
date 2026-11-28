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
	"github.com/vrooli/api-core/lpbsidentity"
	"github.com/vrooli/api-core/servicetoken"
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

const lpbsRemoteProfileSecretPlanID = "remote-profile-service-secret"

// localCredentialHandoff reconciles deployment-generated credentials into a
// local consumer over service discovery. It deliberately has no target/VPS
// transport and never persists or logs the credential value.
func (o *Orchestrator) localCredentialHandoff(ctx context.Context, manifest domain.CloudManifest, deploymentID string, values map[string]string) error {
	for _, handoff := range manifest.LocalCredentialHandoffs {
		if err := reconcileLocalRemoteProfile(ctx, deploymentID, manifest, handoff, values); err != nil {
			return fmt.Errorf("consumer %q: %w", handoff.ConsumerScenario, err)
		}
	}
	return nil
}

func reconcileLocalRemoteProfile(ctx context.Context, deploymentID string, manifest domain.CloudManifest, handoff domain.LocalCredentialHandoff, values map[string]string) error {
	if err := validateHandoff(handoff); err != nil {
		return err
	}
	if err := validateHandoffSecretPlan(manifest, handoff); err != nil {
		return err
	}
	secret := resolveHandoffSecret(manifest, handoff.SecretID, values)
	if secret == "" {
		return fmt.Errorf("deployment credential %q was not materialized", handoff.SecretID)
	}
	authority, err := credentialauthority.Default()
	if err != nil {
		return fmt.Errorf("load local credential authority: %w", err)
	}
	scopedIdentity, err := credentialauthority.ParseIdentity(lpbsidentity.ScenarioToCloudRemoteProfiles.CredentialIdentity("scenario-to-cloud"))
	if err != nil {
		return fmt.Errorf("parse scoped LPBS principal identity")
	}
	scopedTokens := lpbsidentity.ScenarioToCloudRemoteProfiles.NewTokens("scenario-to-cloud", func(ctx context.Context) (string, error) {
		return authority.Require(scopedIdentity, "credential")
	}, servicetoken.WithExchanger(servicetoken.AuthenticatorExchanger(http.DefaultClient, func(ctx context.Context) (string, error) {
		return discovery.ResolveScenarioURLDefault(ctx, "scenario-authenticator")
	})))
	authorization := func(ctx context.Context) (string, error) {
		status := authority.Status(scopedIdentity, "credential")
		legacyServiceToken := ""
		if legacy := handoff.LegacyAuthDescriptor; legacy != nil && legacy.LogicalID == "vrooli/landing-page-business-suite" && legacy.Field == "service-secret" && string(status.ProviderState) == "available" && !status.Configured {
			if identity, parseErr := credentialauthority.ParseIdentity(legacy.LogicalID); parseErr == nil {
				legacyServiceToken, _ = authority.Require(identity, legacy.Field)
			}
		}
		return resolveLPBSServiceAuthorization(ctx, string(status.ProviderState) == "available", status.Configured, scopedTokens.Authorization, legacyServiceToken)
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
	listAuthorization, err := authorization(ctx)
	if err != nil {
		return err
	}
	listReq.Header.Set("Authorization", listAuthorization)
	listReq.Header.Set("X-Vrooli-Deployment-ID", deploymentID)
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
	requestAuthorization, err := authorization(ctx)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", requestAuthorization)
	req.Header.Set("X-Vrooli-Deployment-ID", deploymentID)
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

// resolveLPBSServiceAuthorization uses the versioned scoped principal whenever
// it is configured. The shared service bearer is a migration fallback only for
// a healthy authority that confirms the scoped principal is not issued.
func resolveLPBSServiceAuthorization(ctx context.Context, authorityAvailable, scopedConfigured bool, scoped func(context.Context) (string, error), legacyToken string) (string, error) {
	if !authorityAvailable {
		return "", fmt.Errorf("LPBS service identity authority is unavailable")
	}
	if !scopedConfigured {
		if strings.TrimSpace(legacyToken) == "" {
			return "", fmt.Errorf("LPBS scoped and legacy service identities are unavailable")
		}
		return "Bearer " + strings.TrimSpace(legacyToken), nil
	}
	if scoped == nil {
		return "", fmt.Errorf("LPBS scoped service identity is unavailable")
	}
	header, err := scoped(ctx)
	if err != nil || strings.TrimSpace(header) == "" {
		return "", fmt.Errorf("exchange LPBS scoped service identity")
	}
	return header, nil
}

func resolveHandoffSecret(manifest domain.CloudManifest, secretID string, values map[string]string) string {
	if manifest.Secrets == nil {
		return ""
	}
	var found *domain.BundleSecretPlan
	for _, plan := range manifest.Secrets.BundleSecrets {
		if plan.ID != secretID {
			continue
		}
		if found != nil {
			return ""
		}
		copy := plan
		found = &copy
	}
	if found == nil || found.Class != domain.SecretClassPerInstallGenerated || found.Descriptor == nil || found.Descriptor.LogicalID != "vrooli/landing-page-business-suite" || found.Descriptor.Field != "service-secret" || found.Target.Type != "env" || found.Target.Name != "LPBS_SERVICE_SECRET" {
		return ""
	}
	return strings.TrimSpace(values[secretID])
}

func validateHandoffSecretPlan(manifest domain.CloudManifest, handoff domain.LocalCredentialHandoff) error {
	if handoff.SecretID != lpbsRemoteProfileSecretPlanID {
		return fmt.Errorf("handoff must use the target-bound %s credential plan", lpbsRemoteProfileSecretPlanID)
	}
	if manifest.Secrets == nil {
		return fmt.Errorf("handoff destination credential plan is missing")
	}
	count := 0
	for _, plan := range manifest.Secrets.BundleSecrets {
		if plan.ID != handoff.SecretID {
			continue
		}
		count++
		if plan.Class != domain.SecretClassPerInstallGenerated || !plan.Required {
			return fmt.Errorf("handoff destination credential plan must be a required generated credential")
		}
		if plan.Descriptor == nil || plan.Descriptor.LogicalID != "vrooli/landing-page-business-suite" || plan.Descriptor.Field != "service-secret" {
			return fmt.Errorf("handoff destination credential plan must bind LPBS service-secret on its deployment target")
		}
		if plan.Target.Type != "env" || plan.Target.Name != "LPBS_SERVICE_SECRET" {
			return fmt.Errorf("handoff destination credential plan must target the LPBS service credential binding")
		}
		if plan.Generator == nil || plan.Generator["type"] != "random" || plan.Generator["charset"] != "hex" || plan.Generator["length"] != float64(64) {
			return fmt.Errorf("handoff destination credential plan must use the declared 64-character hex generator")
		}
	}
	if count != 1 {
		return fmt.Errorf("handoff destination credential plan must be declared exactly once")
	}
	return nil
}

func validateHandoff(handoff domain.LocalCredentialHandoff) error {
	expectedIdentity := lpbsidentity.ScenarioToCloudRemoteProfiles.CredentialIdentity("scenario-to-cloud")
	if handoff.AuthDescriptor.LogicalID != expectedIdentity || handoff.AuthDescriptor.Field != "credential" {
		return fmt.Errorf("local auth descriptor must name the scoped scenario-to-cloud remote-profile principal")
	}
	if legacy := handoff.LegacyAuthDescriptor; legacy != nil && (legacy.LogicalID != "vrooli/landing-page-business-suite" || legacy.Field != "service-secret") {
		return fmt.Errorf("legacy auth descriptor must name the LPBS service-secret migration bridge")
	}
	if (handoff.ConsumerPath != "/api/v1/admin/remote-profiles" && !strings.HasPrefix(handoff.ConsumerPath, "/api/v1/admin/remote-profiles/")) || strings.Contains(handoff.ConsumerPath, "..") {
		return fmt.Errorf("consumer path is not an approved local profile endpoint")
	}
	parsed, err := url.Parse(handoff.APIBase)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return fmt.Errorf("deployment API base is invalid")
	}
	return nil
}
