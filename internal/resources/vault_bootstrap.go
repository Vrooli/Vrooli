package resources

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/resources/securestore"
	resourcedeployment "github.com/vrooli/vrooli/packages/resource-deployment"
)

func init() {
	registerManagedSharedBootstrapper("vault", func(ctx context.Context, host *UserResourceHost, instance ManagedInstance, appScope string) error {
		_, err := host.EnsureVault(ctx, instance, appScope, HTTPVaultBootstrapper{})
		return err
	})
	registerManagedPrivateBootstrapper("vault", bootstrapPrivateVault)
}

// privateVaultSecureStore is injectable only to make the resource-native
// bootstrap contract observable in tests. Production always resolves the
// platform credential store and fails closed when it is unavailable.
var privateVaultSecureStore = defaultManagedSharedSecureStore

const legacyVaultBootstrapFilename = ".vrooli-bootstrap.json"

type legacyVaultBootstrap struct {
	UnsealKeys []string `json:"unseal_keys_b64"`
	RootToken  string   `json:"root_token"`
}

// loadLegacyVaultBootstrapMaterial is a one-way migration boundary for the
// Docker-era Vault marker. That marker held plaintext recovery material in the
// Vault data directory; the managed-service contract requires it in the OS
// credential store. Callers must store and verify the material before removing
// the marker. Nothing from this function is logged or returned to a CLI.
func loadLegacyVaultBootstrapMaterial(dataDir string) (VaultBootstrapMaterial, bool, error) {
	path := filepath.Join(dataDir, legacyVaultBootstrapFilename)
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return VaultBootstrapMaterial{}, false, nil
	}
	if err != nil {
		return VaultBootstrapMaterial{}, false, fmt.Errorf("inspect legacy Vault bootstrap marker: %w", err)
	}
	if !info.Mode().IsRegular() {
		return VaultBootstrapMaterial{}, false, fmt.Errorf("legacy Vault bootstrap marker is not a regular file")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return VaultBootstrapMaterial{}, false, fmt.Errorf("read legacy Vault bootstrap marker: %w", err)
	}
	var legacy legacyVaultBootstrap
	if err := json.Unmarshal(data, &legacy); err != nil {
		return VaultBootstrapMaterial{}, false, fmt.Errorf("parse legacy Vault bootstrap marker: %w", err)
	}
	if len(legacy.UnsealKeys) == 0 || strings.TrimSpace(legacy.UnsealKeys[0]) == "" || strings.TrimSpace(legacy.RootToken) == "" {
		return VaultBootstrapMaterial{}, false, fmt.Errorf("legacy Vault bootstrap marker has incomplete management material")
	}
	return VaultBootstrapMaterial{RootToken: legacy.RootToken, UnsealKey: legacy.UnsealKeys[0]}, true, nil
}

func removeLegacyVaultBootstrapMarker(dataDir string) error {
	path := filepath.Join(dataDir, legacyVaultBootstrapFilename)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove migrated legacy Vault bootstrap marker: %w", err)
	}
	return nil
}

func storeVaultBootstrapMaterial(store securestore.Store, instanceID string, material VaultBootstrapMaterial) error {
	encoded, err := json.Marshal(material)
	if err != nil {
		return err
	}
	if err := store.Put("vrooli.resource.vault", instanceID, string(encoded)); err != nil {
		return fmt.Errorf("securely store Vault management material: %w", err)
	}
	stored, err := store.Get("vrooli.resource.vault", instanceID)
	if err != nil {
		return fmt.Errorf("verify stored Vault management material: %w", err)
	}
	if stored != string(encoded) {
		return fmt.Errorf("verify stored Vault management material: secure-store readback did not match")
	}
	return nil
}

