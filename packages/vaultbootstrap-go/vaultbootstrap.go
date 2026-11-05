// Package vaultbootstrap owns the one sequence that brings a Vrooli-managed
// Vault instance from "reachable" to "usable": initialize, persist recovery
// material, unseal, mount KV v2, and mint a scoped token.
//
// It exists because that sequence was implemented twice — once in the control
// plane and once in the desktop runtime — and the two filed their recovery
// material under different service names. Recovery material stored under a name
// only one implementation knows is material a backup, an audit, or a restore
// silently misses, and the halves had already begun to drift: one kept a
// legacy-marker migration path the other never had.
//
// The package holds the shared sequence and the storage seam. It deliberately
// does not own lifecycle policy, broker registration, or resource plans: those
// differ per tier and belong to their callers.
package vaultbootstrap

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
	"github.com/vrooli/vrooli/packages/resource-deployment/securestore"
)

// Service is the single secure-store namespace for Vault recovery material.
//
// One namespace is a prerequisite for backing this material up at all. While
// the same product filed material under vrooli.resource.vault,
// vrooli.resource.vault.private, and vrooli.desktop.vault, any recovery story
// had to know all three or quietly cover a subset.
const Service = "vrooli.resource.vault"

// Material is what makes a sealed Vault usable again. The two halves are not
// equally replaceable and callers should not treat them as one secret:
//
//   - UnsealKey is irreplaceable. Without it the instance stays sealed forever
//     and everything inside it is gone.
//   - RootToken is management-only, and Vault can regenerate one from a quorum
//     of unseal keys. Vrooli initializes with a threshold of 1, so the single
//     unseal key is that quorum.
//
// That asymmetry is why the fields are named rather than collapsed into a
// []string: a backup needs the unseal key and should not carry the root token.
type Material struct {
	RootToken string `json:"root_token"`
	UnsealKey string `json:"unseal_key"`
}

// Valid reports whether both halves are present. A partially stored Material
// cannot restore an instance, so callers verify before relying on it.
func (m Material) Valid() bool {
	return strings.TrimSpace(m.RootToken) != "" && strings.TrimSpace(m.UnsealKey) != ""
}

// State is the coarse lifecycle position of an instance.
type State string

const (
	StateUninitialized State = "uninitialized"
	StateSealed        State = "sealed"
	StateUnsealed      State = "unsealed"
)

// HTTPStatusError carries a non-2xx status so a caller can recognize the one
// case that is not a failure — a KV mount that already exists.
type HTTPStatusError struct{ StatusCode int }

func (e HTTPStatusError) Error() string { return fmt.Sprintf("Vault returned HTTP %d", e.StatusCode) }

// Client performs the bootstrap sequence against one endpoint.
type Client struct {
	Endpoint string
	// HTTP is optional; a nil client uses a bounded default rather than
	// http.DefaultClient, which has no timeout.
	HTTP *http.Client
}

func (c Client) httpClient() *http.Client {
	if c.HTTP != nil {
		return c.HTTP
	}
	return &http.Client{Timeout: 30 * time.Second} //nolint:mnd // Vault bootstrap deadline is an operational contract
}

// Request performs one Vault API call. Body and response are JSON; the token,
// when present, travels in the X-Vault-Token header and never in the path.
func (c Client) Request(ctx context.Context, method, path, token string, input, output any) error {
	var body io.Reader
	if input != nil {
		encoded, err := json.Marshal(input)
		if err != nil {
			return err
		}
		body = bytes.NewReader(encoded)
	}
	request, err := http.NewRequestWithContext(ctx, method, strings.TrimSuffix(c.Endpoint, "/")+path, body)
	if err != nil {
		return err
	}
	if input != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	if strings.TrimSpace(token) != "" {
		request.Header.Set("X-Vault-Token", token)
	}
	response, err := c.httpClient().Do(request)
	if err != nil {
		return err
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		// The body may echo request content, so it is not included: a Vault
		// error must not become a place a token appears in a log.
		return HTTPStatusError{StatusCode: response.StatusCode}
	}
	if output == nil {
		return nil
	}
	return json.NewDecoder(response.Body).Decode(output)
}

