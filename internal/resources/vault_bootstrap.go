package resources

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/tuning"

	"github.com/vrooli/vrooli/internal/resources/securestore"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
	resourcedeployment "github.com/vrooli/vrooli/packages/resource-deployment"
	vaultbootstrap "github.com/vrooli/vrooli/packages/vaultbootstrap-go"
)

const (
	vaultBootstrapVault = "vault"
)

func init() {
	registerManagedSharedBootstrapper(vaultBootstrapVault, func(ctx context.Context, host *UserResourceHost, instance ManagedInstance, appScope string) error {
		_, err := host.EnsureVault(ctx, instance, appScope, HTTPVaultBootstrapper{})
		return err
	})
	registerManagedPrivateBootstrapper(vaultBootstrapVault, bootstrapPrivateVault)
}

// privateVaultSecureStore is injectable only to make the resource-native
// bootstrap contract observable in tests. Production always resolves the
// platform credential store and fails closed when it is unavailable.
var privateVaultSecureStore = defaultManagedSharedSecureStore

func storeVaultBootstrapMaterial(store securestore.Store, instanceID string, material VaultBootstrapMaterial) error {
	return vaultbootstrap.Save(store, unsealKeyStore(), instanceID, material)
}

// recoverVaultFromUnsealKey restores an instance whose stored material is gone
// but whose unseal key survived a recovery bundle.
//
// This is the far end of the backup decision. A bundle carries the unseal key
// and deliberately not the root token, so on a restored host the blob is absent
// while the key is present — a state that used to be reported as unrecoverable
// even though everything needed to recover was sitting in the credential store.
func recoverVaultFromUnsealKey(
	ctx context.Context,
	bootstrap VaultBootstrapper,
	store securestore.Store,
	endpoint, instanceID string,
) (VaultBootstrapMaterial, bool, error) {
	keys := unsealKeyStore()
	if keys == nil {
		return VaultBootstrapMaterial{}, false, nil
	}
	unsealKey, found, err := vaultbootstrap.LoadUnsealKey(keys, instanceID)
	if err != nil || !found {
		// Not found here is not a failure: it simply means this is a fresh
		// instance rather than a restore, and the caller bootstraps normally.
		return VaultBootstrapMaterial{}, false, err
	}

	regenerator, ok := bootstrap.(VaultRootRegenerator)
	if !ok {
		return VaultBootstrapMaterial{}, false, fmt.Errorf(
			"an unseal key is stored for %s but this Vault adapter cannot regenerate a root token", instanceID)
	}
	material := VaultBootstrapMaterial{UnsealKey: unsealKey}
	if recovery, canUnseal := bootstrap.(VaultRecoveryBootstrapper); canUnseal {
		if err := recovery.Unseal(ctx, endpoint, material); err != nil {
			return VaultBootstrapMaterial{}, false, fmt.Errorf("unseal %s with its recovered key: %w", instanceID, err)
		}
	}
	rootToken, err := regenerator.GenerateRootToken(ctx, endpoint, unsealKey)
	if err != nil {
		return VaultBootstrapMaterial{}, false, fmt.Errorf("regenerate root token for %s: %w", instanceID, err)
	}
	material.RootToken = rootToken

	// Persist the re-minted pair so the next start is an ordinary recovery
	// rather than another root generation.
	if err := storeVaultBootstrapMaterial(store, instanceID, material); err != nil {
		return VaultBootstrapMaterial{}, false, err
	}
	return material, true, nil
}

// unsealKeyStore returns the credential authority, or nil when this host has
// none. A nil sink means the unseal key is stored only beside the root token
// and therefore is not in any recovery bundle — degraded, but strictly better
// than refusing to bootstrap an instance that would otherwise work.
//
// It is a variable so tests can supply a sink without a real backend.
var unsealKeyStore = func() vaultbootstrap.UnsealKeyStore {
	authority, err := credentialauthority.Default()
	if err != nil {
		return nil
	}
	return authority
}