// bootstrapPrivateVault gives locally supervised Vault instances the same
// safety transition as shared instances, without registering a reusable
// broker endpoint. The supervisor's stable instance ID namespaces recovery
// material, so restarts recover the same file-backed Vault while distinct
// Vrooli-managed instances never share root or unseal material.
func bootstrapPrivateVault(ctx context.Context, state ManagedServiceState, endpoint string) error {
	if strings.TrimSpace(state.InstanceID) == "" || !isLoopbackManagedEndpoint(endpoint) {
		return fmt.Errorf("private Vault bootstrap requires a verified supervised loopback instance")
	}
	store := privateVaultSecureStore()
	if err := securestore.ProbeWritable(store); err != nil {
		return fmt.Errorf("private Vault requires operating-system secure storage: %w", err)
	}
	instance := ManagedInstance{ID: state.InstanceID, Resource: "vault", Provider: resourcedeployment.ProviderManagedPrivate, Endpoint: endpoint}
	bootstrap := HTTPVaultBootstrapper{}
	if err := waitForVaultBootstrapReachability(ctx, endpoint); err != nil {
		return err
	}
	var material VaultBootstrapMaterial
	if raw, err := store.Get("vrooli.resource.vault.private", state.InstanceID); err == nil {
		if err := json.Unmarshal([]byte(raw), &material); err != nil {
			return fmt.Errorf("parse private Vault recovery material: %w", err)
		}
		if err := bootstrap.Unseal(ctx, endpoint, material); err != nil {
			return fmt.Errorf("recover private Vault: %w", err)
		}
	} else {
		var err error
		material, err = bootstrap.Bootstrap(ctx, endpoint)
		if err != nil {
			return fmt.Errorf("bootstrap private Vault: %w", err)
		}
	}
	if strings.TrimSpace(material.RootToken) == "" || strings.TrimSpace(material.UnsealKey) == "" {
		return fmt.Errorf("private Vault bootstrap returned incomplete management material")
	}
	encoded, err := json.Marshal(material)
	if err != nil {
		return err
	}
	if err := store.Put("vrooli.resource.vault.private", state.InstanceID, string(encoded)); err != nil {
		return fmt.Errorf("securely store private Vault recovery material: %w", err)
	}
	if err := ensureVaultKVv2(ctx, endpoint, material.RootToken); err != nil {
		return fmt.Errorf("configure private Vault KV v2: %w", err)
	}
	lease := Lease{ID: "private-bootstrap-" + state.InstanceID, InstanceID: state.InstanceID, Scope: "private-bootstrap", ExpiresAt: time.Now().Add(5 * time.Minute)}
	credential, err := (VaultCredentialIssuer{ManagementToken: func(ManagedInstance) (string, error) { return material.RootToken, nil }}).IssueScopedCredential(instance, lease)
	if err != nil {
		return fmt.Errorf("issue private Vault scoped credential: %w", err)
	}
	if err := bootstrap.VerifyScopedOperation(ctx, endpoint, credential.Credential); err != nil {
		return fmt.Errorf("verify private Vault scoped operation: %w", err)
	}
	return nil
}

// StartVaultBrokerControl exposes only scoped use/credential operations to
// installed applications. The owner token may authorize host management; app
// tokens cannot. The returned server contains no management material.
func (h *UserResourceHost) StartVaultBrokerControl(listener net.Listener, credentials map[string]string) (*BrokerControlServer, error) {
	if h == nil || h.Broker == nil {
		return nil, fmt.Errorf("user resource host broker is unavailable")
	}
	server, err := StartBrokerControlServer(listener, h.Broker, credentials)
	if err != nil {
		return nil, err
	}
	issuer := VaultCredentialIssuer{ManagementToken: h.VaultManagementToken}
	if err := server.RegisterCredentialIssuer("vault", issuer); err != nil {
		_ = server.Close(context.Background())
		return nil, err
	}
	return server, nil
}

