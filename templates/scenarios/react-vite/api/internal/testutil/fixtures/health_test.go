package fixtures

import (
	"reflect"
	"testing"
	"time"
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
	want := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC).Format(time.RFC3339)
	if r.Timestamp != want {
		t.Errorf("Timestamp = %q, want %q", r.Timestamp, want)
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

// TestNewHealthResponse_WithTimestamp pins the Time→RFC3339 contract:
// callers pass a time.Time for ergonomics, but the fixture stores the
// wire form so JSON round-trips byte-identically with api-core/health.
func TestNewHealthResponse_WithTimestamp(t *testing.T) {
	target := time.Date(2030, 5, 1, 0, 0, 0, 0, time.UTC)
	r := NewHealthResponse(WithHealthTimestamp(target))
	want := target.Format(time.RFC3339)
	if r.Timestamp != want {
		t.Errorf("Timestamp = %q, want %q", r.Timestamp, want)
	}
}

func TestNewHealthResponse_WithDependency(t *testing.T) {
	r := NewHealthResponse(
		WithHealthDependency("database", DependencyStatus{Connected: true, Database: "react_vite"}),
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
// initializes the Dependencies map even when called twice with a fresh
// fixture (i.e., the second call doesn't panic on a nil map).
func TestNewHealthResponse_WithDependency_LazyMapInit(t *testing.T) {
	r := NewHealthResponse(
		WithHealthDependency("a", DependencyStatus{Connected: true}),
		WithHealthDependency("b", DependencyStatus{Connected: false}),
	)
	if len(r.Dependencies) != 2 {
		t.Fatalf("expected 2 dependencies, got %d (%v)", len(r.Dependencies), r.Dependencies)
	}
}

// TestNewHealthResponse_OptOrderIndependent guards against opt
// implementations that read other fields when applying — opts must be
// commutative for the functional-options pattern to remain readable at
// the callsite. reflect.DeepEqual is used because HealthResponse holds
// a map (not comparable with ==).
func TestNewHealthResponse_OptOrderIndependent(t *testing.T) {
	a := NewHealthResponse(
		WithHealthStatus("a"),
		WithHealthVersion("v1"),
		WithHealthDependency("database", DependencyStatus{Connected: true}),
	)
	b := NewHealthResponse(
		WithHealthDependency("database", DependencyStatus{Connected: true}),
		WithHealthVersion("v1"),
		WithHealthStatus("a"),
	)
	if !reflect.DeepEqual(a, b) {
		t.Errorf("opt order changed result:\n a=%+v\n b=%+v", a, b)
	}
}
