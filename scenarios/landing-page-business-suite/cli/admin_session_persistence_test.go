package main

import (
	"path/filepath"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

func TestAdminSessionPersistence_PreservesOtherBaseSessions(t *testing.T) {
	dir := t.TempDir()

	// Core config file only exists to provide a stable config directory for admin_session.json.
	coreCfg, err := cliutil.NewConfigFile(filepath.Join(dir, "config.json"))
	if err != nil {
		t.Fatalf("new config file: %v", err)
	}

	base := "http://localhost:1234"
	core := &cliapp.ScenarioApp{
		ConfigFile: coreCfg,
		APIClient: cliutil.NewAPIClient(
			cliutil.NewHTTPClient(cliutil.HTTPClientOptions{}),
			func() cliutil.APIBaseOptions {
				return cliutil.APIBaseOptions{Override: base}
			},
			func() string { return "" },
		),
	}
	app := &App{core: core}

	if err := app.saveAdminSession(adminSessionConfig{Session: "local-cookie"}); err != nil {
		t.Fatalf("save local session: %v", err)
	}

	base = "https://vrooli.com"
	if err := app.saveAdminSession(adminSessionConfig{Session: "remote-cookie"}); err != nil {
		t.Fatalf("save remote session: %v", err)
	}

	base = "http://localhost:1234"
	cfg, err := app.loadAdminSession()
	if err != nil {
		t.Fatalf("load local session: %v", err)
	}
	if cfg.Session != "local-cookie" {
		t.Fatalf("expected local-cookie, got %q", cfg.Session)
	}
}