// loadVaultBootstrapMaterial reads recovery material for an instance. A clean
// "nothing stored" is reported as found=false and never as an error, so a first
// run is not mistaken for a broken store.
func loadVaultBootstrapMaterial(store securestore.Store, instanceID string) (VaultBootstrapMaterial, bool, error) {
	return vaultbootstrap.Load(store, unsealKeyStore(), instanceID)
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
	instance := ManagedInstance{ID: state.InstanceID, Resource: vaultBootstrapVault, Provider: resourcedeployment.ProviderManagedPrivate, Endpoint: endpoint}
	bootstrap := HTTPVaultBootstrapper{}
	if err := waitForVaultBootstrapReachability(ctx, endpoint); err != nil {
		return err
	}
	material, found, err := loadVaultBootstrapMaterial(store, state.InstanceID)
	if err != nil {
		// A store that cannot answer is not an empty store. Treating a read
		// failure as "no material" would re-initialize a live instance.
		return err
	}
	if found {
		if err := bootstrap.Unseal(ctx, endpoint, material); err != nil {
			return fmt.Errorf("recover private Vault: %w", err)
		}
	} else {
		// A restored host has no blob but does have its unseal key, because a
		// recovery bundle carries the irreplaceable half and nothing else.
		// Trying this before Bootstrap is what turns that state from
		// "unrecoverable" into an ordinary recovery.
		recovered, didRecover, recoverErr := recoverVaultFromUnsealKey(ctx, bootstrap, store, endpoint, state.InstanceID)
		if recoverErr != nil {
			return recoverErr
		}
		if didRecover {
			material = recovered
		} else {
			material, err = bootstrap.Bootstrap(ctx, endpoint)
			if err != nil {
				return fmt.Errorf("bootstrap private Vault: %w", err)
			}
		}
	}
	if !material.Valid() {
		return fmt.Errorf("private Vault bootstrap returned incomplete management material")
	}
	if err := storeVaultBootstrapMaterial(store, state.InstanceID, material); err != nil {
		return err
	}
	if err := ensureVaultKVv2(ctx, endpoint, material.RootToken); err != nil {
		return fmt.Errorf("configure private Vault KV v2: %w", err)
	}
	lease := Lease{ID: "private-bootstrap-" + state.InstanceID, InstanceID: state.InstanceID, Scope: "private-bootstrap", ExpiresAt: time.Now().Add(tuning.VaultBootstrapLease())}
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
	if err := server.RegisterCredentialIssuer(vaultBootstrapVault, issuer); err != nil {
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
		err = server.RegisterOwnerLifecycle(vaultBootstrapVault, lifecycle)
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

// VaultRootRegenerator mints a fresh root token from an unseal key.
//
// It is what makes "back up the unseal key, not the root token" a complete
// story rather than half of one. A recovery bundle deliberately carries only
// the irreplaceable half, so a restored host must be able to re-mint the rest;
// without this it could unseal an instance and then administer nothing.
//
// Optional, like VaultRecoveryBootstrapper: an adapter that cannot regenerate
// says so by not implementing it, rather than by failing at the moment an
// operator is trying to recover.
type VaultRootRegenerator interface {
	GenerateRootToken(ctx context.Context, endpoint, unsealKey string) (string, error)
}

// VaultBootstrapMaterial is the shared type. It is an alias rather than a copy
// so the control plane and a desktop bundle cannot disagree about the shape of
// the one thing that makes a sealed instance recoverable.
type VaultBootstrapMaterial = vaultbootstrap.Material

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
//
//nolint:gocyclo // Vault bootstrap preserves readiness, initialization, policy, and recovery transitions.
func (h *UserResourceHost) EnsureVault(ctx context.Context, instance ManagedInstance, appScope string, bootstrap VaultBootstrapper) (ManagedInstance, error) {
	if h == nil || h.Broker == nil || h.Secrets == nil || bootstrap == nil {
		return ManagedInstance{}, fmt.Errorf("user resource host is incomplete")
	}
	if instance.Resource != vaultBootstrapVault || instance.Provider != resourcedeployment.ProviderManagedShared || instance.OwnerScope != h.OwnerScope || !isLoopbackManagedEndpoint(instance.Endpoint) {
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
	var material VaultBootstrapMaterial
	stored, storedFound, loadErr := loadVaultBootstrapMaterial(h.Secrets, instance.ID)
	if loadErr != nil {
		// A store that cannot answer is not an empty store. Continuing here
		// would bootstrap a second time over a live instance.
		return ManagedInstance{}, loadErr
	}
	if storedFound {
		material = stored
		recovery, ok := bootstrap.(VaultRecoveryBootstrapper)
		if !ok {
			return ManagedInstance{}, fmt.Errorf("Vault bootstrap adapter cannot recover an initialized instance")
		}
		if err := recovery.Unseal(ctx, instance.Endpoint, material); err != nil {
			return ManagedInstance{}, fmt.Errorf("recover user-hosted Vault: %w", err)
		}
	} else if recovered, didRecover, recoverErr := recoverVaultFromUnsealKey(
		ctx, bootstrap, h.Secrets, instance.Endpoint, instance.ID); recoverErr != nil {
		return ManagedInstance{}, recoverErr
	} else if didRecover {
		// Restored from a bundle: the unseal key was in the credential store
		// even though the material blob was not.
		material = recovered
	} else {
		var err error
		material, err = bootstrap.Bootstrap(ctx, instance.Endpoint)
		if err != nil {
			return ManagedInstance{}, fmt.Errorf("bootstrap user-hosted Vault: %w", err)
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
		ExpiresAt:  time.Now().Add(tuning.VaultBootstrapLease()),
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
	return vaultbootstrap.Client{Endpoint: endpoint}.EnsureKVv2(ctx, managementToken)
}

func waitForVaultBootstrapReachability(parent context.Context, endpoint string) error {
	return vaultbootstrap.Client{Endpoint: endpoint}.WaitReachable(parent, tuning.ReachabilityTimeout())
}

// VaultManagementToken is the only bridge from secure storage to Vault's
// resource-native credential issuer. It never serializes the token into
// broker state or host status.
func (h *UserResourceHost) VaultManagementToken(instance ManagedInstance) (string, error) {
	if h == nil || h.Secrets == nil || instance.Resource != vaultBootstrapVault {
		return "", fmt.Errorf("Vault management token is unavailable")
	}
	material, found, err := loadVaultBootstrapMaterial(h.Secrets, instance.ID)
	if err != nil {
		return "", err
	}
	if !found {
		return "", fmt.Errorf("read Vault management material: nothing stored for instance %s", instance.ID)
	}
	if strings.TrimSpace(material.RootToken) == "" {
		return "", fmt.Errorf("secure Vault management material has no root token")
	}
	return material.RootToken, nil
}

// HTTPVaultBootstrapper implements Vault's documented local bootstrap API.
// All requests are loopback-only and bounded by the caller's context.
// HTTPVaultBootstrapper adapts the shared bootstrap sequence to the control
// plane's interfaces. It is an adapter, not a second implementation: the only
// behaviour it adds is the loopback requirement, which is a control-plane
// safety property the shared package deliberately does not assume — a desktop
// bundle addresses its own private instance the same way, but only the control
// plane refuses to bootstrap something that is not provably local.
type HTTPVaultBootstrapper struct{ Client *http.Client }

func (b HTTPVaultBootstrapper) client(endpoint string) vaultbootstrap.Client {
	return vaultbootstrap.Client{Endpoint: endpoint, HTTP: b.Client}
}

func (b HTTPVaultBootstrapper) LifecycleState(ctx context.Context, endpoint string) (VaultLifecycleState, error) {
	if !isLoopbackManagedEndpoint(endpoint) {
		return "", fmt.Errorf("Vault readiness endpoint must be loopback")
	}
	state, err := b.client(endpoint).LifecycleState(ctx)
	if err != nil {
		// A process that is up but not answering is process-started, not a
		// lifecycle position we can name.
		return VaultStateProcessStarted, err
	}
	return vaultLifecycleFor(state), nil
}

// vaultLifecycleFor maps the shared seal-status classification onto the control
// plane's richer enum, which also carries transport-only positions the shared
// package has no opinion about.
func vaultLifecycleFor(state vaultbootstrap.State) VaultLifecycleState {
	switch state {
	case vaultbootstrap.StateUninitialized:
		return VaultStateUninitialized
	case vaultbootstrap.StateSealed:
		return VaultStateSealed
	case vaultbootstrap.StateUnsealed:
		return VaultStateUnsealed
	default:
		return VaultStateProcessStarted
	}
}

// VerifyScopedOperation requires the application credential selected by the
// caller and makes a harmless self-lookup. The caller owns credential
// provenance; this probe deliberately never reads bootstrap material.
func (b HTTPVaultBootstrapper) VerifyScopedOperation(ctx context.Context, endpoint, scopedToken string) error {
	if !isLoopbackManagedEndpoint(endpoint) || strings.TrimSpace(scopedToken) == "" {
		return fmt.Errorf("Vault scoped readiness requires a loopback endpoint and scoped credential")
	}
	return b.client(endpoint).VerifyScopedOperation(ctx, scopedToken)
}

// Bootstrap initializes a fresh instance and leaves it unsealed.
//
// It refuses an already-initialized instance outright. Reaching this path with
// no stored material means the recovery material was lost, and initializing
// again would not recover the instance — it would replace the key to data that
// is still sealed under the old one.
func (b HTTPVaultBootstrapper) Bootstrap(ctx context.Context, endpoint string) (VaultBootstrapMaterial, error) {
	if !isLoopbackManagedEndpoint(endpoint) {
		return VaultBootstrapMaterial{}, fmt.Errorf("Vault bootstrap endpoint must be loopback")
	}
	client := b.client(endpoint)
	state, err := client.LifecycleState(ctx)
	if err != nil {
		return VaultBootstrapMaterial{}, err
	}
	if state != vaultbootstrap.StateUninitialized {
		return VaultBootstrapMaterial{}, fmt.Errorf("initialized Vault recovery requires existing secure management material")
	}
	material, err := client.Initialize(ctx)
	if err != nil {
		return VaultBootstrapMaterial{}, err
	}
	if err := client.Unseal(ctx, material); err != nil {
		return VaultBootstrapMaterial{}, err
	}
	return material, nil
}

// GenerateRootToken mints a root token from the unseal key, for a host whose
// stored material is gone but whose key survived in a recovery bundle.
func (b HTTPVaultBootstrapper) GenerateRootToken(ctx context.Context, endpoint, unsealKey string) (string, error) {
	if !isLoopbackManagedEndpoint(endpoint) {
		return "", fmt.Errorf("Vault root generation endpoint must be loopback")
	}
	return b.client(endpoint).GenerateRootToken(ctx, unsealKey)
}

// Unseal opens a sealed instance and treats an already-open one as success,
// which is what makes a restart idempotent.
func (b HTTPVaultBootstrapper) Unseal(ctx context.Context, endpoint string, material VaultBootstrapMaterial) error {
	if !isLoopbackManagedEndpoint(endpoint) || strings.TrimSpace(material.UnsealKey) == "" {
		return fmt.Errorf("Vault recovery requires a loopback endpoint and secure unseal material")
	}
	client := b.client(endpoint)
	state, err := client.LifecycleState(ctx)
	if err != nil {
		return err
	}
	if state != vaultbootstrap.StateSealed {
		return nil
	}
	return client.Unseal(ctx, material)
}
