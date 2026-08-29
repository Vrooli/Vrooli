//nolint:goconst // test data deliberately reuses stable credential fixtures.
package credentialauthority

import (
	"encoding/base64"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/vrooli/vrooli/internal/resources/securestore"
)

const testIdentity Identity = "vrooli/test-scenario"

type mintStore struct {
	mu       sync.Mutex
	value    string
	getErr   error
	putErr   error
	putCount int
}

func (s *mintStore) Put(_, _, value string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.putCount++
	if s.putErr != nil {
		return s.putErr
	}
	s.value = value
	s.getErr = nil
	return nil
}

func (s *mintStore) Get(_, _ string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.value, s.getErr
}

func (s *mintStore) Delete(_, _ string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.value = ""
	s.getErr = securestore.ErrNotFound
	return nil
}

type mintWitness struct {
	minted      bool
	mintedErr   error
	recordErr   error
	recordCount int
}

func (w *mintWitness) Minted(Identity, string) (bool, error) {
	return w.minted, w.mintedErr
}

func (w *mintWitness) RecordMint(Identity, string) error {
	w.recordCount++
	if w.recordErr == nil {
		w.minted = true
	}
	return w.recordErr
}

func newTestAuthority(t *testing.T, store securestore.Store) *Authority {
	t.Helper()
	authority, err := NewAuthority(store)
	if err != nil {
		t.Fatalf("NewAuthority() error = %v", err)
	}
	return authority
}

func TestResolveOrMintStoredValueDoesNotMint(t *testing.T) {
	store := &mintStore{value: "stored"}
	witness := &mintWitness{minted: true}
	authority := newTestAuthority(t, store)
	mintCount := 0

	got, err := authority.ResolveOrMint(testIdentity, "secret", witness, func() (string, error) {
		mintCount++
		return "replacement", nil
	})
	if err != nil {
		t.Fatalf("ResolveOrMint() error = %v", err)
	}
	if got != "stored" || mintCount != 0 || store.putCount != 0 || witness.recordCount != 0 {
		t.Fatalf("stored resolution = %q, mintCount=%d, puts=%d, records=%d", got, mintCount, store.putCount, witness.recordCount)
	}
}

func TestResolveOrMintAdoptsStoredValueIntoWitness(t *testing.T) {
	store := &mintStore{value: "stored"}
	witness := &mintWitness{}
	authority := newTestAuthority(t, store)

	got, err := authority.ResolveOrMint(testIdentity, "secret", witness, func() (string, error) {
		t.Fatal("mint called for stored value")
		return "", nil
	})
	if err != nil {
		t.Fatalf("ResolveOrMint() error = %v", err)
	}
	if got != "stored" || witness.recordCount != 1 || !witness.minted {
		t.Fatalf("stored adoption = %q, records=%d, minted=%t", got, witness.recordCount, witness.minted)
	}
}

func TestResolveOrMintRefusesProviderFailures(t *testing.T) {
	tests := []struct {
		name     string
		storeErr error
		want     error
	}{
		{name: "unavailable", storeErr: securestore.ErrUnavailable, want: ErrProviderUnavailable},
		{name: "absent", storeErr: securestore.ErrAbsent, want: ErrProviderAbsent},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			store := &mintStore{getErr: testCase.storeErr}
			authority := newTestAuthority(t, store)
			mintCount := 0
			_, err := authority.ResolveOrMint(testIdentity, "secret", nil, func() (string, error) {
				mintCount++
				return "replacement", nil
			})
			if !errors.Is(err, testCase.want) {
				t.Fatalf("ResolveOrMint() error = %v, want errors.Is(%v)", err, testCase.want)
			}
			var resolutionErr *ResolutionError
			if !errors.As(err, &resolutionErr) {
				t.Fatalf("ResolveOrMint() error type = %T, want *ResolutionError", err)
			}
			if mintCount != 0 || store.putCount != 0 {
				t.Fatalf("provider failure minted=%d puts=%d", mintCount, store.putCount)
			}
		})
	}
}

func TestResolveOrMintRefusesCredentialLoss(t *testing.T) {
	store := &mintStore{getErr: securestore.ErrNotFound}
	witness := &mintWitness{minted: true}
	authority := newTestAuthority(t, store)
	mintCount := 0

	_, err := authority.ResolveOrMint(testIdentity, "secret", witness, func() (string, error) {
		mintCount++
		return "replacement", nil
	})
	var refused *MintRefusedError
	if !errors.As(err, &refused) {
		t.Fatalf("ResolveOrMint() error = %v, want *MintRefusedError", err)
	}
	if mintCount != 0 || store.putCount != 0 || witness.recordCount != 0 {
		t.Fatalf("credential loss minted=%d puts=%d records=%d", mintCount, store.putCount, witness.recordCount)
	}
}