// StartVaultBrokerControlWithLifecycle is the production embedding for the
// broker's owner-only Vault operations. Applications still receive only use
// credentials; the supplied controller and manifest remain fixed on the host
// side and are never selected by a broker request.
func (h *UserResourceHost) StartVaultBrokerControlWithLifecycle(listener net.Listener, credentials map[string]string, controller *Controller, manifest ResourceManifest) (*BrokerControlServer, error) {
	server, err := h.StartVaultBrokerControl(listener, credentials)
	if err != nil {
		return nil, err
	}
	lifecycle, err := NewManagedServiceOwnerLifecycle(controller, manifest)
	if err == nil {
		err = server.RegisterOwnerLifecycle("vault", lifecycle)
	}
	if err != nil {
		_ = server.Close(context.Background())
		return nil, err
	}
	return server, nil
}

// VaultBootstrapper performs Vault-native initialization and unseal work. It
// returns management material only to this resource-native boundary.
type VaultBootstrapper interface {
	Bootstrap(context.Context, string) (VaultBootstrapMaterial, error)
}

type VaultRecoveryBootstrapper interface {
	Unseal(context.Context, string, VaultBootstrapMaterial) error
}

type VaultBootstrapMaterial struct {
	RootToken string `json:"root_token"`
	UnsealKey string `json:"unseal_key"`
}

// VaultLifecycleState distinguishes transport reachability from the state an
// application may safely consume. In particular, Vault deliberately returns
// HTTP 501 while reachable but uninitialized; treating that response as a
// health success would expose a non-functional service to applications.
type VaultLifecycleState string

const (
	VaultStateProcessStarted VaultLifecycleState = "process-started"
	VaultStateReachable      VaultLifecycleState = "reachable"
	VaultStateUninitialized  VaultLifecycleState = "uninitialized"
	VaultStateSealed         VaultLifecycleState = "sealed"
	VaultStateUnsealed       VaultLifecycleState = "unsealed"
	VaultStateUsable         VaultLifecycleState = "usable"
)

// VaultReadiness is the resource-native readiness seam. Generic supervisors
// may establish process and transport health, but only this contract may
// attest that Vault has completed initialization/recovery and a scoped use
// operation. Root or recovery material is intentionally not part of it.
type VaultReadiness interface {
	LifecycleState(context.Context, string) (VaultLifecycleState, error)
	VerifyScopedOperation(context.Context, string, string) error
}

// ClassifyVaultSealStatus maps Vault's documented seal-status response into a
// lifecycle state. A caller must separately prove a scoped operation before
// reporting VaultStateUsable.
func ClassifyVaultSealStatus(initialized, sealed bool) VaultLifecycleState {
	if !initialized {
		return VaultStateUninitialized
	}
	if sealed {
		return VaultStateSealed
	}
	return VaultStateUnsealed
}

