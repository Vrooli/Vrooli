package resources

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/tuning"
)

// VaultCredentialIssuer turns an already-authorized broker lease into a
// Vault child token. The management token is obtained through a caller-owned
// function at issuance time and is neither persisted by the broker nor sent to
// an application. Each app receives a policy whose path prefix derives from a
// one-way scope hash, keeping app identifiers out of Vault policy names.
type VaultCredentialIssuer struct {
	ManagementToken func(ManagedInstance) (string, error)
	HTTPClient      *http.Client
	Now             func() time.Time
	Mount           string
}

func (i VaultCredentialIssuer) IssueScopedCredential(instance ManagedInstance, lease Lease) (ScopedCredential, error) {
	if instance.Resource != vaultBootstrapVault {
		return ScopedCredential{}, fmt.Errorf("Vault credential issuer cannot issue credentials for %s", instance.Resource)
	}
	if !isLoopbackManagedEndpoint(instance.Endpoint) {
		return ScopedCredential{}, fmt.Errorf("Vault credential issuance requires a verified loopback endpoint")
	}
	if i.ManagementToken == nil {
		return ScopedCredential{}, fmt.Errorf("Vault credential issuer requires a management token source")
	}
	now := time.Now
	if i.Now != nil {
		now = i.Now
	}
	ttl := time.Until(lease.ExpiresAt)
	if i.Now != nil {
		ttl = lease.ExpiresAt.Sub(now())
	}
	if ttl <= 0 {
		return ScopedCredential{}, fmt.Errorf("cannot issue Vault credential for expired lease")
	}
	managementToken, err := i.ManagementToken(instance)
	if err != nil {
		return ScopedCredential{}, fmt.Errorf("get Vault management token: %w", err)
	}
	if strings.TrimSpace(managementToken) == "" {
		return ScopedCredential{}, fmt.Errorf("Vault management token source returned an empty token")
	}
	client := i.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: tuning.ControlPlaneClientTimeout()}
	}
	policyName, policy := i.policyForScope(lease.Scope)
	if err := vaultJSONRequest(client, instance.Endpoint, "/v1/sys/policies/acl/"+policyName, managementToken, http.MethodPut, map[string]string{"policy": policy}, nil); err != nil {
		return ScopedCredential{}, fmt.Errorf("write scoped Vault policy: %w", err)
	}
	var tokenResponse struct {
		Auth struct {
			ClientToken string `json:"client_token"`
		} `json:"auth"`
	}
	if err := vaultJSONRequest(client, instance.Endpoint, "/v1/auth/token/create", managementToken, http.MethodPost, map[string]any{
		"policies":     []string{policyName},
		"ttl":          ttl.Round(time.Second).String(),
		"display_name": "vrooli-lease-" + lease.ID,
		"no_parent":    true,
	}, &tokenResponse); err != nil {
		return ScopedCredential{}, fmt.Errorf("create scoped Vault token: %w", err)
	}
	if strings.TrimSpace(tokenResponse.Auth.ClientToken) == "" {
		return ScopedCredential{}, fmt.Errorf("Vault token response omitted client token")
	}
	return ScopedCredential{LeaseID: lease.ID, Resource: "vault", Scope: lease.Scope, ExpiresAt: lease.ExpiresAt, Credential: tokenResponse.Auth.ClientToken}, nil
}

func (i VaultCredentialIssuer) policyForScope(scope string) (string, string) {
	sum := sha256.Sum256([]byte(scope))
	scopeID := hex.EncodeToString(sum[:16])
	mount := strings.Trim(strings.TrimSpace(i.Mount), "/")
	if mount == "" {
		mount = "secret"
	}
	policyName := "vrooli-app-" + scopeID
	prefix := "apps/" + scopeID
	policy := fmt.Sprintf("path %q { capabilities = [\"create\", \"read\", \"update\", \"delete\", \"list\"] }\npath %q { capabilities = [\"list\"] }\n", mount+"/data/"+prefix+"/*", mount+"/metadata/"+prefix+"/*")
	return policyName, policy
}

func vaultJSONRequest(client *http.Client, endpoint, path, token, method string, input, output any) error {
	base, err := url.Parse(endpoint)
	if err != nil {
		return err
	}
	relative, err := url.Parse(path)
	if err != nil {
		return err
	}
	target := base.ResolveReference(relative)
	data, err := json.Marshal(input)
	if err != nil {
		return err
	}
	request, err := http.NewRequest(method, target.String(), bytes.NewReader(data))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Vault-Token", token)
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("Vault returned %s", response.Status)
	}
	if output == nil {
		return nil
	}
	return json.NewDecoder(response.Body).Decode(output)
}