func TestResolveOrMintWithCredentialLossOverrideReplacesAndRecords(t *testing.T) {
	store := &mintStore{getErr: securestore.ErrNotFound}
	witness := &mintWitness{minted: true}
	authority := newTestAuthority(t, store)

	got, err := authority.ResolveOrMintWithCredentialLossOverride(testIdentity, "secret", witness, func() (string, error) {
		return "replacement", nil
	})
	if err != nil {
		t.Fatalf("ResolveOrMintWithCredentialLossOverride() error = %v", err)
	}
	if got != "replacement" || store.putCount != 1 || witness.recordCount != 1 {
		t.Fatalf("override resolution = %q, puts=%d, records=%d", got, store.putCount, witness.recordCount)
	}
}

func TestResolveOrMintMintsAndRecordsVirginDeployment(t *testing.T) {
	store := &mintStore{getErr: securestore.ErrNotFound}
	witness := &mintWitness{}
	authority := newTestAuthority(t, store)
	mintCount := 0

	got, err := authority.ResolveOrMint(testIdentity, "secret", witness, func() (string, error) {
		mintCount++
		return "generated", nil
	})
	if err != nil {
		t.Fatalf("ResolveOrMint() error = %v", err)
	}
	if got != "generated" || mintCount != 1 || store.putCount != 1 || witness.recordCount != 1 {
		t.Fatalf("virgin mint = %q, mints=%d puts=%d records=%d", got, mintCount, store.putCount, witness.recordCount)
	}
}

func TestResolveOrMintNilWitnessMints(t *testing.T) {
	store := &mintStore{getErr: securestore.ErrNotFound}
	authority := newTestAuthority(t, store)
	got, err := authority.ResolveOrMint(testIdentity, "secret", nil, func() (string, error) { return "generated", nil })
	if err != nil || got != "generated" || store.putCount != 1 {
		t.Fatalf("ResolveOrMint() = %q, %v, puts=%d", got, err, store.putCount)
	}
}

func TestResolveOrMintRetriesFailedWitnessRecordForStoredValue(t *testing.T) {
	store := &mintStore{getErr: securestore.ErrNotFound}
	witness := &mintWitness{recordErr: errors.New("database unavailable")}
	authority := newTestAuthority(t, store)

	_, err := authority.ResolveOrMint(testIdentity, "secret", witness, func() (string, error) { return "generated", nil })
	if err == nil || store.value != "generated" || witness.recordCount != 1 {
		t.Fatalf("first ResolveOrMint() error=%v value=%q records=%d", err, store.value, witness.recordCount)
	}
	witness.recordErr = nil
	got, err := authority.ResolveOrMint(testIdentity, "secret", witness, func() (string, error) {
		t.Fatal("mint retried despite stored value")
		return "", nil
	})
	if err != nil || got != "generated" || witness.recordCount != 2 || !witness.minted {
		t.Fatalf("retry ResolveOrMint() = %q, %v, records=%d minted=%t", got, err, witness.recordCount, witness.minted)
	}
}

func TestResolveOrMintSerializesConcurrentMints(t *testing.T) {
	store := &mintStore{getErr: securestore.ErrNotFound}
	authority := newTestAuthority(t, store)
	start := make(chan struct{})
	results := make(chan string, 2)
	errs := make(chan error, 2)
	mintCount := 0
	var mintMu sync.Mutex
	mint := func() (string, error) {
		mintMu.Lock()
		mintCount++
		mintMu.Unlock()
		<-start
		return "generated", nil
	}
	for range 2 {
		go func() {
			value, err := authority.ResolveOrMint(testIdentity, "secret", nil, mint)
			results <- value
			errs <- err
		}()
	}
	close(start)
	for range 2 {
		if err := <-errs; err != nil {
			t.Fatalf("ResolveOrMint() error = %v", err)
		}
		if got := <-results; got != "generated" {
			t.Fatalf("ResolveOrMint() value = %q", got)
		}
	}
	if mintCount != 1 || store.putCount != 1 {
		t.Fatalf("concurrent mints=%d puts=%d, want 1 each", mintCount, store.putCount)
	}
}

