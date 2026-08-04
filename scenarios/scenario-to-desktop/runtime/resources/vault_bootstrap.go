package resources

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
	"github.com/vrooli/vrooli/packages/resource-deployment/securestore"
	vaultbootstrap "github.com/vrooli/vrooli/packages/vaultbootstrap-go"
)

var desktopVaultStore = securestore.Default

// desktopUnsealKeyStore returns the credential authority, or nil on a host
// without one. Nil means the unseal key is stored beside the root token only
// and is therefore absent from recovery bundles — degraded, not fatal.
var desktopUnsealKeyStore = func() vaultbootstrap.UnsealKeyStore {
	authority, err := credentialauthority.Default()
	if err != nil {
		return nil
	}
	return authority
}

// This file is the desktop runtime's wiring around the shared Vault bootstrap
// sequence. The sequence itself — initialize, persist, unseal, mount KV, mint a
// scoped token, verify it — lives in packages/vaultbootstrap-go, so the control
// plane and a desktop bundle cannot drift apart or file their recovery material
// under different names.
//
// What stays here is what is genuinely desktop-specific: how an instance is
// identified from its app data directory, and how the resulting token reaches
// the bundle's process environment.

func init() { registerPrivateServiceBootstrapper("vault", bootstrapPrivateVault) }

// bootstrapPrivateVault owns only this bundle's instance, obtains an app-scoped
// token, and returns it solely for process-environment injection.
func bootstrapPrivateVault(ctx context.Context, _ Item, ports map[string]int, appDataDir string) (map[string]string, error) {
	port := ports["http"]
	if port <= 0 {
		return nil, fmt.Errorf("Vault private service has no HTTP port")
	}
	store := desktopVaultStore()
	// Recovery material is fail-closed: an instance published as usable whose
	// unseal key was never stored is an instance nobody can ever reopen.
	if err := securestore.ProbeWritable(store); err != nil {
		return nil, fmt.Errorf("private Vault requires platform secure storage: %w", err)
	}

	instanceID := desktopVaultInstanceID(appDataDir)
	client := vaultbootstrap.Client{Endpoint: "http://127.0.0.1:" + strconv.Itoa(port)}
	if err := client.WaitReachable(ctx, 60*time.Second); err != nil {
		return nil, err
	}

	material, err := loadOrBootstrapDesktopVault(ctx, client, store, instanceID)
	if err != nil {
		return nil, err
	}
	if err := client.EnsureKVv2(ctx, material.RootToken); err != nil {
		return nil, err
	}
	token, err := desktopVaultScopedToken(ctx, client, material.RootToken, instanceID)
	if err != nil {
		return nil, err
	}
	if err := client.VerifyScopedOperation(ctx, token); err != nil {
		return nil, err
	}
	return map[string]string{"VAULT_ADDR": client.Endpoint, "VAULT_TOKEN": token}, nil
}

// desktopVaultInstanceID derives a stable identity from the bundle's data
// directory, so a restart recovers the same instance and two bundles on one
// machine never share recovery material.
func desktopVaultInstanceID(appDataDir string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(appDataDir)))
	return hex.EncodeToString(sum[:16])
}

// loadOrBootstrapDesktopVault recovers an existing instance or creates one.
//
// The order matters on the create path: material is stored before the instance
// is unsealed, because an unsealed Vault whose key was never persisted is
// unrecoverable the moment the process exits.
func loadOrBootstrapDesktopVault(
	ctx context.Context,
	client vaultbootstrap.Client,
	store securestore.Store,
	instanceID string,
) (vaultbootstrap.Material, error) {
	material, found, err := vaultbootstrap.Load(store, instanceID)
	if err != nil {
		return vaultbootstrap.Material{}, err
	}
	if found {
		if err := client.Unseal(ctx, material); err != nil {
			return vaultbootstrap.Material{}, fmt.Errorf("recover private Vault: %w", err)
		}
		return material, nil
	}

	state, err := client.LifecycleState(ctx)
	if err != nil {
		return vaultbootstrap.Material{}, err
	}
	if state != vaultbootstrap.StateUninitialized {
		// An initialized instance with no stored blob is the restore case: a
		// recovery bundle carries the unseal key and deliberately not the root
		// token, so the token is re-minted here rather than being something the
		// operator has to have kept.
		return recoverDesktopVaultFromUnsealKey(ctx, client, store, instanceID, state)
	}

	material, err = client.Initialize(ctx)
	if err != nil {
		return vaultbootstrap.Material{}, err
	}
	// The bundle's unseal key goes through the credential authority too, so a
	// desktop install's Vault is covered by the same recovery bundle as every
	// other credential rather than being the one thing a backup misses.
	if err := vaultbootstrap.Save(store, desktopUnsealKeyStore(), instanceID, material); err != nil {
		return vaultbootstrap.Material{}, err
	}
	if err := client.Unseal(ctx, material); err != nil {
		return vaultbootstrap.Material{}, err
	}
	return material, nil
}

