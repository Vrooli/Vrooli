package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEveryDeclaredCLICommandHasAnEndpoint(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", ".vrooli", "endpoints.json"))
	if err != nil {
		t.Fatal(err)
	}
	var contract struct {
		CLICommands []struct {
			Name       string `json:"name"`
			EndpointID string `json:"endpoint_id"`
		} `json:"cli_commands"`
		Endpoints []struct {
			ID string `json:"id"`
		} `json:"endpoints"`
	}
	if err := json.Unmarshal(data, &contract); err != nil {
		t.Fatal(err)
	}
	ids := map[string]bool{}
	for _, endpoint := range contract.Endpoints {
		ids[endpoint.ID] = true
	}
	for _, command := range contract.CLICommands {
		if command.EndpointID == "" || !ids[command.EndpointID] {
			t.Errorf("%s has no registered endpoint", command.Name)
		}
	}
}

func TestEveryCliSurfaceRouteHasACommand(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "api", "testdata", "route-surface.json"))
	if err != nil {
		t.Fatal(err)
	}
	var contract struct {
		Routes []struct {
			Method   string   `json:"method"`
			Path     string   `json:"path"`
			Surfaces []string `json:"surfaces"`
		} `json:"routes"`
	}
	if err := json.Unmarshal(data, &contract); err != nil {
		t.Fatal(err)
	}
	files := []string{"app.go"}
	_ = filepath.Walk("domains", func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info != nil && !info.IsDir() && strings.HasSuffix(path, ".go") {
			files = append(files, path)
		}
		return nil
	})
	joined := strings.Builder{}
	for _, file := range files {
		body, readErr := os.ReadFile(file)
		if readErr != nil {
			t.Fatal(readErr)
		}
		joined.Write(body)
	}
	source := joined.String()
	for _, route := range contract.Routes {
		if !contains(route.Surfaces, "cli") {
			continue
		}
		if route.Path == "/health" {
			continue
		} // cli-core owns the status probe.
		path := strings.TrimPrefix(route.Path, "/api")
		path = strings.TrimSuffix(path, "{name}")
		path = strings.TrimSuffix(path, "{run_id}")
		if !strings.Contains(source, path) {
			t.Errorf("CLI surface route %s %s has no command reference", route.Method, route.Path)
		}
	}
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