// EnsureVault stores Vault recovery material before the shared instance is
// leasable. It is recovery-safe: an initialized instance is unsealed using
// secure material, while a fresh instance initializes once.
func (h *UserResourceHost) EnsureVault(ctx context.Context, instance ManagedInstance, appScope string, bootstrap VaultBootstrapper) (ManagedInstance, error) {
	if h == nil || h.Broker == nil || h.Secrets == nil || bootstrap == nil {
		return ManagedInstance{}, fmt.Errorf("user resource host is incomplete")
	}
	if instance.Resource != "vault" || instance.Provider != resourcedeployment.ProviderManagedShared || instance.OwnerScope != h.OwnerScope || !isLoopbackManagedEndpoint(instance.Endpoint) {
		return ManagedInstance{}, fmt.Errorf("Vault user-host instance is not a verified owned loopback service")
	}
	if h.VerifyAttestation != nil {
		if err := h.VerifyAttestation(instance); err != nil {
			return ManagedInstance{}, fmt.Errorf("verify Vault shared ownership attestation: %w", err)
		}
	}
	if err := h.SecureStorageReady(instance.ID); err != nil {
		return ManagedInstance{}, fmt.Errorf("secure storage is not ready; refusing Vault initialization: %w", err)
	}
	if err := waitForVaultBootstrapReachability(ctx, instance.Endpoint); err != nil {
		return ManagedInstance{}, err
	}
	var legacyDataDir string
	legacyMarker := false
	var material VaultBootstrapMaterial
	if raw, err := h.Secrets.Get("vrooli.resource.vault", instance.ID); err == nil {
		if err := json.Unmarshal([]byte(raw), &material); err != nil {
			return ManagedInstance{}, fmt.Errorf("parse secure Vault management material: %w", err)
		}
		recovery, ok := bootstrap.(VaultRecoveryBootstrapper)
		if !ok {
			return ManagedInstance{}, fmt.Errorf("Vault bootstrap adapter cannot recover an initialized instance")
		}
		if err := recovery.Unseal(ctx, instance.Endpoint, material); err != nil {
			return ManagedInstance{}, fmt.Errorf("recover user-hosted Vault: %w", err)
		}
	} else {
		material, err = bootstrap.Bootstrap(ctx, instance.Endpoint)
		if err != nil {
			paths, pathsErr := resourceStoragePaths("vault")
			if pathsErr != nil {
				return ManagedInstance{}, fmt.Errorf("bootstrap user-hosted Vault: %w", err)
			}
			legacyMaterial, found, migrationErr := loadLegacyVaultBootstrapMaterial(paths.DataDir)
			if migrationErr != nil {
				return ManagedInstance{}, fmt.Errorf("migrate legacy Vault bootstrap material: %w", migrationErr)
			}
			if !found {
				return ManagedInstance{}, fmt.Errorf("bootstrap user-hosted Vault: %w", err)
			}
			recovery, ok := bootstrap.(VaultRecoveryBootstrapper)
			if !ok {
				return ManagedInstance{}, fmt.Errorf("Vault bootstrap adapter cannot recover migrated legacy material")
			}
			material = legacyMaterial
			if err := recovery.Unseal(ctx, instance.Endpoint, material); err != nil {
				return ManagedInstance{}, fmt.Errorf("recover user-hosted Vault from migrated legacy material: %w", err)
			}
			legacyDataDir = paths.DataDir
			legacyMarker = true
		}
	}
	if strings.TrimSpace(material.RootToken) == "" || strings.TrimSpace(material.UnsealKey) == "" {
		return ManagedInstance{}, fmt.Errorf("Vault bootstrap returned incomplete management material")
	}
	if err := storeVaultBootstrapMaterial(h.Secrets, instance.ID, material); err != nil {
		return ManagedInstance{}, err
	}
	if err := ensureVaultKVv2(ctx, instance.Endpoint, material.RootToken); err != nil {
		return ManagedInstance{}, fmt.Errorf("configure shared Vault KV v2: %w", err)
	}
	// A reachable, unsealed process is still not usable until an application
	// credential can authenticate. Prove that condition before publishing the
	// instance to the broker: otherwise another application could lease a
	// service that has only completed its bootstrap transport steps.
	readiness, ok := bootstrap.(VaultReadiness)
	if !ok {
		return ManagedInstance{}, fmt.Errorf("Vault bootstrap adapter does not verify scoped readiness")
	}
	lease := Lease{
		ID:         "bootstrap-" + instance.ID,
		InstanceID: instance.ID,
		Scope:      appScope,
		ExpiresAt:  time.Now().Add(5 * time.Minute),
	}
	credential, err := (VaultCredentialIssuer{ManagementToken: h.VaultManagementToken}).IssueScopedCredential(instance, lease)
	if err != nil {
		return ManagedInstance{}, fmt.Errorf("issue Vault bootstrap scoped credential: %w", err)
	}
	if err := readiness.VerifyScopedOperation(ctx, instance.Endpoint, credential.Credential); err != nil {
		return ManagedInstance{}, fmt.Errorf("verify Vault bootstrap scoped operation: %w", err)
	}
	registered, err := h.Broker.RegisterOrGrantScope(instance, appScope)
	if err != nil {
		return ManagedInstance{}, err
	}
	if legacyMarker {
		if err := removeLegacyVaultBootstrapMarker(legacyDataDir); err != nil {
			return ManagedInstance{}, err
		}
	}
	return registered, nil
}

