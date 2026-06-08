package discovery

import (
	"errors"
	"testing"

	"connectrpc.com/connect"
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
