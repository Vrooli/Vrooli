package securestore

import (
	"errors"
	"testing"
)

type probeStore struct {
	values      map[string]string
	failPut     bool
	failGet     bool
	failDelete  bool
	deleteCalls int
}

func (s *probeStore) Put(service, key, value string) error {
	if s.failPut {
		return errors.New("put unavailable")
	}
	if s.values == nil {
		s.values = map[string]string{}
	}
	s.values[service+"/"+key] = value
	return nil
}

func (s *probeStore) Get(service, key string) (string, error) {
	if s.failGet {
		return "", errors.New("get unavailable")
	}
	return s.values[service+"/"+key], nil
}

func (s *probeStore) Delete(service, key string) error {
	s.deleteCalls++
	if s.failDelete {
		return errors.New("delete unavailable")
	}
	delete(s.values, service+"/"+key)
	return nil
}

func TestProbeVerifiesStoreReadbackAndDeletion(t *testing.T) {
	store := &probeStore{}
	if err := Probe(store); err != nil {
		t.Fatalf("Probe() error = %v", err)
	}
	if len(store.values) != 0 || store.deleteCalls == 0 {
		t.Fatalf("Probe() did not remove its value: %+v", store)
	}
}

func TestProbeFailsClosedWhenStoreIsUnavailable(t *testing.T) {
	for _, store := range []Store{nil, &probeStore{failPut: true}, &probeStore{failGet: true}, &probeStore{failDelete: true}} {
		if err := Probe(store); !errors.Is(err, ErrUnavailable) {
			t.Fatalf("Probe(%T) error = %v, want ErrUnavailable", store, err)
		}
	}
}
