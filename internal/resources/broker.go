package resources

import (
	"fmt"
	"net"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	resourcedeployment "github.com/vrooli/vrooli/packages/resource-deployment"
)

// ManagedInstance is a broker-verified Vrooli-owned service. Endpoint alone is
// intentionally not identity: callers can never register or reuse an arbitrary
// local process merely because it answers on a familiar port.
type ManagedInstance struct {
	ID                string
	Resource          string
	Provider          resourcedeployment.ProviderMode
	OwnerScope        string
	CapabilityVersion string
	Endpoint          string
	AuthorizedScopes  []string
	Attestation       OwnershipAttestation
}

// OwnershipAttestation is non-secret broker evidence that binds a shared
// registration to the process the verified supervisor owns. Its proof is
// derived from the process-only ownership token; the token itself is never
// persisted in broker state or returned to applications.
type OwnershipAttestation struct {
	InstanceID        string    `json:"instance_id"`
	ArtifactSHA256    string    `json:"artifact_sha256"`
	Endpoint          string    `json:"endpoint"`
	ControlCapability string    `json:"control_capability"`
	IssuedAt          time.Time `json:"issued_at"`
	ExpiresAt         time.Time `json:"expires_at"`
	Proof             string    `json:"proof"`
}

// Lease grants use access to one shared instance. It deliberately carries no
// management authority; only the registered owner can operate the service.
type Lease struct {
	ID         string
	InstanceID string
	Scope      string
	ExpiresAt  time.Time
}

// ScopedCredential is issued by a resource-specific policy adapter after the
// broker has authorized a lease. It is intentionally ephemeral: the broker
// persists no bearer material and a lease never grants management authority.
type ScopedCredential struct {
	LeaseID    string
	Resource   string
	Scope      string
	ExpiresAt  time.Time
	Credential string
}

// CredentialIssuer is the seam where a managed resource turns an authorized
// app scope into its native credential/policy representation (for example a
// Vault token bound to an app-specific policy). Implementations must not issue
// a credential for an unverified instance or an expired lease.
type CredentialIssuer interface {
	IssueScopedCredential(instance ManagedInstance, lease Lease) (ScopedCredential, error)
}

type Broker struct {
	mu              sync.Mutex
	now             func() time.Time
	instances       map[string]ManagedInstance
	leases          map[string]Lease
	sequence        uint64
	store           BrokerStore
	verifyOwnership func(ManagedInstance) error
}

// SetOwnershipVerifier installs the supervisor-backed proof check used by a
// user resource host. It is deliberately injected: the broker remains
// resource-agnostic while production reuse still verifies each lease against
// the live supervised process.
func (b *Broker) SetOwnershipVerifier(verifier func(ManagedInstance) error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.verifyOwnership = verifier
}

func (b *Broker) verifyLocked(instance ManagedInstance) error {
	if b.verifyOwnership == nil {
		return nil
	}
	if err := b.verifyOwnership(instance); err != nil {
		return fmt.Errorf("verify managed ownership attestation: %w", err)
	}
	return nil
}

func NewBroker(now func() time.Time) *Broker {
	if now == nil {
		now = time.Now
	}
	return &Broker{now: now, instances: make(map[string]ManagedInstance), leases: make(map[string]Lease)}
}

// NewPersistentBroker restores durable ownership and lease records before the
// broker accepts requests. The store never contains credentials; it holds only
// control-plane identity and authorization metadata.
func NewPersistentBroker(now func() time.Time, store BrokerStore) (*Broker, error) {
	b := NewBroker(now)
	b.store = store
	if store == nil {
		return b, nil
	}
	state, err := store.Load()
	if err != nil {
		return nil, err
	}
	b.instances, b.leases, b.sequence = state.Instances, state.Leases, state.Sequence
	if b.instances == nil {
		b.instances = make(map[string]ManagedInstance)
	}
	if b.leases == nil {
		b.leases = make(map[string]Lease)
	}
	return b, nil
}