// desktopVaultScopedToken mints a token limited to this instance's own app
// path. The root token never leaves this process; only the scoped token is
// handed to the bundle.
func desktopVaultScopedToken(
	ctx context.Context,
	client vaultbootstrap.Client,
	rootToken, instanceID string,
) (string, error) {
	policyName := "vrooli-desktop-" + instanceID
	policy := fmt.Sprintf(
		"path %q { capabilities = [\"create\", \"read\", \"update\", \"delete\", \"list\"] }\npath %q { capabilities = [\"list\"] }\n",
		"secret/data/apps/"+instanceID+"/*", "secret/metadata/apps/"+instanceID+"/*")
	if err := client.Request(ctx, http.MethodPut, "/v1/sys/policies/acl/"+policyName, rootToken,
		map[string]string{"policy": policy}, nil); err != nil {
		return "", fmt.Errorf("write private Vault policy: %w", err)
	}

	var response struct {
		Auth struct {
			ClientToken string `json:"client_token"`
		} `json:"auth"`
	}
	err := client.Request(ctx, http.MethodPost, "/v1/auth/token/create", rootToken,
		map[string]any{"policies": []string{policyName}, "ttl": (24 * time.Hour).String(), "no_parent": true},
		&response)
	if err != nil {
		return "", fmt.Errorf("create private scoped Vault token: %w", err)
	}
	if response.Auth.ClientToken == "" {
		return "", fmt.Errorf("create private scoped Vault token: Vault returned no token")
	}
	return response.Auth.ClientToken, nil
}

// recoverDesktopVaultFromUnsealKey restores an instance from the one half a
// recovery bundle carries.
//
// This is the other end of the backup decision: because a bundle holds only the
// unseal key, a restored host must be able to unseal and then mint its own root
// token. Without this path the operator would hold the irreplaceable half and
// still be unable to administer the instance.
func recoverDesktopVaultFromUnsealKey(
	ctx context.Context,
	client vaultbootstrap.Client,
	store securestore.Store,
	instanceID string,
	state vaultbootstrap.State,
) (vaultbootstrap.Material, error) {
	keys := desktopUnsealKeyStore()
	if keys == nil {
		return vaultbootstrap.Material{}, fmt.Errorf(
			"private Vault is %s and this host has no credential authority to recover its unseal key from", state)
	}
	unsealKey, found, err := vaultbootstrap.LoadUnsealKey(keys, instanceID)
	if err != nil {
		return vaultbootstrap.Material{}, err
	}
	if !found {
		return vaultbootstrap.Material{}, fmt.Errorf(
			"private Vault is %s but no unseal key is stored for it; restore a recovery bundle containing %s",
			state, "vrooli/vault/"+instanceID+":unseal-key")
	}

	material := vaultbootstrap.Material{UnsealKey: unsealKey}
	if state == vaultbootstrap.StateSealed {
		if err := client.Unseal(ctx, material); err != nil {
			return vaultbootstrap.Material{}, err
		}
	}
	rootToken, err := client.GenerateRootToken(ctx, unsealKey)
	if err != nil {
		return vaultbootstrap.Material{}, err
	}
	material.RootToken = rootToken

	// Persist the re-minted pair so the next start is an ordinary recovery
	// rather than another root generation.
	if err := vaultbootstrap.Save(store, keys, instanceID, material); err != nil {
		return vaultbootstrap.Material{}, err
	}
	return material, nil
}