// WaitReachable blocks until the instance answers its seal-status endpoint.
func (c Client) WaitReachable(parent context.Context, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()
	for {
		if err := c.Request(ctx, http.MethodGet, "/v1/sys/seal-status", "", nil, nil); err == nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("wait for Vault reachability at %s: %w", c.Endpoint, ctx.Err())
		case <-time.After(100 * time.Millisecond): //nolint:mnd // polling interval bounds bootstrap readiness latency
		}
	}
}

// LifecycleState reports where the instance is, so a caller can tell an
// instance that needs initializing from one that only needs unsealing.
func (c Client) LifecycleState(ctx context.Context) (State, error) {
	var status struct {
		Initialized bool `json:"initialized"`
		Sealed      bool `json:"sealed"`
	}
	if err := c.Request(ctx, http.MethodGet, "/v1/sys/seal-status", "", nil, &status); err != nil {
		return "", err
	}
	return Classify(status.Initialized, status.Sealed), nil
}

// Classify maps a seal-status answer onto a lifecycle position.
func Classify(initialized, sealed bool) State {
	switch {
	case !initialized:
		return StateUninitialized
	case sealed:
		return StateSealed
	default:
		return StateUnsealed
	}
}

// Initialize creates a new instance and returns the only copy of its recovery
// material. A caller that does not persist the result has permanently lost the
// instance, so every caller stores it before doing anything else.
//
// One share with a threshold of one is deliberate: Vrooli holds the key in a
// secure store rather than splitting it across operators, and a threshold above
// one would mean an instance nobody can unseal unattended.
func (c Client) Initialize(ctx context.Context) (Material, error) {
	var initialized struct {
		Keys      []string `json:"keys"`
		RootToken string   `json:"root_token"`
	}
	err := c.Request(ctx, http.MethodPut, "/v1/sys/init",
		"", map[string]int{"secret_shares": 1, "secret_threshold": 1}, &initialized)
	if err != nil {
		return Material{}, fmt.Errorf("initialize Vault: %w", err)
	}
	if len(initialized.Keys) != 1 || strings.TrimSpace(initialized.RootToken) == "" {
		return Material{}, fmt.Errorf("initialize Vault: response did not carry one unseal key and a root token")
	}
	return Material{RootToken: initialized.RootToken, UnsealKey: initialized.Keys[0]}, nil
}

// Unseal opens a sealed instance with stored material.
func (c Client) Unseal(ctx context.Context, material Material) error {
	if strings.TrimSpace(material.UnsealKey) == "" {
		return fmt.Errorf("unseal Vault: no unseal key")
	}
	if err := c.Request(ctx, http.MethodPut, "/v1/sys/unseal", "",
		map[string]string{"key": material.UnsealKey}, nil); err != nil {
		return fmt.Errorf("unseal Vault: %w", err)
	}
	return nil
}

// EnsureKVv2 mounts the KV v2 engine, treating an existing mount as success.
// Vault answers a duplicate mount with 400, which is the normal state on every
// restart after the first.
func (c Client) EnsureKVv2(ctx context.Context, rootToken string) error {
	err := c.Request(ctx, http.MethodPost, "/v1/sys/mounts/secret", rootToken,
		map[string]any{"type": "kv", "options": map[string]string{"version": "2"}}, nil)
	var status HTTPStatusError
	if err != nil && (!errors.As(err, &status) || status.StatusCode != http.StatusBadRequest) {
		return fmt.Errorf("enable Vault KV v2: %w", err)
	}
	return nil
}

// VerifyScopedOperation proves a minted token actually works before an instance
// is published as usable. Publishing an endpoint whose token silently fails is
// how a resource reports healthy and serves nothing.
func (c Client) VerifyScopedOperation(ctx context.Context, scopedToken string) error {
	if strings.TrimSpace(scopedToken) == "" {
		return fmt.Errorf("verify scoped Vault operation: no token")
	}
	if err := c.Request(ctx, http.MethodGet, "/v1/auth/token/lookup-self", scopedToken, nil, nil); err != nil {
		return fmt.Errorf("verify scoped Vault operation: %w", err)
	}
	return nil
}

// UnsealKeyField is the field the irreplaceable half is stored under.
const UnsealKeyField = "unseal-key"