// ensureVaultKVv2 creates the resource's scoped-secret mount once. Vault
// reports HTTP 400 when an existing mount already occupies this path, which is
// the safe restart condition; every other non-success response is a bootstrap
// failure rather than a reason to expose an unusable service.
func ensureVaultKVv2(ctx context.Context, endpoint, managementToken string) error {
	if !isLoopbackManagedEndpoint(endpoint) || strings.TrimSpace(managementToken) == "" {
		return fmt.Errorf("Vault KV bootstrap requires a loopback endpoint and management token")
	}
	data, err := json.Marshal(map[string]any{"type": "kv", "options": map[string]string{"version": "2"}})
	if err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(endpoint, "/")+"/v1/sys/mounts/secret", bytes.NewReader(data))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Vault-Token", managementToken)
	response, err := (&http.Client{Timeout: 10 * time.Second}).Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode >= http.StatusOK && response.StatusCode < http.StatusMultipleChoices || response.StatusCode == http.StatusBadRequest {
		return nil
	}
	return fmt.Errorf("Vault KV mount request returned %s", response.Status)
}

func waitForVaultBootstrapReachability(parent context.Context, endpoint string) error {
	ctx, cancel := context.WithTimeout(parent, 60*time.Second)
	defer cancel()
	client := &http.Client{Timeout: 5 * time.Second}
	for {
		var status struct {
			Initialized bool `json:"initialized"`
		}
		if err := vaultBootstrapRequest(ctx, client, endpoint, http.MethodGet, "/v1/sys/seal-status", nil, &status); err == nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("wait for Vault bootstrap reachability: %w", ctx.Err())
		case <-time.After(100 * time.Millisecond):
		}
	}
}

// VaultManagementToken is the only bridge from secure storage to Vault's
// resource-native credential issuer. It never serializes the token into
// broker state or host status.
func (h *UserResourceHost) VaultManagementToken(instance ManagedInstance) (string, error) {
	if h == nil || h.Secrets == nil || instance.Resource != "vault" {
		return "", fmt.Errorf("Vault management token is unavailable")
	}
	raw, err := h.Secrets.Get("vrooli.resource.vault", instance.ID)
	if err != nil {
		return "", fmt.Errorf("read Vault management material: %w", err)
	}
	var material VaultBootstrapMaterial
	if err := json.Unmarshal([]byte(raw), &material); err != nil {
		return "", fmt.Errorf("parse secure Vault management material: %w", err)
	}
	if strings.TrimSpace(material.RootToken) == "" {
		return "", fmt.Errorf("secure Vault management material has no root token")
	}
	return material.RootToken, nil
}

// HTTPVaultBootstrapper implements Vault's documented local bootstrap API.
// All requests are loopback-only and bounded by the caller's context.
type HTTPVaultBootstrapper struct{ Client *http.Client }

func (b HTTPVaultBootstrapper) LifecycleState(ctx context.Context, endpoint string) (VaultLifecycleState, error) {
	if !isLoopbackManagedEndpoint(endpoint) {
		return "", fmt.Errorf("Vault readiness endpoint must be loopback")
	}
	client := b.Client
	if client == nil {
		client = &http.Client{}
	}
	var status struct {
		Initialized bool `json:"initialized"`
		Sealed      bool `json:"sealed"`
	}
	if err := vaultBootstrapRequest(ctx, client, endpoint, http.MethodGet, "/v1/sys/seal-status", nil, &status); err != nil {
		return VaultStateProcessStarted, err
	}
	return ClassifyVaultSealStatus(status.Initialized, status.Sealed), nil
}

