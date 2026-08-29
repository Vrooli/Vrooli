package credentialauthority

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"sync"
)

var resolveOrMintMu sync.Mutex

// Witness testifies whether a deployment has history. Implementations must
// store that history with the data protected by the credential, never in the
// credential store whose absence is being interpreted.
type Witness interface {
	Minted(identity Identity, field string) (bool, error)
	RecordMint(identity Identity, field string) error
}

// MintRefusedError reports that a generated credential disappeared from a
// deployment that has already used it. Re-minting is intentionally an explicit
// operator decision because it may invalidate tokens or sealed data.
type MintRefusedError struct {
	Identity Identity
	Field    string
}

func (e *MintRefusedError) Error() string {
	return fmt.Sprintf(
		"%s:%s has been minted on this deployment before and is now absent "+
			"from the credential store. This is credential loss, not a first "+
			"start. Restore the store from your recovery bundle "+
			"(`vrooli credentials recovery restore --input <bundle>`). "+
			"Re-minting would invalidate every token and every value sealed "+
			"with it; if that is genuinely what you want, pass "+
			"--accept-credential-loss.",
		e.Identity, e.Field,
	)
}

// ResolutionError preserves the credential authority's three-way failure
// taxonomy while adding the address an operator needs to remediate it.
type ResolutionError struct {
	Identity Identity
	Field    string
	Err      error
}

func (e *ResolutionError) Error() string {
	return fmt.Sprintf("resolve credential %s:%s: %v", e.Identity, e.Field, e.Err)
}

func (e *ResolutionError) Unwrap() error { return e.Err }

// Require resolves an operator-supplied credential. It never converts a
// provider fault or an empty value into a successful, unconfigured answer.
func (a *Authority) Require(identity Identity, field string) (string, error) {
	identity, field, err := normalizeAddress(identity, field)
	if err != nil {
		return "", err
	}
	if a == nil || a.inner == nil {
		return "", &ResolutionError{Identity: identity, Field: field, Err: ErrProviderAbsent}
	}
	value, err := a.Resolve(identity, field)
	if err == nil && strings.TrimSpace(value) == "" {
		err = ErrUnconfigured
	}
	if err != nil {
		return "", &ResolutionError{Identity: identity, Field: field, Err: err}
	}
	return value, nil
}

// ResolveOrMint resolves a generated credential or creates it exactly once in
// this process. Provider failures always fail closed. A witness turns a clean
// "not configured" response into an explicit credential-loss refusal when the
// deployment has history.
func (a *Authority) ResolveOrMint(identity Identity, field string, witness Witness, mint func() (string, error)) (string, error) {
	return a.resolveOrMint(identity, field, witness, mint, false)
}

// ResolveOrMintWithCredentialLossOverride resolves or mints a generated
// credential while explicitly accepting replacement after a witness reports
// that the original value was lost. Provider failures and persistence errors
// remain fail-closed; the override only bypasses MintRefusedError.
func (a *Authority) ResolveOrMintWithCredentialLossOverride(identity Identity, field string, witness Witness, mint func() (string, error)) (string, error) {
	return a.resolveOrMint(identity, field, witness, mint, true)
}

//nolint:gocyclo // Resolution deliberately keeps provider, witness, mint, persistence, and override failures explicit.
func (a *Authority) resolveOrMint(identity Identity, field string, witness Witness, mint func() (string, error), acceptCredentialLoss bool) (string, error) {
	identity, field, err := normalizeAddress(identity, field)
	if err != nil {
		return "", err
	}
	if mint == nil {
		return "", fmt.Errorf("mint function is required for %s:%s", identity, field)
	}
	if a == nil || a.inner == nil {
		return "", &ResolutionError{Identity: identity, Field: field, Err: ErrProviderAbsent}
	}

	// Default returns a fresh public wrapper around a process-wide internal
	// authority. Resolve and Put are individually synchronized by that internal
	// authority, but the read-check-write sequence must also be indivisible
	// across distinct wrappers or two startup paths can both mint.
	resolveOrMintMu.Lock()
	defer resolveOrMintMu.Unlock()

	value, err := a.Resolve(identity, field)
	if err == nil && strings.TrimSpace(value) == "" {
		err = ErrUnconfigured
	}
	switch {
	case err == nil:
		if witness != nil {
			// This also safely adopts credentials minted before witnesses were
			// introduced. Recording is idempotent by contract.
			minted, witnessErr := witness.Minted(identity, field)
			if witnessErr != nil {
				return "", fmt.Errorf("witness check %s:%s: %w", identity, field, witnessErr)
			}
			if !minted {
				if witnessErr := witness.RecordMint(identity, field); witnessErr != nil {
					return "", fmt.Errorf("record existing mint %s:%s: %w", identity, field, witnessErr)
				}
			}
		}
		return value, nil
	case errors.Is(err, ErrUnconfigured):
		// Continue to the witness gate.
	default:
		return "", &ResolutionError{Identity: identity, Field: field, Err: err}
	}

	if witness != nil {
		minted, witnessErr := witness.Minted(identity, field)
		if witnessErr != nil {
			return "", fmt.Errorf("witness check %s:%s: %w", identity, field, witnessErr)
		}
		if minted && !acceptCredentialLoss {
			return "", &MintRefusedError{Identity: identity, Field: field}
		}
	}

	generated, err := mint()
	if err != nil {
		return "", fmt.Errorf("mint %s:%s: %w", identity, field, err)
	}
	if strings.TrimSpace(generated) == "" {
		return "", fmt.Errorf("mint %s:%s returned an empty credential", identity, field)
	}
	if err := a.Put(identity, field, generated); err != nil {
		return "", fmt.Errorf("store generated %s:%s: %w", identity, field, err)
	}
	if witness != nil {
		if err := witness.RecordMint(identity, field); err != nil {
			// The stored value remains recoverable. A subsequent call observes it
			// and retries this idempotent witness write before returning it.
			return "", fmt.Errorf("record mint %s:%s: %w", identity, field, err)
		}
	}
	return generated, nil
}

// RandomBase64 returns n cryptographically random bytes encoded without
// padding using the URL-safe base64 alphabet.
func RandomBase64(n int) (string, error) {
	if n <= 0 {
		return "", fmt.Errorf("random byte count must be positive")
	}
	raw := make([]byte, n)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("read cryptographic randomness: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func normalizeAddress(identity Identity, field string) (Identity, string, error) {
	normalizedIdentity, err := ParseIdentity(string(identity))
	if err != nil {
		return "", "", err
	}
	field = strings.TrimSpace(field)
	if field == "" || strings.ContainsAny(field, "/\\") {
		return "", "", fmt.Errorf("credential field is required and cannot contain a path separator")
	}
	return normalizedIdentity, field, nil
}