// UnsealKeyIdentity is the durable name for one instance's unseal key.
//
// The unseal key lives in the credential authority rather than beside the root
// token, and that is the whole point: the authority's namespace is the one
// `recovery export --all`, `doctor`, and the recovery drill already cover. A
// value stored anywhere else is a value no backup captures — which is exactly
// how Vault material came to have no recovery path at all.
func UnsealKeyIdentity(instanceID string) (credentialauthority.Identity, error) {
	trimmed := strings.TrimSpace(instanceID)
	if trimmed == "" {
		return "", fmt.Errorf("vault instance id is required to name its unseal key")
	}
	return credentialauthority.ParseIdentity("vrooli/vault/" + trimmed)
}

// UnsealKeyStore is the credential-authority seam. It is an interface so the
// package does not force every caller to construct an authority, and so tests
// can supply one without a real backend.
type UnsealKeyStore interface {
	Put(identity credentialauthority.Identity, field, value string) error
	Resolve(identity credentialauthority.Identity, field string) (string, error)
}

// SaveUnsealKey records the irreplaceable half where recovery can reach it.
func SaveUnsealKey(keys UnsealKeyStore, instanceID, unsealKey string) error {
	if strings.TrimSpace(unsealKey) == "" {
		return fmt.Errorf("refusing to store an empty vault unseal key")
	}
	identity, err := UnsealKeyIdentity(instanceID)
	if err != nil {
		return err
	}
	if err := keys.Put(identity, UnsealKeyField, unsealKey); err != nil {
		return fmt.Errorf("store vault unseal key: %w", err)
	}
	stored, err := keys.Resolve(identity, UnsealKeyField)
	if err != nil {
		return fmt.Errorf("verify stored vault unseal key: %w", err)
	}
	if stored != unsealKey {
		return fmt.Errorf("verify stored vault unseal key: readback did not match")
	}
	return nil
}

// LoadUnsealKey reads the unseal key back, for a host restoring from a bundle
// that carried it without a root token.
func LoadUnsealKey(keys UnsealKeyStore, instanceID string) (string, bool, error) {
	identity, err := UnsealKeyIdentity(instanceID)
	if err != nil {
		return "", false, err
	}
	value, err := keys.Resolve(identity, UnsealKeyField)
	if err != nil {
		if errors.Is(err, credentialauthority.ErrUnconfigured) {
			return "", false, nil
		}
		return "", false, fmt.Errorf("read vault unseal key: %w", err)
	}
	if strings.TrimSpace(value) == "" {
		return "", false, nil
	}
	return value, true, nil
}

// Load reads stored material for an instance. found is false when the store
// answered cleanly and holds nothing, which is a normal first-run answer and
// never a fault.
func Load(store securestore.Store, keys UnsealKeyStore, instanceID string) (Material, bool, error) {
	raw, err := store.Get(Service, instanceID)
	if err != nil {
		if errors.Is(err, securestore.ErrNotFound) {
			return Material{}, false, nil
		}
		return Material{}, false, fmt.Errorf("read Vault recovery material: %w", err)
	}
	var material Material
	if err := json.Unmarshal([]byte(raw), &material); err != nil {
		return Material{}, false, fmt.Errorf("parse Vault recovery material: %w", err)
	}
	if !material.Valid() {
		return Material{}, false, fmt.Errorf("stored Vault recovery material is incomplete")
	}
	if keys != nil {
		_, found, keyErr := LoadUnsealKey(keys, instanceID)
		if keyErr == nil && !found {
			// Backfill is best effort. A locked or unavailable credential store must
			// never block Vault startup; doctor reports the missing recovery entry.
			_ = SaveUnsealKey(keys, instanceID, material.UnsealKey)
		}
	}
	return material, true, nil
}

// Save persists material and reads it back before reporting success.
//
// The readback is not defensive padding. This value is the only copy of the
// instance's recovery material, and a store that accepted a write it did not
// keep would leave a running Vault that nothing can ever unseal again.
// Save persists material. keys may be nil for callers that have no credential
// authority available; when it is supplied the unseal key is additionally
// written there, which is what puts it inside a recovery bundle.
//
// The unseal key is written FIRST. If the second write fails the operator has
// the half that cannot be regenerated, and a root token is re-mintable from it.
// Writing the blob first and failing afterwards would leave the reverse.
func Save(store securestore.Store, keys UnsealKeyStore, instanceID string, material Material) error {
	if !material.Valid() {
		return fmt.Errorf("refusing to store incomplete Vault recovery material")
	}
	if keys != nil {
		if err := SaveUnsealKey(keys, instanceID, material.UnsealKey); err != nil {
			return err
		}
	}
	encoded, err := json.Marshal(material)
	if err != nil {
		return err
	}
	if err := store.Put(Service, instanceID, string(encoded)); err != nil {
		return fmt.Errorf("securely store Vault recovery material: %w", err)
	}
	stored, err := store.Get(Service, instanceID)
	if err != nil {
		return fmt.Errorf("verify stored Vault recovery material: %w", err)
	}
	if stored != string(encoded) {
		return fmt.Errorf("verify stored Vault recovery material: readback did not match")
	}
	return nil
}

