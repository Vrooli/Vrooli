package discovery

import (
	"errors"
	"testing"

	"connectrpc.com/connect"

	"github.com/ecosystem-manager/api/pkg/tasks"
)

func TestToConnectError(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want connect.Code
	}{
		{"empty name", ErrEmptyName, connect.CodeInvalidArgument},
		{"resource not found", ErrResourceNotFound, connect.CodeNotFound},
		{"scenario not found", ErrScenarioNotFound, connect.CodeNotFound},
		{"discovery unavailable", ErrDiscoveryUnavailable, connect.CodeUnavailable},
		{"generic", errors.New("boom"), connect.CodeInternal},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ToConnectError(tc.err)
			if connect.CodeOf(got) != tc.want {
				t.Fatalf("ToConnectError(%v) = %v, want %v", tc.err, connect.CodeOf(got), tc.want)
			}
		})
	}
	if ToConnectError(nil) != nil {
		t.Fatal("ToConnectError(nil) should be nil")
	}
}

func TestServiceResourceEmptyName(t *testing.T) {
	s := NewService(nil)
	if _, _, err := s.Resource("", false); !errors.Is(err, ErrEmptyName) {
		t.Fatalf("Resource(\"\") err = %v, want ErrEmptyName", err)
	}
}

func TestServiceScenarioEmptyName(t *testing.T) {
	s := NewService(nil)
	if _, _, err := s.Scenario("", false); !errors.Is(err, ErrEmptyName) {
		t.Fatalf("Scenario(\"\") err = %v, want ErrEmptyName", err)
	}
}

func TestServiceResourcesDegradesToLastGoodOnFailure(t *testing.T) {
	s := NewService(nil)
	good := []tasks.ResourceInfo{{Name: "redis"}, {Name: "vault"}}
	s.discoverResources = func() ([]tasks.ResourceInfo, error) { return good, nil }

	// First sweep succeeds and populates the cache.
	got, fromCache, err := s.Resources(true)
	if err != nil || fromCache || len(got) != 2 {
		t.Fatalf("first sweep: got %d resources, fromCache=%v, err=%v", len(got), fromCache, err)
	}

	// Next sweep fails: must serve last-good (stale) data, flagged with the
	// sentinel — never an empty list, and the cache must not be poisoned.
	s.discoverResources = func() ([]tasks.ResourceInfo, error) { return nil, errors.New("cli boom") }
	got, fromCache, err = s.Resources(true)
	if !errors.Is(err, ErrDiscoveryUnavailable) {
		t.Fatalf("want ErrDiscoveryUnavailable, got %v", err)
	}
	if !fromCache || len(got) != 2 {
		t.Fatalf("expected 2 last-good resources from cache, got %d (fromCache=%v)", len(got), fromCache)
	}
}

func TestServiceResourcesSurfacesErrorWhenNoCache(t *testing.T) {
	s := NewService(nil)
	s.discoverResources = func() ([]tasks.ResourceInfo, error) { return nil, errors.New("cli boom") }

	got, fromCache, err := s.Resources(true)
	if !errors.Is(err, ErrDiscoveryUnavailable) {
		t.Fatalf("want ErrDiscoveryUnavailable, got %v", err)
	}
	if len(got) != 0 || fromCache {
		t.Fatalf("expected no data and no cache hit, got %d (fromCache=%v)", len(got), fromCache)
	}
	// The failure must not have been cached as an empty success.
	if _, ok := s.cache.lastResources(); ok {
		t.Fatal("a failed sweep must not poison the cache")
	}
}

func TestServiceCategoriesNonEmpty(t *testing.T) {
	s := NewService(nil)
	if len(s.ResourceCategories()) == 0 || len(s.ScenarioCategories()) == 0 {
		t.Fatal("category maps should be non-empty")
	}
}

func TestServiceOperationsNilAssembler(t *testing.T) {
	s := NewService(nil)
	if ops := s.Operations(); ops != nil {
		t.Fatalf("Operations() with nil assembler = %v, want nil", ops)
	}
}
