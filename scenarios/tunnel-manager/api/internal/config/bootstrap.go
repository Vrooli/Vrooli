package config

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
)

// CloudflareBootstrapAPI is the narrow adopt-or-create seam. It keeps the
// bootstrap policy testable without making the credential authority or the
// Cloudflare HTTP transport part of the policy itself.
type CloudflareBootstrapAPI interface {
	VerifyToken(context.Context, string) error
	ListAccounts(context.Context, string) ([]CloudflareAccount, error)
	ListTunnels(context.Context, string, string) ([]CloudflareTunnel, error)
	CreateTunnel(context.Context, string, string, string) (CloudflareTunnel, error)
	ConnectorToken(context.Context, string, string, string) (string, error)
}

type CloudflareAccount struct{ ID string }

type CloudflareTunnel struct {
	ID      string
	Name    string
	Deleted bool
}

type BootstrapRequest struct {
	APIToken               string
	AccountID              string
	TunnelID               string
	TunnelName             string
	ExistingConnectorToken string
	DryRun                 bool
}

type BootstrapResult struct {
	AccountID string `json:"account_id"`
	TunnelID  string `json:"tunnel_id"`
	Adopted   bool   `json:"adopted"`
	Created   bool   `json:"created"`
	Written   bool   `json:"written"`
}

