package service

import (
	"errors"
	"testing"
	"time"

	"tunnel-manager/domain"
)

// --- Validation tests (no store needed) ---

// [REQ:ROUTE-004] Route manifest validation - missing required fields
func TestRouteValidationMissingFields(t *testing.T) {
	ms := &mockRouteStore{}
	svc := NewRouteService(ms)

	tests := []struct {
		name  string
		input domain.RouteInput
	}{
		{"missing subdomain", domain.RouteInput{ScenarioName: "x", LocalPort: 3000}},
		{"missing scenario_name", domain.RouteInput{Subdomain: "x", LocalPort: 3000}},
		{"missing local_port", domain.RouteInput{Subdomain: "x", ScenarioName: "x"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.Create(tc.input)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	}
}

// [REQ:ROUTE-004] Route manifest validation - invalid port range
func TestRouteValidationInvalidPort(t *testing.T) {
	ms := &mockRouteStore{}
	svc := NewRouteService(ms)

	_, err := svc.Create(domain.RouteInput{
		Subdomain:    "test",
		ScenarioName: "test",
		LocalPort:    99999,
	})
	if err == nil {
		t.Fatal("expected error for port > 65535")
	}
}

// [REQ:ROUTE-004] Route manifest validation - default health path
func TestRouteValidationDefaultHealthPath(t *testing.T) {
	ms := &mockRouteStore{
		createFn: func(subdomain, scenarioName string, localPort int, healthPath, publicURL string, enabled bool) (*domain.Route, error) {
			return &domain.Route{
				ID:           1,
				Subdomain:    subdomain,
				ScenarioName: scenarioName,
				LocalPort:    localPort,
				HealthPath:   healthPath,
				PublicURL:    publicURL,
				Enabled:      enabled,
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
			}, nil
		},
	}
	svc := NewRouteService(ms)

	route, err := svc.Create(domain.RouteInput{
		Subdomain:    "test-defaults",
		ScenarioName: "test",
		LocalPort:    3000,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if route.HealthPath != "/health" {
		t.Errorf("health_path = %q, want %q", route.HealthPath, "/health")
	}
	if !route.Enabled {
		t.Error("enabled should default to true")
	}
}

// [REQ:ROUTE-004] Validation edge cases

func TestRouteValidation_NegativePort(t *testing.T) {
	ms := &mockRouteStore{}
	svc := NewRouteService(ms)

	_, err := svc.Create(domain.RouteInput{
		Subdomain:    "neg-port",
		ScenarioName: "test",
		LocalPort:    -1,
	})
	if err == nil {
		t.Error("expected error for negative port")
	}
}

func TestRouteValidation_MaxPort(t *testing.T) {
	ms := &mockRouteStore{
		createFn: func(subdomain, scenarioName string, localPort int, healthPath, publicURL string, enabled bool) (*domain.Route, error) {
			return &domain.Route{
				ID:        1,
				Subdomain: subdomain,
				LocalPort: localPort,
				Enabled:   enabled,
			}, nil
		},
	}
	svc := NewRouteService(ms)

	route, err := svc.Create(domain.RouteInput{
		Subdomain:    "max-port",
		ScenarioName: "test",
		LocalPort:    65535,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if route.LocalPort != 65535 {
		t.Errorf("local_port = %d, want 65535", route.LocalPort)
	}
}

func TestRouteValidation_MinPort(t *testing.T) {
	ms := &mockRouteStore{
		createFn: func(subdomain, scenarioName string, localPort int, healthPath, publicURL string, enabled bool) (*domain.Route, error) {
			return &domain.Route{
				ID:        1,
				Subdomain: subdomain,
				LocalPort: localPort,
				Enabled:   enabled,
			}, nil
		},
	}
	svc := NewRouteService(ms)

	route, err := svc.Create(domain.RouteInput{
		Subdomain:    "min-port",
		ScenarioName: "test",
		LocalPort:    1,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if route.LocalPort != 1 {
		t.Errorf("local_port = %d, want 1", route.LocalPort)
	}
}

func TestRouteValidation_UpdateInvalidPort(t *testing.T) {
	ms := &mockRouteStore{
		getByIDFn: func(id int) (*domain.Route, error) {
			return &domain.Route{ID: id, Subdomain: "update-bad-port", ScenarioName: "test", LocalPort: 3000, HealthPath: "/health", Enabled: true}, nil
		},
	}
	svc := NewRouteService(ms)

	_, err := svc.Update(1, domain.RouteInput{LocalPort: 99999})
	if err == nil {
		t.Error("expected error for port > 65535 on update")
	}
}

// --- Service edge case tests ---

// [REQ:ROUTE-002] RouteService service-level edge case tests

func TestRouteService_GetByID_NotFound(t *testing.T) {
	ms := &mockRouteStore{
		getByIDFn: func(id int) (*domain.Route, error) {
			return nil, nil
		},
	}
	svc := NewRouteService(ms)

	got, err := svc.GetByID(99999)
	if got != nil {
		t.Error("expected nil for non-existent route")
	}
	var de *domain.DomainError
	if !errors.As(err, &de) || de.Code != domain.ErrCodeNotFound {
		t.Fatalf("expected not_found domain error, got: %v", err)
	}
}

func TestRouteService_Delete_NotFound(t *testing.T) {
	ms := &mockRouteStore{
		deleteFn: func(id int) error {
			return domain.ErrNotFound("route not found")
		},
	}
	svc := NewRouteService(ms)

	err := svc.Delete(99999)
	if err == nil {
		t.Error("expected error deleting non-existent route")
	}
}

func TestRouteService_Update_NotFound(t *testing.T) {
	ms := &mockRouteStore{
		getByIDFn: func(id int) (*domain.Route, error) {
			return nil, nil
		},
	}
	svc := NewRouteService(ms)

	got, err := svc.Update(99999, domain.RouteInput{LocalPort: 3000})
	if got != nil {
		t.Error("expected nil for non-existent route update")
	}
	var de *domain.DomainError
	if !errors.As(err, &de) || de.Code != domain.ErrCodeNotFound {
		t.Fatalf("expected not_found domain error, got: %v", err)
	}
}

func TestRouteService_List_EmptyDB(t *testing.T) {
	ms := &mockRouteStore{
		listFn: func() ([]domain.Route, error) {
			return []domain.Route{}, nil
		},
	}
	svc := NewRouteService(ms)

	routes, err := svc.List()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(routes) != 0 {
		t.Errorf("expected 0 routes, got %d", len(routes))
	}
}

func TestRouteService_List_OrderedBySubdomain(t *testing.T) {
	ms := &mockRouteStore{
		listFn: func() ([]domain.Route, error) {
			// Store returns pre-ordered results (as the real store does ORDER BY subdomain)
			return []domain.Route{
				{ID: 2, Subdomain: "a-app"},
				{ID: 3, Subdomain: "m-app"},
				{ID: 1, Subdomain: "z-app"},
			}, nil
		},
	}
	svc := NewRouteService(ms)

	routes, err := svc.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(routes) != 3 {
		t.Fatalf("expected 3 routes, got %d", len(routes))
	}
	if routes[0].Subdomain != "a-app" {
		t.Errorf("first route subdomain = %q, want a-app", routes[0].Subdomain)
	}
	if routes[1].Subdomain != "m-app" {
		t.Errorf("second route subdomain = %q, want m-app", routes[1].Subdomain)
	}
	if routes[2].Subdomain != "z-app" {
		t.Errorf("third route subdomain = %q, want z-app", routes[2].Subdomain)
	}
}

func TestRouteService_Create_TimestampsSet(t *testing.T) {
	now := time.Now()
	ms := &mockRouteStore{
		createFn: func(subdomain, scenarioName string, localPort int, healthPath, publicURL string, enabled bool) (*domain.Route, error) {
			return &domain.Route{
				ID:           1,
				Subdomain:    subdomain,
				ScenarioName: scenarioName,
				LocalPort:    localPort,
				HealthPath:   healthPath,
				Enabled:      enabled,
				CreatedAt:    now,
				UpdatedAt:    now,
			}, nil
		},
	}
	svc := NewRouteService(ms)

	route, err := svc.Create(domain.RouteInput{
		Subdomain:    "ts-test",
		ScenarioName: "test",
		LocalPort:    3000,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if route.CreatedAt.IsZero() {
		t.Error("created_at should be set")
	}
	if route.UpdatedAt.IsZero() {
		t.Error("updated_at should be set")
	}
}

func TestRouteService_Update_PreservesUnchangedFields(t *testing.T) {
	ms := &mockRouteStore{
		getByIDFn: func(id int) (*domain.Route, error) {
			return &domain.Route{
				ID:           id,
				Subdomain:    "preserve-test",
				ScenarioName: "original-scenario",
				LocalPort:    3000,
				HealthPath:   "/health",
				Enabled:      true,
			}, nil
		},
		updateFn: func(id int, subdomain, scenarioName string, localPort int, healthPath, publicURL string, enabled bool) (*domain.Route, error) {
			return &domain.Route{
				ID:           id,
				Subdomain:    subdomain,
				ScenarioName: scenarioName,
				LocalPort:    localPort,
				HealthPath:   healthPath,
				PublicURL:    publicURL,
				Enabled:      enabled,
				UpdatedAt:    time.Now(),
			}, nil
		},
	}
	svc := NewRouteService(ms)

	updated, err := svc.Update(1, domain.RouteInput{LocalPort: 4000})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.Subdomain != "preserve-test" {
		t.Errorf("subdomain changed: %q", updated.Subdomain)
	}
	if updated.ScenarioName != "original-scenario" {
		t.Errorf("scenario_name changed: %q", updated.ScenarioName)
	}
	if updated.LocalPort != 4000 {
		t.Errorf("local_port = %d, want 4000", updated.LocalPort)
	}
}
