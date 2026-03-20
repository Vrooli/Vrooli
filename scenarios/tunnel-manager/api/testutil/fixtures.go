package testutil

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"tunnel-manager/domain"
	"tunnel-manager/service"
	"tunnel-manager/store"
)

func Itoa(i int) string {
	return fmt.Sprintf("%d", i)
}

func WriteServiceJSON(t *testing.T, dir string, uiPort int) {
	t.Helper()
	svc := map[string]any{
		"ports": map[string]any{
			"ui": map[string]any{
				"port":    uiPort,
				"env_var": "UI_PORT",
			},
		},
	}
	data, _ := json.Marshal(svc)
	if err := os.WriteFile(filepath.Join(dir, "service.json"), data, 0o644); err != nil {
		t.Fatalf("write service.json: %v", err)
	}
}

// SeedTestRoute inserts a route for testing and returns it.
func SeedTestRoute(t *testing.T, db *sql.DB, subdomain, scenario string, port int) domain.Route {
	t.Helper()
	routeStore := store.NewRouteStore(db)
	svc := service.NewRouteService(routeStore)
	route, err := svc.Create(domain.RouteInput{
		Subdomain:    subdomain,
		ScenarioName: scenario,
		LocalPort:    port,
		PublicURL:    fmt.Sprintf("https://%s.example.com", subdomain),
	})
	if err != nil {
		t.Fatalf("SeedTestRoute: %v", err)
	}
	return *route
}