// BootstrapCloudflare validates the operator token, adopts a supplied/live
// tunnel when possible, and writes the complete credential set only after the
// connector token has been fetched. A failed run therefore cannot leave a
// half-provisioned derived set behind.
func BootstrapCloudflare(ctx context.Context, api CloudflareBootstrapAPI, authority CredentialAuthority, request BootstrapRequest) (BootstrapResult, error) {
	if api == nil || authority == nil {
		return BootstrapResult{}, fmt.Errorf("Cloudflare bootstrap requires an API and credential authority")
	}
	token := strings.TrimSpace(request.APIToken)
	if token == "" {
		return BootstrapResult{}, fmt.Errorf("Cloudflare API token is required")
	}
	identity, err := credentialauthority.ParseIdentity(cloudflareCredentialID)
	if err != nil {
		return BootstrapResult{}, err
	}
	// A connector token is an adoption hint, not an operator input. Reading it
	// from the authority lets a re-bootstrap recover the account/tunnel
	// coordinates from the already-running connector without ever decoding the
	// unrelated API token or exposing either secret in the request surface.
	connectorHint := strings.TrimSpace(request.ExistingConnectorToken)
	if connectorHint == "" {
		connectorHint, _ = authority.Resolve(identity, "cloudflare-connector-token")
		connectorHint = strings.TrimSpace(connectorHint)
	}
	if err := api.VerifyToken(ctx, token); err != nil {
		return BootstrapResult{}, fmt.Errorf("verify Cloudflare API token: %w", err)
	}
	accounts, err := api.ListAccounts(ctx, token)
	if err != nil {
		return BootstrapResult{}, fmt.Errorf("list Cloudflare accounts: %w", err)
	}
	accountID := strings.TrimSpace(request.AccountID)
	decodedAccountID, decodedTunnelID := decodeCloudflareTunnelToken(connectorHint)
	if accountID == "" {
		accountID = decodedAccountID
	}
	if accountID == "" {
		if len(accounts) != 1 {
			return BootstrapResult{}, fmt.Errorf("Cloudflare bootstrap found %d accounts; supply an account id to disambiguate", len(accounts))
		}
		accountID = strings.TrimSpace(accounts[0].ID)
	}
	if accountID == "" {
		return BootstrapResult{}, fmt.Errorf("Cloudflare account id is empty")
	}
	accountKnown := false
	for _, account := range accounts {
		if strings.EqualFold(strings.TrimSpace(account.ID), accountID) {
			accountKnown = true
			break
		}
	}
	if !accountKnown {
		return BootstrapResult{}, fmt.Errorf("Cloudflare account %q is not visible to the supplied token", accountID)
	}

	tunnelID := strings.TrimSpace(request.TunnelID)
	if tunnelID == "" {
		tunnelID = decodedTunnelID
	}
	adopted := false
	created := false
	if tunnelID != "" {
		tunnels, listErr := api.ListTunnels(ctx, token, accountID)
		if listErr != nil {
			return BootstrapResult{}, fmt.Errorf("confirm Cloudflare tunnel: %w", listErr)
		}
		confirmed := false
		for _, tunnel := range tunnels {
			if !tunnel.Deleted && strings.EqualFold(strings.TrimSpace(tunnel.ID), tunnelID) {
				confirmed = true
				adopted = true
				break
			}
		}
		if !confirmed {
			return BootstrapResult{}, fmt.Errorf("Cloudflare tunnel %q is not visible in account %q", tunnelID, accountID)
		}
	}
	if tunnelID == "" {
		name := strings.TrimSpace(request.TunnelName)
		if name == "" {
			name = "vrooli"
		}
		tunnels, listErr := api.ListTunnels(ctx, token, accountID)
		if listErr != nil {
			return BootstrapResult{}, fmt.Errorf("list Cloudflare tunnels: %w", listErr)
		}
		for _, tunnel := range tunnels {
			if !tunnel.Deleted && strings.EqualFold(strings.TrimSpace(tunnel.Name), name) {
				tunnelID = strings.TrimSpace(tunnel.ID)
				adopted = tunnelID != ""
				break
			}
		}
		if tunnelID == "" {
			if request.DryRun {
				return BootstrapResult{AccountID: accountID}, nil
			}
			tunnel, createErr := api.CreateTunnel(ctx, token, accountID, name)
			if createErr != nil {
				return BootstrapResult{}, fmt.Errorf("create Cloudflare tunnel: %w", createErr)
			}
			tunnelID = strings.TrimSpace(tunnel.ID)
			created = true
		}
	}
	if tunnelID == "" {
		return BootstrapResult{}, fmt.Errorf("Cloudflare tunnel id is empty")
	}
	if request.DryRun {
		return BootstrapResult{AccountID: accountID, TunnelID: tunnelID, Adopted: adopted, Created: created}, nil
	}
	connector, err := api.ConnectorToken(ctx, token, accountID, tunnelID)
	if err != nil {
		return BootstrapResult{}, fmt.Errorf("fetch Cloudflare connector token: %w", err)
	}
	values := []struct{ field, value string }{
		{"cloudflare-api-token", token},
		{"cloudflare-account-id", accountID},
		{"cloudflare-tunnel-id", tunnelID},
		{"cloudflare-connector-token", strings.TrimSpace(connector)},
	}
	for _, value := range values {
		if value.value == "" {
			return BootstrapResult{}, fmt.Errorf("Cloudflare bootstrap returned an empty %s", value.field)
		}
	}
	previous := make(map[string]string, len(values))
	present := make(map[string]bool, len(values))
	for _, value := range values {
		if old, resolveErr := authority.Resolve(identity, value.field); resolveErr == nil {
			previous[value.field] = old
			present[value.field] = true
		}
	}
	for _, value := range values {
		if err := authority.Put(identity, value.field, value.value); err != nil {
			for _, rollback := range values {
				if present[rollback.field] {
					_ = authority.Put(identity, rollback.field, previous[rollback.field])
				} else {
					_ = authority.Delete(identity, rollback.field)
				}
			}
			return BootstrapResult{}, fmt.Errorf("write Cloudflare bootstrap credential %s: %w", value.field, err)
		}
	}
	return BootstrapResult{AccountID: accountID, TunnelID: tunnelID, Adopted: adopted, Created: created, Written: true}, nil
}

// decodeCloudflareTunnelToken extracts only the non-secret account and tunnel
// coordinates from the connector token formats emitted by cloudflared. It is
// best-effort: API discovery remains authoritative and malformed tokens are
// simply ignored rather than becoming an alternate credential source.
func decodeCloudflareTunnelToken(token string) (accountID, tunnelID string) {
	candidates := []string{token}
	parts := strings.Split(token, ".")
	if len(parts) >= 2 {
		candidates = append(candidates, parts[1])
	}
	for _, candidate := range candidates {
		for _, encoding := range []*base64.Encoding{base64.RawURLEncoding, base64.URLEncoding, base64.RawStdEncoding, base64.StdEncoding} {
			decoded, err := encoding.DecodeString(strings.TrimSpace(candidate))
			if err != nil {
				continue
			}
			var payload struct {
				AccountID string `json:"a"`
				TunnelID  string `json:"t"`
			}
			if json.Unmarshal(decoded, &payload) == nil && strings.TrimSpace(payload.AccountID) != "" && strings.TrimSpace(payload.TunnelID) != "" {
				return strings.TrimSpace(payload.AccountID), strings.TrimSpace(payload.TunnelID)
			}
		}
	}
	return "", ""
}