func TestResolveOrMintSerializesConcurrentMintsAcrossAuthorityWrappers(t *testing.T) {
	store := &mintStore{getErr: securestore.ErrNotFound}
	firstAuthority := newTestAuthority(t, store)
	secondAuthority := newTestAuthority(t, store)
	start := make(chan struct{})
	results := make(chan string, 2)
	errs := make(chan error, 2)
	mintCount := 0
	var mintMu sync.Mutex
	mint := func() (string, error) {
		mintMu.Lock()
		mintCount++
		mintMu.Unlock()
		<-start
		return "generated", nil
	}
	for _, authority := range []*Authority{firstAuthority, secondAuthority} {
		go func() {
			value, err := authority.ResolveOrMint(testIdentity, "secret", nil, mint)
			results <- value
			errs <- err
		}()
	}
	close(start)
	for range 2 {
		if err := <-errs; err != nil {
			t.Fatalf("ResolveOrMint() error = %v", err)
		}
		if got := <-results; got != "generated" {
			t.Fatalf("ResolveOrMint() value = %q", got)
		}
	}
	if mintCount != 1 || store.putCount != 1 {
		t.Fatalf("concurrent cross-wrapper mints=%d puts=%d, want 1 each", mintCount, store.putCount)
	}
}

func TestRequirePreservesFailureTaxonomyAndRejectsEmpty(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		storeErr error
		want     error
	}{
		{name: "unconfigured", storeErr: securestore.ErrNotFound, want: ErrUnconfigured},
		{name: "empty", value: "", want: ErrUnconfigured},
		{name: "unavailable", storeErr: securestore.ErrUnavailable, want: ErrProviderUnavailable},
		{name: "absent", storeErr: securestore.ErrAbsent, want: ErrProviderAbsent},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			authority := newTestAuthority(t, &mintStore{value: testCase.value, getErr: testCase.storeErr})
			value, err := authority.Require(testIdentity, "secret")
			if value != "" || !errors.Is(err, testCase.want) {
				t.Fatalf("Require() = %q, %v, want empty and errors.Is(%v)", value, err, testCase.want)
			}
			var resolutionErr *ResolutionError
			if !errors.As(err, &resolutionErr) {
				t.Fatalf("Require() error type = %T, want *ResolutionError", err)
			}
		})
	}
}

func TestHelpersNormalizeCredentialAddress(t *testing.T) {
	store := &mintStore{getErr: securestore.ErrNotFound}
	authority := newTestAuthority(t, store)
	witness := &mintWitness{}

	got, err := authority.ResolveOrMint(" /VROOLI/Test-Scenario/ ", " secret ", witness, func() (string, error) {
		return "generated", nil
	})
	if err != nil || got != "generated" {
		t.Fatalf("ResolveOrMint() = %q, %v", got, err)
	}
	stored, err := authority.Require(testIdentity, "secret")
	if err != nil || stored != "generated" {
		t.Fatalf("Require(normalized address) = %q, %v", stored, err)
	}
}

func TestHelpersRejectZeroValueAuthority(t *testing.T) {
	var authority Authority
	for _, testCase := range []struct {
		name string
		call func() (string, error)
	}{
		{name: "require", call: func() (string, error) { return authority.Require(testIdentity, "secret") }},
		{name: "resolve or mint", call: func() (string, error) {
			return authority.ResolveOrMint(testIdentity, "secret", nil, func() (string, error) { return "generated", nil })
		}},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			value, err := testCase.call()
			if value != "" || !errors.Is(err, ErrProviderAbsent) {
				t.Fatalf("helper = %q, %v, want empty and ErrProviderAbsent", value, err)
			}
		})
	}
}

func TestMintRefusedErrorMessageIsActionable(t *testing.T) {
	message := (&MintRefusedError{Identity: testIdentity, Field: "secret"}).Error()
	for _, fragment := range []string{string(testIdentity), "secret", "recovery restore", "--accept-credential-loss"} {
		if !strings.Contains(message, fragment) {
			t.Errorf("MintRefusedError.Error() = %q, missing %q", message, fragment)
		}
	}
}

func TestRandomBase64(t *testing.T) {
	value, err := RandomBase64(32)
	if err != nil {
		t.Fatalf("RandomBase64() error = %v", err)
	}
	raw, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil || len(raw) != 32 {
		t.Fatalf("RandomBase64() decoded length = %d, error = %v", len(raw), err)
	}
	if _, err := RandomBase64(0); err == nil {
		t.Fatal("RandomBase64(0) error = nil")
	}
}
