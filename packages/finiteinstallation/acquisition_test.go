package finiteinstallation

import (
	"errors"
	"reflect"
	"testing"

	internalca "github.com/vrooli/vrooli/internal/credentialauthority"
	"github.com/vrooli/vrooli/internal/securestore"
)

type acquisitionAddress struct{ service, key string }
type acquisitionMemoryStore struct {
	values        map[acquisitionAddress]string
	gets          []acquisitionAddress
	puts, deletes int
	failure       error
}

func (s *acquisitionMemoryStore) Put(service, key, value string) error {
	s.puts++
	s.values[acquisitionAddress{service, key}] = value
	return nil
}
func (s *acquisitionMemoryStore) Get(service, key string) (string, error) {
	address := acquisitionAddress{service, key}
	s.gets = append(s.gets, address)
	if s.failure != nil {
		return "", s.failure
	}
	value, ok := s.values[address]
	if !ok {
		return "", securestore.ErrNotFound
	}
	return value, nil
}
func (s *acquisitionMemoryStore) Delete(service, key string) error {
	s.deletes++
	delete(s.values, acquisitionAddress{service, key})
	return nil
}

// DefaultAuthority is a documented test seam, never a production reassignment.
// These tests are deliberately nonparallel and restore the original function
// without ever invoking it. No installed backend, record or refresh is reached.
func acquisitionFixture(t *testing.T) (*Plan, *acquisitionMemoryStore, *int) {
	t.Helper()
	plan, _, _, _ := composedHandleFixture(t)
	plan.bundle.ServerIdentity = "fixture/pm-server"
	plan.bundle.ClientIdentity = "fixture/pm-client"
	serverRecord, serverPin := composedHandleRecord(t, plan, "server", 100)
	clientRecord, clientPin := composedHandleRecord(t, plan, "client", 200)
	plan.bundle.Installation.ServerPins["pm"] = serverPin
	plan.bundle.ClientPin = clientPin
	store := &acquisitionMemoryStore{values: map[acquisitionAddress]string{}}
	authority, err := internalca.NewAuthority(store)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range []struct{ identity, value string }{{plan.bundle.ServerIdentity, string(serverRecord)}, {plan.bundle.ClientIdentity, string(clientRecord)}} {
		identity, err := internalca.ParseIdentity(entry.identity)
		if err != nil {
			t.Fatal(err)
		}
		if err := authority.Put(identity, TLSField, entry.value); err != nil {
			t.Fatal(err)
		}
	}
	if store.puts != 2 || store.deletes != 0 || len(store.gets) != 0 {
		t.Fatal("fixture setup escaped the normal Put path")
	}
	store.puts = 0
	original := internalca.DefaultAuthority
	calls := new(int)
	internalca.DefaultAuthority = func() (*internalca.Authority, error) { *calls++; return authority, nil }
	t.Cleanup(func() { internalca.DefaultAuthority = original })
	return plan, store, calls
}

func assertAcquisitionReadOnly(t *testing.T, plan *Plan, store *acquisitionMemoryStore, calls int, wantKeys []string) {
	t.Helper()
	if calls != 1 || store.puts != 0 || store.deletes != 0 {
		t.Fatalf("acquisition refreshed, wrote or deleted: defaults=%d puts=%d deletes=%d", calls, store.puts, store.deletes)
	}
	var expected []acquisitionAddress
	if len(wantKeys) != 0 {
		expected = make([]acquisitionAddress, len(wantKeys))
	}
	for i, key := range wantKeys {
		expected[i] = acquisitionAddress{"vrooli.credentials.v1", key}
	}
	if !reflect.DeepEqual(store.gets, expected) {
		t.Fatalf("acquisition used an unexpected identity/field or provider probe: got %v want %v", store.gets, expected)
	}
}

func TestProtectedAcquisitionUsesConcreteDefaultAndExactReadOnlyRecords(t *testing.T) {
	for _, operation := range []string{"tls", "handles"} {
		t.Run(operation, func(t *testing.T) {
			plan, store, calls := acquisitionFixture(t)
			if operation == "tls" {
				server, client, err := plan.AcquireTLS()
				if err != nil || server == nil || client == nil {
					t.Fatal("actual AcquireTLS refused disposable existing records", err)
				}
			} else {
				handles, err := plan.AcquireExistingHandles()
				if err != nil || handles == nil || handles.Publication == nil || handles.Broker == nil || handles.Profiles == nil {
					t.Fatal("actual AcquireExistingHandles refused disposable existing records", err)
				}
			}
			assertAcquisitionReadOnly(t, plan, store, *calls, []string{plan.bundle.ServerIdentity + ":" + TLSField, plan.bundle.ClientIdentity + ":" + TLSField})
		})
	}
}

func TestProtectedAcquisitionMissingOrUnavailableHasNoFallback(t *testing.T) {
	for _, operation := range []string{"tls", "handles"} {
		t.Run(operation, func(t *testing.T) {
			for _, mode := range []string{"missing-server", "missing-client", "unavailable", "absent", "default-unavailable"} {
				t.Run(mode, func(t *testing.T) {
					plan, store, calls := acquisitionFixture(t)
					serverKey, clientKey := plan.bundle.ServerIdentity+":"+TLSField, plan.bundle.ClientIdentity+":"+TLSField
					want := []string{serverKey}
					wantError := internalca.ErrUnconfigured
					switch mode {
					case "missing-server":
						delete(store.values, acquisitionAddress{"vrooli.credentials.v1", serverKey})
					case "missing-client":
						delete(store.values, acquisitionAddress{"vrooli.credentials.v1", clientKey})
						want = append(want, clientKey)
					case "unavailable":
						store.failure = securestore.ErrUnavailable
						wantError = internalca.ErrProviderUnavailable
					case "absent":
						store.failure = securestore.ErrAbsent
						wantError = internalca.ErrProviderAbsent
					case "default-unavailable":
						want = nil
						wantError = internalca.ErrProviderUnavailable
						internalca.DefaultAuthority = func() (*internalca.Authority, error) { *calls++; return nil, internalca.ErrProviderUnavailable }
					}
					// Repeat the same call: no refresh, alternate identity, field or backend
					// may convert the first exact refusal into a successful fallback.
					for attempt := 0; attempt < 2; attempt++ {
						*calls = 0
						store.gets = nil
						var err error
						if operation == "tls" {
							server, client, failure := plan.AcquireTLS()
							err = failure
							if server != nil || client != nil {
								t.Fatal("refusal returned partial TLS handles")
							}
						} else {
							handles, failure := plan.AcquireExistingHandles()
							err = failure
							if handles != nil {
								t.Fatal("refusal returned partial client handles")
							}
						}
						if !errors.Is(err, wantError) {
							t.Fatalf("failure taxonomy lost: got %v want %v", err, wantError)
						}
						assertAcquisitionReadOnly(t, plan, store, *calls, want)
					}
				})
			}
		})
	}
}
