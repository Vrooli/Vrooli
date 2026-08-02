package secrets

import (
	"errors"
	"fmt"
	"testing"

	"github.com/vrooli/vrooli/internal/resources/securestore"
)

// authorityStore is a conforming in-memory Store: it answers a missing key
// with securestore.ErrNotFound, which is what separates "no value" from
// "backend broken" everywhere else in this package.
type authorityStore struct {
	values map[string]string
	calls  int
}

func (s *authorityStore) Put(service, key, value string) error {
	s.calls++
	if s.values == nil {
		s.values = map[string]string{}
	}
	s.values[service+"/"+key] = value
	return nil
}

func (s *authorityStore) Get(service, key string) (string, error) {
	s.calls++
	value, ok := s.values[service+"/"+key]
	if !ok {
		return "", fmt.Errorf("%w: %s/%s", securestore.ErrNotFound, service, key)
	}
	return value, nil
}

func (s *authorityStore) Delete(service, key string) error {
	s.calls++
	delete(s.values, service+"/"+key)
	return nil
}

// failingStore reports a fixed transport failure for every operation, standing
// in for a broken keyring session or a platform with no adapter.
type failingStore struct{ err error }

func (s failingStore) Put(string, string, string) error { return s.err }
func (s failingStore) Get(string, string) (string, error) {
	return "", s.err
}
func (s failingStore) Delete(string, string) error { return s.err }

func TestAuthorityStoresAndInjectsOnlyScopedValue(t *testing.T) {
	authority, err := NewAuthority(&authorityStore{})
	if err != nil {
		t.Fatal(err)
	}
	id, err := ParseIdentity("vrooli/openrouter")
	if err != nil {
		t.Fatal(err)
	}
	if err := authority.Put(id, "api-key", "test-value"); err != nil {
		t.Fatal(err)
	}
	status := authority.Status(id, "api-key")
	if !status.Configured || status.ProviderState != ProviderAvailable {
		t.Fatalf("unsafe status: %+v", status)
	}
	env := map[string]string{}
	if err := authority.Inject(id, "api-key", "OPENROUTER_API_KEY", env); err != nil {
		t.Fatal(err)
	}
	if env["OPENROUTER_API_KEY"] != "test-value" {
		t.Fatal("scoped injection did not resolve value")
	}
	if err := authority.Delete(id, "api-key"); err != nil {
		t.Fatal(err)
	}
	if authority.Status(id, "api-key").Configured {
		t.Fatal("deleted credential remains configured")
	}
}

func TestAuthorityRejectsInvalidIdentityAndNeverFallsBack(t *testing.T) {
	if _, err := ParseIdentity("openrouter"); err == nil {
		t.Fatal("unnamespaced identity accepted")
	}
	if _, err := NewAuthority(nil); !errors.Is(err, ErrProviderAbsent) {
		t.Fatalf("error = %v, want ErrProviderAbsent", err)
	}
}

// TestCredentialFailureTaxonomyIsThreeDistinctConditions is the core guarantee
// of this package: a host fault, a missing backend, and an unset value are
// separately detectable, and none of them can masquerade as another.
func TestCredentialFailureTaxonomyIsThreeDistinctConditions(t *testing.T) {
	identity, err := ParseIdentity("vrooli/openrouter")
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name          string
		store         securestore.Store
		want          error
		notWant       []error
		providerState ProviderState
	}{
		{
			name:          "unreachable backend is a provider outage",
			store:         failingStore{err: fmt.Errorf("%w: broken session bus", securestore.ErrUnavailable)},
			want:          ErrProviderUnavailable,
			notWant:       []error{ErrUnconfigured, ErrProviderAbsent},
			providerState: ProviderUnavailable,
		},
		{
			name:          "unclassified backend failure is still a provider outage, never unconfigured",
			store:         failingStore{err: errors.New("some adapter forgot to classify this")},
			want:          ErrProviderUnavailable,
			notWant:       []error{ErrUnconfigured, ErrProviderAbsent},
			providerState: ProviderUnavailable,
		},
		{
			name:          "host with no adapter is a provider absence",
			store:         securestore.Absent("no adapter on this platform"),
			want:          ErrProviderAbsent,
			notWant:       []error{ErrUnconfigured, ErrProviderUnavailable},
			providerState: ProviderAbsent,
		},
		{
			name:          "working backend with no stored value is unconfigured",
			store:         &authorityStore{},
			want:          ErrUnconfigured,
			notWant:       []error{ErrProviderAbsent, ErrProviderUnavailable},
			providerState: ProviderAvailable,
		},
		{
			name:          "working backend with an empty stored value is unconfigured",
			store:         &authorityStore{values: map[string]string{credentialService + "/vrooli/openrouter:api-key": "   "}},
			want:          ErrUnconfigured,
			notWant:       []error{ErrProviderAbsent, ErrProviderUnavailable},
			providerState: ProviderAvailable,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			authority, err := NewAuthority(testCase.store)
			if err != nil {
				t.Fatal(err)
			}
			injectErr := authority.Inject(identity, "api-key", "OPENROUTER_API_KEY", map[string]string{})
			if !errors.Is(injectErr, testCase.want) {
				t.Fatalf("Inject error = %v, want %v", injectErr, testCase.want)
			}
			for _, forbidden := range testCase.notWant {
				if errors.Is(injectErr, forbidden) {
					t.Fatalf("Inject error = %v must not also be %v", injectErr, forbidden)
				}
			}
			if state := ProviderStateFor(injectErr); state != testCase.providerState {
				t.Fatalf("ProviderStateFor(%v) = %q, want %q", injectErr, state, testCase.providerState)
			}
			if state := authority.Status(identity, "api-key").ProviderState; state != testCase.providerState {
				t.Fatalf("Status provider state = %q, want %q", state, testCase.providerState)
			}
		})
	}
}

// TestAvailabilityIsLazyAndProbesAtMostOnce pins the property that broke the
// start path: constructing an authority must not touch the operator keyring.
func TestAvailabilityIsLazyAndProbesAtMostOnce(t *testing.T) {
	store := &authorityStore{}
	authority, err := NewAuthority(store)
	if err != nil {
		t.Fatal(err)
	}
	if store.calls != 0 {
		t.Fatalf("NewAuthority performed %d store calls, want 0", store.calls)
	}
	if err := authority.Availability(); err != nil {
		t.Fatalf("Availability() = %v, want nil for a reachable backend", err)
	}
	after := store.calls
	if after == 0 {
		t.Fatal("Availability() did not probe the store")
	}
	for range 5 {
		if err := authority.Availability(); err != nil {
			t.Fatal(err)
		}
	}
	if store.calls != after {
		t.Fatalf("Availability() probed %d extra times, want a cached result", store.calls-after)
	}
}

func TestAvailabilityReportsProviderConditions(t *testing.T) {
	for _, testCase := range []struct {
		name  string
		store securestore.Store
		want  error
	}{
		{"absent", securestore.Absent("no adapter"), ErrProviderAbsent},
		{"unavailable", securestore.Unavailable("session unreachable"), ErrProviderUnavailable},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			authority, err := NewAuthority(testCase.store)
			if err != nil {
				t.Fatal(err)
			}
			if err := authority.Availability(); !errors.Is(err, testCase.want) {
				t.Fatalf("Availability() = %v, want %v", err, testCase.want)
			}
		})
	}
}
