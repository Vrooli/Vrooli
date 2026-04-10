package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"tunnel-manager/domain"

	"github.com/gorilla/mux"
)

// mockRouteManager implements RouteManager for handler tests.
type mockRouteManager struct {
	listFn    func() ([]domain.Route, error)
	getByIDFn func(int) (*domain.Route, error)
	createFn  func(domain.RouteInput) (*domain.Route, error)
	updateFn  func(int, domain.RouteInput) (*domain.Route, error)
	deleteFn  func(int) error
}

func (m *mockRouteManager) List() ([]domain.Route, error)                      { return m.listFn() }
func (m *mockRouteManager) GetByID(id int) (*domain.Route, error)              { return m.getByIDFn(id) }
func (m *mockRouteManager) Create(in domain.RouteInput) (*domain.Route, error) { return m.createFn(in) }

func (m *mockRouteManager) Update(id int, in domain.RouteInput) (*domain.Route, error) {
	return m.updateFn(id, in)
}

func (m *mockRouteManager) Delete(id int) error { return m.deleteFn(id) }

// --- Handler edge case tests ---

// [REQ:ROUTE-002] Route handler edge cases

func TestHandlerGetRoute_NotFound(t *testing.T) {
	svc := &mockRouteManager{
		getByIDFn: func(id int) (*domain.Route, error) {
			return nil, domain.ErrNotFound("route not found")
		},
	}

	h := HandleGetRoute(svc)
	req := httptest.NewRequest("GET", "/api/v1/routes/99999", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "99999"})
	w := httptest.NewRecorder()
	h(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestHandlerGetRoute_InvalidID(t *testing.T) {
	svc := &mockRouteManager{}

	h := HandleGetRoute(svc)
	req := httptest.NewRequest("GET", "/api/v1/routes/abc", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "abc"})
	w := httptest.NewRecorder()
	h(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandlerCreateRoute_InvalidJSON(t *testing.T) {
	svc := &mockRouteManager{}

	h := HandleCreateRoute(svc)
	req := httptest.NewRequest("POST", "/api/v1/routes", bytes.NewBufferString("{invalid"))
	w := httptest.NewRecorder()
	h(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandlerCreateRoute_MissingRequiredFields(t *testing.T) {
	svc := &mockRouteManager{
		createFn: func(in domain.RouteInput) (*domain.Route, error) {
			return nil, domain.ErrValidation("scenario_name is required")
		},
	}

	h := HandleCreateRoute(svc)
	req := httptest.NewRequest("POST", "/api/v1/routes", bytes.NewBufferString(`{"subdomain":"test"}`))
	w := httptest.NewRecorder()
	h(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandlerUpdateRoute_InvalidID(t *testing.T) {
	svc := &mockRouteManager{}

	h := HandleUpdateRoute(svc)
	req := httptest.NewRequest("PUT", "/api/v1/routes/abc", bytes.NewBufferString(`{"local_port":3500}`))
	req = mux.SetURLVars(req, map[string]string{"id": "abc"})
	w := httptest.NewRecorder()
	h(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandlerUpdateRoute_NotFound(t *testing.T) {
	svc := &mockRouteManager{
		updateFn: func(id int, in domain.RouteInput) (*domain.Route, error) {
			return nil, domain.ErrNotFound("route not found")
		},
	}

	h := HandleUpdateRoute(svc)
	req := httptest.NewRequest("PUT", "/api/v1/routes/99999", bytes.NewBufferString(`{"local_port":3500}`))
	req = mux.SetURLVars(req, map[string]string{"id": "99999"})
	w := httptest.NewRecorder()
	h(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestHandlerUpdateRoute_InvalidJSON(t *testing.T) {
	svc := &mockRouteManager{}

	h := HandleUpdateRoute(svc)
	req := httptest.NewRequest("PUT", "/api/v1/routes/1", bytes.NewBufferString("{bad"))
	req = mux.SetURLVars(req, map[string]string{"id": "1"})
	w := httptest.NewRecorder()
	h(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandlerDeleteRoute_InvalidID(t *testing.T) {
	svc := &mockRouteManager{}

	h := HandleDeleteRoute(svc)
	req := httptest.NewRequest("DELETE", "/api/v1/routes/abc", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "abc"})
	w := httptest.NewRecorder()
	h(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandlerDeleteRoute_NotFound(t *testing.T) {
	svc := &mockRouteManager{
		deleteFn: func(id int) error {
			return domain.ErrNotFound("route not found")
		},
	}

	h := HandleDeleteRoute(svc)
	req := httptest.NewRequest("DELETE", "/api/v1/routes/99999", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "99999"})
	w := httptest.NewRecorder()
	h(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestHandlerListRoutes_EmptyDB(t *testing.T) {
	svc := &mockRouteManager{
		listFn: func() ([]domain.Route, error) {
			return []domain.Route{}, nil
		},
	}

	h := HandleListRoutes(svc)
	req := httptest.NewRequest("GET", "/api/v1/routes", nil)
	w := httptest.NewRecorder()
	h(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var routes []domain.Route
	if err := json.Unmarshal(w.Body.Bytes(), &routes); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(routes) != 0 {
		t.Errorf("expected empty list, got %d routes", len(routes))
	}
}

func TestHandlerCreateRoute_DuplicateSubdomain(t *testing.T) {
	svc := &mockRouteManager{
		createFn: func(in domain.RouteInput) (*domain.Route, error) {
			return nil, domain.ErrConflict("subdomain already exists")
		},
	}

	h := HandleCreateRoute(svc)
	body := `{"subdomain":"dup-test","scenario_name":"scenario-b","local_port":3001,"public_url":"https://dup-test.example.com"}`
	req := httptest.NewRequest("POST", "/api/v1/routes", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	h(w, req)

	if w.Code == http.StatusCreated {
		t.Error("expected error for duplicate subdomain, got 201")
	}
}

func TestHandlerCreateRoute_CustomHealthPath(t *testing.T) {
	svc := &mockRouteManager{
		createFn: func(in domain.RouteInput) (*domain.Route, error) {
			return &domain.Route{
				ID:         1,
				Subdomain:  in.Subdomain,
				LocalPort:  in.LocalPort,
				HealthPath: in.HealthPath,
				PublicURL:  in.PublicURL,
				Enabled:    true,
				CreatedAt:  time.Now(),
				UpdatedAt:  time.Now(),
			}, nil
		},
	}

	h := HandleCreateRoute(svc)
	body := `{"subdomain":"custom-hp","scenario_name":"test","local_port":3000,"health_path":"/api/health","public_url":"https://custom.example.com"}`
	req := httptest.NewRequest("POST", "/api/v1/routes", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	h(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var route domain.Route
	if err := json.Unmarshal(w.Body.Bytes(), &route); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if route.HealthPath != "/api/health" {
		t.Errorf("health_path = %q, want %q", route.HealthPath, "/api/health")
	}
}

func TestHandlerUpdateRoute_MultipleFields(t *testing.T) {
	svc := &mockRouteManager{
		updateFn: func(id int, in domain.RouteInput) (*domain.Route, error) {
			return &domain.Route{
				ID:         id,
				Subdomain:  "multi-update",
				LocalPort:  in.LocalPort,
				HealthPath: in.HealthPath,
				PublicURL:  in.PublicURL,
				Enabled:    true,
				UpdatedAt:  time.Now(),
			}, nil
		},
	}

	h := HandleUpdateRoute(svc)
	body := `{"local_port":4000,"health_path":"/healthz","public_url":"https://updated.example.com"}`
	req := httptest.NewRequest("PUT", "/api/v1/routes/1", bytes.NewBufferString(body))
	req = mux.SetURLVars(req, map[string]string{"id": "1"})
	w := httptest.NewRecorder()
	h(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var updated domain.Route
	if err := json.Unmarshal(w.Body.Bytes(), &updated); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if updated.LocalPort != 4000 {
		t.Errorf("local_port = %d, want 4000", updated.LocalPort)
	}
	if updated.HealthPath != "/healthz" {
		t.Errorf("health_path = %q, want /healthz", updated.HealthPath)
	}
	if updated.PublicURL != "https://updated.example.com" {
		t.Errorf("public_url = %q, want https://updated.example.com", updated.PublicURL)
	}
	// Unchanged fields should be preserved
	if updated.Subdomain != "multi-update" {
		t.Errorf("subdomain = %q, want multi-update (should be preserved)", updated.Subdomain)
	}
}

func TestHandlerCreateRoute_DisabledRoute(t *testing.T) {
	svc := &mockRouteManager{
		createFn: func(in domain.RouteInput) (*domain.Route, error) {
			enabled := true
			if in.Enabled != nil {
				enabled = *in.Enabled
			}
			return &domain.Route{
				ID:        1,
				Subdomain: in.Subdomain,
				LocalPort: in.LocalPort,
				PublicURL: in.PublicURL,
				Enabled:   enabled,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}, nil
		},
	}

	h := HandleCreateRoute(svc)
	body := `{"subdomain":"disabled-app","scenario_name":"test","local_port":3000,"enabled":false,"public_url":"https://disabled.example.com"}`
	req := httptest.NewRequest("POST", "/api/v1/routes", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	h(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var route domain.Route
	if err := json.Unmarshal(w.Body.Bytes(), &route); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if route.Enabled {
		t.Error("expected enabled=false")
	}
}

// --- Tests from route_crud_test.go ---

// [REQ:ROUTE-002] Route manifest CRUD via API - create route
func TestHandlerCreateRoute(t *testing.T) {
	svc := &mockRouteManager{
		createFn: func(in domain.RouteInput) (*domain.Route, error) {
			return &domain.Route{
				ID:           1,
				Subdomain:    in.Subdomain,
				ScenarioName: in.ScenarioName,
				LocalPort:    in.LocalPort,
				HealthPath:   "/health",
				PublicURL:    in.PublicURL,
				Enabled:      true,
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
			}, nil
		},
	}

	h := HandleCreateRoute(svc)
	body := `{"subdomain":"agent-manager","scenario_name":"agent-manager","local_port":35001,"public_url":"https://agent-manager.example.com"}`
	req := httptest.NewRequest("POST", "/api/v1/routes", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	h(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var route domain.Route
	if err := json.Unmarshal(w.Body.Bytes(), &route); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if route.Subdomain != "agent-manager" {
		t.Errorf("subdomain = %q, want %q", route.Subdomain, "agent-manager")
	}
	if route.LocalPort != 35001 {
		t.Errorf("local_port = %d, want %d", route.LocalPort, 35001)
	}
}

// [REQ:ROUTE-002] Route manifest CRUD via API - list routes
func TestHandlerListRoutes(t *testing.T) {
	svc := &mockRouteManager{
		listFn: func() ([]domain.Route, error) {
			return []domain.Route{
				{ID: 1, Subdomain: "app-a", ScenarioName: "scenario-a", LocalPort: 3000},
				{ID: 2, Subdomain: "app-b", ScenarioName: "scenario-b", LocalPort: 3001},
			}, nil
		},
	}

	h := HandleListRoutes(svc)
	req := httptest.NewRequest("GET", "/api/v1/routes", nil)
	w := httptest.NewRecorder()
	h(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var routes []domain.Route
	if err := json.Unmarshal(w.Body.Bytes(), &routes); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(routes) != 2 {
		t.Errorf("expected 2 routes, got %d", len(routes))
	}
}