// VerifyScopedOperation requires the application credential selected by the
// caller and makes a harmless self-lookup. The caller owns credential
// provenance; this probe deliberately never reads bootstrap material.
func (b HTTPVaultBootstrapper) VerifyScopedOperation(ctx context.Context, endpoint, scopedToken string) error {
	if !isLoopbackManagedEndpoint(endpoint) || strings.TrimSpace(scopedToken) == "" {
		return fmt.Errorf("Vault scoped readiness requires a loopback endpoint and scoped credential")
	}
	client := b.Client
	if client == nil {
		client = &http.Client{}
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(endpoint, "/")+"/v1/auth/token/lookup-self", nil)
	if err != nil {
		return err
	}
	request.Header.Set("X-Vault-Token", scopedToken)
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("Vault scoped readiness request returned %s", response.Status)
	}
	return nil
}

func (b HTTPVaultBootstrapper) Bootstrap(ctx context.Context, endpoint string) (VaultBootstrapMaterial, error) {
	if !isLoopbackManagedEndpoint(endpoint) {
		return VaultBootstrapMaterial{}, fmt.Errorf("Vault bootstrap endpoint must be loopback")
	}
	client := b.Client
	if client == nil {
		client = &http.Client{}
	}
	var status struct {
		Initialized bool `json:"initialized"`
		Sealed      bool `json:"sealed"`
	}
	if err := vaultBootstrapRequest(ctx, client, endpoint, http.MethodGet, "/v1/sys/seal-status", nil, &status); err != nil {
		return VaultBootstrapMaterial{}, err
	}
	if !status.Initialized {
		var initialized struct {
			Keys      []string `json:"keys"`
			RootToken string   `json:"root_token"`
		}
		if err := vaultBootstrapRequest(ctx, client, endpoint, http.MethodPut, "/v1/sys/init", map[string]int{"secret_shares": 1, "secret_threshold": 1}, &initialized); err != nil {
			return VaultBootstrapMaterial{}, err
		}
		if len(initialized.Keys) != 1 || strings.TrimSpace(initialized.RootToken) == "" {
			return VaultBootstrapMaterial{}, fmt.Errorf("Vault initialization returned incomplete material")
		}
		if err := vaultBootstrapRequest(ctx, client, endpoint, http.MethodPut, "/v1/sys/unseal", map[string]string{"key": initialized.Keys[0]}, nil); err != nil {
			return VaultBootstrapMaterial{}, err
		}
		return VaultBootstrapMaterial{RootToken: initialized.RootToken, UnsealKey: initialized.Keys[0]}, nil
	}
	return VaultBootstrapMaterial{}, fmt.Errorf("initialized Vault recovery requires existing secure management material")
}

func (b HTTPVaultBootstrapper) Unseal(ctx context.Context, endpoint string, material VaultBootstrapMaterial) error {
	if !isLoopbackManagedEndpoint(endpoint) || strings.TrimSpace(material.UnsealKey) == "" {
		return fmt.Errorf("Vault recovery requires a loopback endpoint and secure unseal material")
	}
	client := b.Client
	if client == nil {
		client = &http.Client{}
	}
	var status struct {
		Sealed bool `json:"sealed"`
	}
	if err := vaultBootstrapRequest(ctx, client, endpoint, http.MethodGet, "/v1/sys/seal-status", nil, &status); err != nil {
		return err
	}
	if !status.Sealed {
		return nil
	}
	return vaultBootstrapRequest(ctx, client, endpoint, http.MethodPut, "/v1/sys/unseal", map[string]string{"key": material.UnsealKey}, nil)
}

func vaultBootstrapRequest(ctx context.Context, client *http.Client, endpoint, method, path string, input, output any) error {
	data, err := json.Marshal(input)
	if err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(ctx, method, strings.TrimRight(endpoint, "/")+path, bytes.NewReader(data))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("Vault bootstrap API returned %s", response.Status)
	}
	if output == nil {
		return nil
	}
	return json.NewDecoder(response.Body).Decode(output)
}
