package app

import (
	"database/sql"
	"errors"
	"test-genie/agentmanager"
	"test-genie/internal/app/httpserver"
	"test-genie/internal/app/runtime"
	"testing"
)

func TestNewServerPropagatesLoadConfigError(t *testing.T) {
	originalLoadConfig := loadConfig
	originalBuildDependencies := buildDependencies
	originalNewHTTPServer := newHTTPServer
	defer func() {
		loadConfig = originalLoadConfig
		buildDependencies = originalBuildDependencies
		newHTTPServer = originalNewHTTPServer
	}()

	wantErr := errors.New("missing env")
	loadConfig = func() (*runtime.Config, error) {
		return nil, wantErr
	}

	_, err := NewServer()
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected load-config error to be returned, got %v", err)
	}
}

func TestNewServerWiresRuntimeDependenciesIntoHTTPTransport(t *testing.T) {
	originalLoadConfig := loadConfig
	originalBuildDependencies := buildDependencies
	originalNewHTTPServer := newHTTPServer
	defer func() {
		loadConfig = originalLoadConfig
		buildDependencies = originalBuildDependencies
		newHTTPServer = originalNewHTTPServer
	}()

	cfg := &runtime.Config{
		Port:          "9911",
		DatabasePath:  "/tmp/test-genie.db",
		DatabaseDSN:   "file:/tmp/test-genie.db",
		ScenariosRoot: "/tmp/scenarios",
	}
	agentSvc := agentmanager.NewAgentService(agentmanager.Config{
		ProfileName: "Test Genie Agent",
		ProfileKey:  "test-genie",
		Enabled:     true,
	})
	bootstrapped := &runtime.Bootstrapped{
		DB:           &sql.DB{},
		AgentService: agentSvc,
	}

	loadConfig = func() (*runtime.Config, error) {
		return cfg, nil
	}
	buildDependencies = func(input *runtime.Config) (*runtime.Bootstrapped, error) {
		if input != cfg {
			t.Fatalf("expected buildDependencies to receive loaded config")
		}
		return bootstrapped, nil
	}

	expectedServer := &httpserver.Server{}
	newHTTPServer = func(httpCfg httpserver.Config, deps httpserver.Dependencies) (*httpserver.Server, error) {
		if httpCfg.Port != "9911" {
			t.Fatalf("expected port to be forwarded, got %q", httpCfg.Port)
		}
		if httpCfg.ServiceName != "Test Genie API" {
			t.Fatalf("expected service name to be stable, got %q", httpCfg.ServiceName)
		}
		if deps.DB != bootstrapped.DB {
			t.Fatal("expected DB dependency to be forwarded")
		}
		if deps.AgentService != agentSvc {
			t.Fatal("expected agent service to be forwarded")
		}
		return expectedServer, nil
	}

	server, err := NewServer()
	if err != nil {
		t.Fatalf("expected NewServer to succeed, got %v", err)
	}
	if server != expectedServer {
		t.Fatal("expected NewServer to return the constructed HTTP server")
	}
}
