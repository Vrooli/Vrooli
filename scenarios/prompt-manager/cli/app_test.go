package main

import "testing"

func TestNewApp_APIPortEnvVars_DoNotIncludeGenericAPIPort(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() error = %v", err)
	}

	portEnvVars := app.core.APIBaseOptions().PortEnvVars
	if len(portEnvVars) == 0 {
		t.Fatal("expected APIPortEnvVars to be configured")
	}
	if portEnvVars[0] != "PROMPT_MANAGER_API_PORT" {
		t.Fatalf("first APIPortEnvVar = %q, want %q", portEnvVars[0], "PROMPT_MANAGER_API_PORT")
	}
	for _, key := range portEnvVars {
		if key == "API_PORT" {
			t.Fatalf("unexpected generic API_PORT in APIPortEnvVars: %v", portEnvVars)
		}
	}
}
