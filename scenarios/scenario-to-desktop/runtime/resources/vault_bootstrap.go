package resources

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/vrooli/vrooli/packages/resource-deployment/securestore"
)

const desktopVaultSecretService = "vrooli.desktop.vault" // #nosec G101 -- secure-store service namespace, not a credential

var desktopVaultStore = securestore.Default

// desktopVaultMaterial is recovery material read only from platform secure
// storage and never logged or serialized outside that store. #nosec G101
type desktopVaultMaterial struct {
	RootToken string `json:"root_token"`
	UnsealKey string `json:"unseal_key"`
}

type desktopVaultHTTPStatusError struct{ StatusCode int }

func (e desktopVaultHTTPStatusError) Error() string {
	return fmt.Sprintf("Vault returned HTTP %d", e.StatusCode)
}

func init() { registerPrivateServiceBootstrapper("vault", bootstrapPrivateVault) }

// bootstrapPrivateVault is deliberately desktop-native: it owns only this
// bundle's recovery material, obtains an app-scoped token, and returns that
// token solely for process-environment injection.
func bootstrapPrivateVault(ctx context.Context, _ Item, ports map[string]int, appDataDir string) (map[string]string, error) {
	port := ports["http"]
	if port <= 0 {
		return nil, fmt.Errorf("Vault private service has no HTTP port")
	}
	store := desktopVaultStore()
	if err := securestore.Probe(store); err != nil {
		return nil, fmt.Errorf("private Vault requires platform secure storage: %w", err)
	}
	instanceID := desktopVaultInstanceID(appDataDir)
	endpoint := "http://127.0.0.1:" + strconv.Itoa(port)
	if err := desktopVaultWaitReachable(ctx, endpoint); err != nil {
		return nil, err
	}
	material, err := loadOrBootstrapDesktopVault(ctx, store, instanceID, endpoint)
	if err != nil {
		return nil, err
	}
	if err := desktopVaultEnsureKV(ctx, endpoint, material.RootToken); err != nil {
		return nil, err
	}
	token, err := desktopVaultScopedToken(ctx, endpoint, material.RootToken, instanceID)
	if err != nil {
		return nil, err
	}
	if err := desktopVaultRequest(ctx, endpoint, http.MethodGet, "/v1/auth/token/lookup-self", token, nil, nil); err != nil {
		return nil, fmt.Errorf("verify scoped Vault operation: %w", err)
	}
	return map[string]string{"VAULT_ADDR": endpoint, "VAULT_TOKEN": token}, nil
}

func desktopVaultWaitReachable(parent context.Context, endpoint string) error {
	ctx, cancel := context.WithTimeout(parent, 60*time.Second)
	defer cancel()
	for {
		if err := desktopVaultRequest(ctx, endpoint, http.MethodGet, "/v1/sys/seal-status", "", nil, nil); err == nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("wait for private Vault reachability: %w", ctx.Err())
		case <-time.After(100 * time.Millisecond):
		}
	}
}

func desktopVaultInstanceID(appDataDir string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(appDataDir)))
	return hex.EncodeToString(sum[:16])
}

