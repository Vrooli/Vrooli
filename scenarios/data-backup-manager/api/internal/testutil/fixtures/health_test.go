package fixtures

import (
	"testing"

	healthv1 "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/health"
	"google.golang.org/protobuf/proto"
)

func TestNewHealthResponse_Defaults(t *testing.T) {
	r := NewHealthResponse()
	if r.Status != "healthy" {
		t.Errorf("Status = %q, want healthy", r.Status)
	}
	if r.Service != "react-vite-test" {
		t.Errorf("Service = %q, want react-vite-test", r.Service)
	}
	if r.Version != "1.0.0" {
		t.Errorf("Version = %q, want 1.0.0", r.Version)
	}
	if !r.Readiness {
		t.Errorf("Readiness = false, want true")
	}
	if r.Timestamp != "2026-01-01T00:00:00Z" {
		t.Errorf("Timestamp = %q, want 2026-01-01T00:00:00Z", r.Timestamp)
	}
	if r.Dependencies != nil {
		t.Errorf("Dependencies should default to nil; got %v", r.Dependencies)
	}
}

func TestNewHealthResponse_WithStatus(t *testing.T) {
	r := NewHealthResponse(WithHealthStatus("degraded"))
	if r.Status != "degraded" {
		t.Errorf("Status = %q, want degraded", r.Status)
	}
}

func TestNewHealthResponse_WithService(t *testing.T) {
	r := NewHealthResponse(WithHealthService("override"))
	if r.Service != "override" {
		t.Errorf("Service = %q, want override", r.Service)
	}
}

func TestNewHealthResponse_WithVersion(t *testing.T) {
	r := NewHealthResponse(WithHealthVersion("9.9.9"))
	if r.Version != "9.9.9" {
		t.Errorf("Version = %q, want 9.9.9", r.Version)
	}
}

func TestNewHealthResponse_WithReadiness(t *testing.T) {
	r := NewHealthResponse(WithHealthReadiness(false))
	if r.Readiness {
		t.Errorf("Readiness = true, want false")
	}
}

func TestNewHealthResponse_WithTimestamp(t *testing.T) {
	r := NewHealthResponse(WithHealthTimestamp("2030-05-01T00:00:00Z"))
	if r.Timestamp != "2030-05-01T00:00:00Z" {
		t.Errorf("Timestamp = %q, want 2030-05-01T00:00:00Z", r.Timestamp)
	}
}

func TestNewHealthResponse_WithDependency(t *testing.T) {
	r := NewHealthResponse(
		WithHealthDependency("database", &healthv1.DependencyStatus{Connected: true, Database: "react_vite"}),
	)
	got, ok := r.Dependencies["database"]
	if !ok {
		t.Fatalf("Dependencies missing 'database' entry; got %v", r.Dependencies)
	}
	if !got.Connected {
		t.Errorf("dependency.Connected = false, want true")
	}
	if got.Database != "react_vite" {
		t.Errorf("dependency.Database = %q, want react_vite", got.Database)
	}
}

// TestNewHealthResponse_WithDependency_LazyMapInit proves the opt
// initialises the Dependencies map on first call so callers don't have
// to construct it ahead of time.
func TestNewHealthResponse_WithDependency_LazyMapInit(t *testing.T) {
	r := NewHealthResponse(
		WithHealthDependency("a", &healthv1.DependencyStatus{Connected: true}),
		WithHealthDependency("b", &healthv1.DependencyStatus{Connected: false}),
	)
	if len(r.Dependencies) != 2 {
		t.Fatalf("expected 2 dependencies, got %d (%v)", len(r.Dependencies), r.Dependencies)
	}
}

// TestNewHealthResponse_OptOrderIndependent guards against opt
// implementations that read other fields when applying — opts must be
// commutative for the functional-options pattern to remain readable at
// the call site. Uses proto.Equal because proto messages carry internal
// state that == doesn't compare correctly.
func TestNewHealthResponse_OptOrderIndependent(t *testing.T) {
	a := NewHealthResponse(
		WithHealthStatus("a"),
		WithHealthVersion("v1"),
		WithHealthDependency("database", &healthv1.DependencyStatus{Connected: true}),
	)
	b := NewHealthResponse(
		WithHealthDependency("database", &healthv1.DependencyStatus{Connected: true}),
		WithHealthVersion("v1"),
		WithHealthStatus("a"),
	)
	if !proto.Equal(a, b) {
		t.Errorf("opt order changed result:\n a=%+v\n b=%+v", a, b)
	}
}

// TestHealthResponse_TypeAlias confirms `fixtures.HealthResponse` is
// the generated proto type, not a hand-rolled mirror. Compile-time
// guarantee that we never silently drift back to a parallel struct.
func TestHealthResponse_TypeAlias(t *testing.T) {
	var _ HealthResponse = healthv1.Response{}
	var _ DependencyStatus = healthv1.DependencyStatus{}
}