// GenerateRootToken mints a fresh root token from the unseal key.
//
// This is what makes "back up the unseal key, not the root token" a complete
// recovery story rather than half of one. A host restored from a bundle holds
// the irreplaceable half and nothing else; without this it could unseal the
// instance and then administer nothing — no KV mount, no scoped tokens.
//
// Vault's own guidance is that the initial root token should be revoked after
// setup precisely because it is re-mintable this way, so a bundle that carried
// it would widen the blast radius for no recovery benefit.
//
// The exchange is Vault's documented three steps: start an attempt (the server
// returns a nonce and a one-time pad), submit a quorum of unseal keys against
// that nonce, then unmask the returned token with the pad. Vrooli initializes
// with a threshold of one, so the single stored key is that quorum.
func (c Client) GenerateRootToken(ctx context.Context, unsealKey string) (string, error) {
	if strings.TrimSpace(unsealKey) == "" {
		return "", fmt.Errorf("generate Vault root token: no unseal key")
	}

	// Cancel any attempt left behind by an interrupted run. Vault refuses to
	// start a second one, so a crashed recovery would otherwise block every
	// later attempt with an error that names nothing an operator can act on.
	_ = c.Request(ctx, http.MethodDelete, "/v1/sys/generate-root/attempt", "", nil, nil)

	var attempt struct {
		Nonce string `json:"nonce"`
		OTP   string `json:"otp"`
	}
	if err := c.Request(ctx, http.MethodPut, "/v1/sys/generate-root/attempt", "", map[string]any{}, &attempt); err != nil {
		return "", fmt.Errorf("start Vault root generation: %w", err)
	}
	if strings.TrimSpace(attempt.Nonce) == "" || strings.TrimSpace(attempt.OTP) == "" {
		return "", fmt.Errorf("start Vault root generation: Vault returned no nonce or one-time pad")
	}

	var update struct {
		Complete     bool   `json:"complete"`
		EncodedToken string `json:"encoded_token"`
	}
	err := c.Request(ctx, http.MethodPut, "/v1/sys/generate-root/update", "",
		map[string]string{"key": unsealKey, "nonce": attempt.Nonce}, &update)
	if err != nil {
		return "", fmt.Errorf("submit Vault unseal key for root generation: %w", err)
	}
	if !update.Complete || strings.TrimSpace(update.EncodedToken) == "" {
		// Incomplete means Vault wants more key shares than we hold, which on a
		// threshold-of-one instance means the stored key is not one of its own.
		return "", fmt.Errorf("Vault root generation did not complete with the stored unseal key")
	}

	token, err := unmaskRootToken(update.EncodedToken, attempt.OTP)
	if err != nil {
		return "", err
	}
	return token, nil
}

// unmaskRootToken reverses Vault's one-time pad. The returned token is XORed
// with the pad so the value never crosses the wire in the clear, even to the
// operator who started the attempt.
func unmaskRootToken(encodedToken, otp string) (string, error) {
	// Vault emits the token with raw (unpadded) standard base64.
	masked, err := base64.RawStdEncoding.DecodeString(encodedToken)
	if err != nil {
		return "", fmt.Errorf("decode generated Vault root token: %w", err)
	}
	pad := []byte(otp)
	if len(masked) != len(pad) {
		// A length mismatch means the pad does not belong to this token, and
		// XORing anyway would yield a plausible-looking string that is not a
		// token — a failure that would surface much later as a bare 403.
		return "", fmt.Errorf("generated Vault root token is %d bytes but its one-time pad is %d", len(masked), len(pad))
	}
	unmasked := make([]byte, len(masked))
	for index := range masked {
		unmasked[index] = masked[index] ^ pad[index]
	}
	return string(unmasked), nil
}