func loadOrBootstrapDesktopVault(ctx context.Context, store securestore.Store, instanceID, endpoint string) (desktopVaultMaterial, error) {
	if raw, err := store.Get(desktopVaultSecretService, instanceID); err == nil {
		var material desktopVaultMaterial
		if err := json.Unmarshal([]byte(raw), &material); err != nil || material.RootToken == "" || material.UnsealKey == "" {
			return desktopVaultMaterial{}, fmt.Errorf("parse private Vault recovery material: %w", err)
		}
		if err := desktopVaultRequest(ctx, endpoint, http.MethodPut, "/v1/sys/unseal", "", map[string]string{"key": material.UnsealKey}, nil); err != nil {
			return desktopVaultMaterial{}, fmt.Errorf("unseal private Vault: %w", err)
		}
		return material, nil
	}
	var status struct {
		Initialized bool `json:"initialized"`
	}
	if err := desktopVaultRequest(ctx, endpoint, http.MethodGet, "/v1/sys/seal-status", "", nil, &status); err != nil {
		return desktopVaultMaterial{}, err
	}
	if status.Initialized {
		return desktopVaultMaterial{}, fmt.Errorf("private Vault is initialized but has no recoverable platform credential")
	}
	var initialized struct {
		Keys      []string `json:"keys"`
		RootToken string   `json:"root_token"`
	}
	if err := desktopVaultRequest(ctx, endpoint, http.MethodPut, "/v1/sys/init", "", map[string]int{"secret_shares": 1, "secret_threshold": 1}, &initialized); err != nil || len(initialized.Keys) != 1 || initialized.RootToken == "" {
		return desktopVaultMaterial{}, fmt.Errorf("initialize private Vault: %w", err)
	}
	material := desktopVaultMaterial{RootToken: initialized.RootToken, UnsealKey: initialized.Keys[0]}
	if err := desktopVaultRequest(ctx, endpoint, http.MethodPut, "/v1/sys/unseal", "", map[string]string{"key": material.UnsealKey}, nil); err != nil {
		return desktopVaultMaterial{}, err
	}
	encoded, err := json.Marshal(material)
	if err != nil {
		return desktopVaultMaterial{}, err
	}
	if err := store.Put(desktopVaultSecretService, instanceID, string(encoded)); err != nil {
		return desktopVaultMaterial{}, fmt.Errorf("store private Vault recovery material: %w", err)
	}
	return material, nil
}

func desktopVaultEnsureKV(ctx context.Context, endpoint, rootToken string) error {
	// Vault returns a conflict when the mount already exists; it is safe because
	// this app-private server owns its data directory.
	err := desktopVaultRequest(ctx, endpoint, http.MethodPost, "/v1/sys/mounts/secret", rootToken, map[string]any{"type": "kv", "options": map[string]string{"version": "2"}}, nil)
	var status desktopVaultHTTPStatusError
	if err != nil && (!errors.As(err, &status) || status.StatusCode != http.StatusBadRequest) {
		return fmt.Errorf("enable private Vault KV store: %w", err)
	}
	return nil
}

func desktopVaultScopedToken(ctx context.Context, endpoint, rootToken, instanceID string) (string, error) {
	policyName := "vrooli-desktop-" + instanceID
	policy := fmt.Sprintf("path %q { capabilities = [\"create\", \"read\", \"update\", \"delete\", \"list\"] }\npath %q { capabilities = [\"list\"] }\n", "secret/data/apps/"+instanceID+"/*", "secret/metadata/apps/"+instanceID+"/*")
	if err := desktopVaultRequest(ctx, endpoint, http.MethodPut, "/v1/sys/policies/acl/"+policyName, rootToken, map[string]string{"policy": policy}, nil); err != nil {
		return "", err
	}
	var response struct {
		Auth struct {
			ClientToken string `json:"client_token"`
		} `json:"auth"`
	}
	if err := desktopVaultRequest(ctx, endpoint, http.MethodPost, "/v1/auth/token/create", rootToken, map[string]any{"policies": []string{policyName}, "ttl": (24 * time.Hour).String(), "no_parent": true}, &response); err != nil || response.Auth.ClientToken == "" {
		return "", fmt.Errorf("create private scoped Vault token: %w", err)
	}
	return response.Auth.ClientToken, nil
}

func desktopVaultRequest(ctx context.Context, endpoint, method, path, token string, input, output any) error {
	body, err := json.Marshal(input)
	if err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(ctx, method, strings.TrimRight(endpoint, "/")+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	if token != "" {
		request.Header.Set("X-Vault-Token", token)
	}
	response, err := (&http.Client{Timeout: 10 * time.Second}).Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return desktopVaultHTTPStatusError{StatusCode: response.StatusCode}
	}
	if output != nil {
		return json.NewDecoder(response.Body).Decode(output)
	}
	return nil
}
