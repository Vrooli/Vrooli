package securestore

import (
	"errors"
	"fmt"
	"testing"
)

type probeStore struct {
	values      map[string]string
	failPut     bool
	failGet     bool
	failDelete  bool
	putCalls    int
	deleteCalls int
}

func (s *probeStore) Put(service, key, value string) error {
	s.putCalls++
	if s.failPut {
		return fmt.Errorf("%w: put unavailable", ErrUnavailable)
	}
	if s.values == nil {
		s.values = map[string]string{}
	}
	s.values[service+"/"+key] = value
	return nil
}

func (s *probeStore) Get(service, key string) (string, error) {
	if s.failGet {
		return "", fmt.Errorf("%w: get unavailable", ErrUnavailable)
	}
	value, ok := s.values[service+"/"+key]
	if !ok {
		return "", fmt.Errorf("%w: %s/%s", ErrNotFound, service, key)
	}
	return value, nil
}

func (s *probeStore) Delete(service, key string) error {
	s.deleteCalls++
	if s.failDelete {
		return fmt.Errorf("%w: delete unavailable", ErrUnavailable)
	}
	delete(s.values, service+"/"+key)
	return nil
}

// TestProbeReadsAndNeverWrites is the reason Probe exists in this shape:
// resolving environment for a scenario must not require write access to the
// operator keyring, and a clean not-found is proof the backend answered.
func TestProbeReadsAndNeverWrites(t *testing.T) {
	store := &probeStore{}
	if err := Probe(store); err != nil {
		t.Fatalf("Probe() error = %v, want nil for a reachable empty store", err)
	}
	if store.putCalls != 0 || store.deleteCalls != 0 {
		t.Fatalf("Probe() wrote to the store: %d puts, %d deletes", store.putCalls, store.deleteCalls)
	}
}

func TestProbeClassifiesHostConditions(t *testing.T) {
	for _, testCase := range []struct {
		name  string
		store Store
		want  error
	}{
		{"nil store has no adapter", nil, ErrAbsent},
		{"absent adapter", Absent("no backend on this platform"), ErrAbsent},
		{"unreachable adapter", Unavailable("session unreachable"), ErrUnavailable},
		{"failing read", &probeStore{failGet: true}, ErrUnavailable},
		{"unclassified read failure", failingStore{err: errors.New("raw adapter error")}, ErrUnavailable},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			err := Probe(testCase.store)
			if !errors.Is(err, testCase.want) {
				t.Fatalf("Probe() error = %v, want %v", err, testCase.want)
			}
		})
	}
}

type failingStore struct{ err error }

func (s failingStore) Put(string, string, string) error   { return s.err }
func (s failingStore) Get(string, string) (string, error) { return "", s.err }
func (s failingStore) Delete(string, string) error        { return s.err }

func TestProbeWritableVerifiesStoreReadbackAndDeletion(t *testing.T) {
	store := &probeStore{}
	if err := ProbeWritable(store); err != nil {
		t.Fatalf("ProbeWritable() error = %v", err)
	}
	if len(store.values) != 0 || store.deleteCalls == 0 {
		t.Fatalf("ProbeWritable() did not remove its value: %+v", store)
	}
}

func TestProbeWritableFailsClosedWhenStoreIsUnavailable(t *testing.T) {
	for _, store := range []Store{nil, &probeStore{failPut: true}, &probeStore{failGet: true}, &probeStore{failDelete: true}} {
		err := ProbeWritable(store)
		if !errors.Is(err, ErrUnavailable) && !errors.Is(err, ErrAbsent) {
			t.Fatalf("ProbeWritable(%T) error = %v, want a provider condition", store, err)
		}
	}
}

func TestAdapterNameReportsBackendWithoutSecrets(t *testing.T) {
	for _, testCase := range []struct {
		store Store
		want  string
	}{
		{nil, "none"},
		{Absent("no adapter"), "none"},
		{Unavailable("unreachable"), "unreachable"},
		{&probeStore{}, "unknown"},
	} {
		if got := AdapterName(testCase.store); got != testCase.want {
			t.Fatalf("AdapterName(%T) = %q, want %q", testCase.store, got, testCase.want)
		}
	}
}