// InstancesForResource returns every registered instance of one resource,
// sorted by ID so callers are deterministic.
//
// It exists so recovery can enumerate what a manifest cannot name. Declared
// credentials are inventoried from manifests, but a managed instance's ID is
// generated at runtime, so nothing static describes it — and material stored
// under an ID no inventory knows is material no backup captures. That is
// precisely how Vault unseal keys came to have no recovery path.
//
// It returns instances, never credentials: the caller resolves values through
// the credential authority, and the broker deliberately holds no bearer
// material of its own.
func (b *Broker) InstancesForResource(resource string) []ManagedInstance {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]ManagedInstance, 0, len(b.instances))
	for _, instance := range b.instances {
		if instance.Resource == resource {
			out = append(out, instance)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func (b *Broker) Register(instance ManagedInstance) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if strings.TrimSpace(instance.ID) == "" || strings.TrimSpace(instance.Resource) == "" || strings.TrimSpace(instance.OwnerScope) == "" || strings.TrimSpace(instance.CapabilityVersion) == "" {
		return fmt.Errorf("managed instance requires id, resource, owner scope, and capability version")
	}
	if instance.Provider != resourcedeployment.ProviderManagedPrivate && instance.Provider != resourcedeployment.ProviderManagedShared {
		return fmt.Errorf("only Vrooli-owned managed providers may register")
	}
	if strings.TrimSpace(instance.Endpoint) == "" {
		return fmt.Errorf("managed instance requires a control endpoint")
	}
	if !isLoopbackManagedEndpoint(instance.Endpoint) {
		return fmt.Errorf("managed instance control endpoint must be an HTTP loopback address")
	}
	if err := b.verifyLocked(instance); err != nil {
		return err
	}
	if _, exists := b.instances[instance.ID]; exists {
		return fmt.Errorf("managed instance %q is already registered", instance.ID)
	}
	b.instances[instance.ID] = instance
	return b.persistLocked()
}

// RegisterOrGrantScope records a verified user-hosted instance exactly once
// and then admits an additional authenticated application scope. It does not
// discover endpoints: callers must have completed bootstrap/ownership
// verification before this operation. Broker persistence contains only the
// non-secret identity and scope list.
func (b *Broker) RegisterOrGrantScope(instance ManagedInstance, scope string) (ManagedInstance, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if strings.TrimSpace(scope) == "" {
		return ManagedInstance{}, fmt.Errorf("application scope is required")
	}
	if existing, ok := b.instances[instance.ID]; ok {
		if existing.Resource != instance.Resource || existing.Provider != instance.Provider || existing.OwnerScope != instance.OwnerScope || existing.CapabilityVersion != instance.CapabilityVersion || existing.Endpoint != instance.Endpoint {
			return ManagedInstance{}, fmt.Errorf("managed instance %q identity changed", instance.ID)
		}
		// A restart intentionally produces a new process ownership token. Verify
		// the supplied current attestation, then replace the old persisted proof
		// atomically with it; re-checking the old proof here would make an owner
		// unable to restart its own shared service.
		if err := b.verifyLocked(instance); err != nil {
			return ManagedInstance{}, err
		}
		attestationChanged := existing.Attestation != instance.Attestation
		existing.Attestation = instance.Attestation
		changed := attestationChanged
		if !scopeAllowed(existing.AuthorizedScopes, scope) {
			existing.AuthorizedScopes = append(existing.AuthorizedScopes, scope)
			changed = true
		}
		if changed {
			b.instances[existing.ID] = existing
			if err := b.persistLocked(); err != nil {
				return ManagedInstance{}, err
			}
		}
		return existing, nil
	}
	if strings.TrimSpace(instance.ID) == "" || strings.TrimSpace(instance.Resource) == "" || strings.TrimSpace(instance.OwnerScope) == "" || strings.TrimSpace(instance.CapabilityVersion) == "" || strings.TrimSpace(instance.Endpoint) == "" {
		return ManagedInstance{}, fmt.Errorf("managed instance requires id, resource, owner scope, capability version, and endpoint")
	}
	if instance.Provider != resourcedeployment.ProviderManagedShared || !isLoopbackManagedEndpoint(instance.Endpoint) {
		return ManagedInstance{}, fmt.Errorf("only verified loopback user-hosted instances may admit application scopes")
	}
	if err := b.verifyLocked(instance); err != nil {
		return ManagedInstance{}, err
	}
	instance.AuthorizedScopes = append([]string(nil), instance.AuthorizedScopes...)
	if !scopeAllowed(instance.AuthorizedScopes, scope) {
		instance.AuthorizedScopes = append(instance.AuthorizedScopes, scope)
	}
	b.instances[instance.ID] = instance
	if err := b.persistLocked(); err != nil {
		delete(b.instances, instance.ID)
		return ManagedInstance{}, err
	}
	return instance, nil
}

func isLoopbackManagedEndpoint(raw string) bool {
	endpoint, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || (endpoint.Scheme != "http" && endpoint.Scheme != "https") || endpoint.Hostname() == "" {
		return false
	}
	if strings.EqualFold(endpoint.Hostname(), "localhost") {
		return true
	}
	ip := net.ParseIP(endpoint.Hostname())
	return ip != nil && ip.IsLoopback()
}

func (b *Broker) Acquire(resource, scope string, ttl time.Duration) (Lease, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if strings.TrimSpace(resource) == "" || strings.TrimSpace(scope) == "" || ttl <= 0 {
		return Lease{}, fmt.Errorf("resource, scope, and positive lease ttl are required")
	}
	for _, instance := range b.instances {
		if instance.Resource != resource || instance.Provider != resourcedeployment.ProviderManagedShared || !scopeAllowed(instance.AuthorizedScopes, scope) {
			continue
		}
		if err := b.verifyLocked(instance); err != nil {
			continue
		}
		b.sequence++
		lease := Lease{ID: fmt.Sprintf("lease-%d", b.sequence), InstanceID: instance.ID, Scope: scope, ExpiresAt: b.now().Add(ttl)}
		b.leases[lease.ID] = lease
		if err := b.persistLocked(); err != nil {
			delete(b.leases, lease.ID)
			return Lease{}, err
		}
		return lease, nil
	}
	return Lease{}, fmt.Errorf("no verified shared instance authorizes scope %q for %s", scope, resource)
}

func (b *Broker) persistLocked() error {
	if b.store == nil {
		return nil
	}
	return b.store.Save(BrokerState{Instances: b.instances, Leases: b.leases, Sequence: b.sequence})
}

// AuthorizeUse validates an unexpired scope lease and returns the verified
// service identity. The boolean is deliberately absent: every denial is a
// typed error callers must surface rather than silently falling back to a port.
func (b *Broker) AuthorizeUse(leaseID, resource, scope string) (ManagedInstance, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	lease, ok := b.leases[leaseID]
	if !ok || !b.now().Before(lease.ExpiresAt) || lease.Scope != scope {
		return ManagedInstance{}, fmt.Errorf("lease is missing, expired, or scoped to another application")
	}
	instance, ok := b.instances[lease.InstanceID]
	if !ok || instance.Resource != resource {
		return ManagedInstance{}, fmt.Errorf("lease does not resolve a verified %s instance", resource)
	}
	return instance, nil
}

// AuthorizeManagement is intentionally owner-only. Lease holders and attach-
// only integrations must never initialize, unseal, stop, or rewrite a service.
func (b *Broker) AuthorizeManagement(instanceID, ownerScope string) (ManagedInstance, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	instance, ok := b.instances[instanceID]
	if !ok || instance.OwnerScope != ownerScope {
		return ManagedInstance{}, fmt.Errorf("management requires the registered owner scope")
	}
	return instance, nil
}

// IssueScopedCredential authorizes use before delegating to the resource's
// credential policy adapter. This keeps policy implementation resource-native
// without allowing the adapter to turn a raw endpoint into trusted ownership.
func (b *Broker) IssueScopedCredential(leaseID, resource, scope string, issuer CredentialIssuer) (ScopedCredential, error) {
	if issuer == nil {
		return ScopedCredential{}, fmt.Errorf("scoped credential issuer is required")
	}
	instance, err := b.AuthorizeUse(leaseID, resource, scope)
	if err != nil {
		return ScopedCredential{}, err
	}
	b.mu.Lock()
	lease, ok := b.leases[leaseID]
	b.mu.Unlock()
	if !ok {
		return ScopedCredential{}, fmt.Errorf("authorized lease disappeared before credential issuance")
	}
	credential, err := issuer.IssueScopedCredential(instance, lease)
	if err != nil {
		return ScopedCredential{}, err
	}
	if credential.LeaseID != lease.ID || credential.Resource != resource || credential.Scope != scope || !credential.ExpiresAt.Equal(lease.ExpiresAt) || strings.TrimSpace(credential.Credential) == "" {
		return ScopedCredential{}, fmt.Errorf("credential issuer returned a credential outside the authorized lease scope")
	}
	return credential, nil
}

func scopeAllowed(scopes []string, scope string) bool {
	for _, allowed := range scopes {
		if allowed == scope {
			return true
		}
	}
	return false
}
