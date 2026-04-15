package main

import (
	"testing"

	"tunnel-manager/cli/internal/flags"
)

// [REQ:CLI-ROUTE-CRUD-001] Route CRUD commands are registered
func TestRouteCRUDCommandsRegistered(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}

	groups := app.subcommandGroups()
	wantCmds := map[string]bool{"get": false, "create": false, "update": false, "delete": false}

	for _, group := range groups {
		if group.Name != "route" {
			continue
		}
		for _, cmd := range group.Subcommands {
			if _, ok := wantCmds[cmd.Name]; ok {
				wantCmds[cmd.Name] = true
				if !group.NeedsAPI && !cmd.NeedsAPI {
					t.Errorf("route %s should require API", cmd.Name)
				}
			}
		}
	}

	for name, found := range wantCmds {
		if !found {
			t.Errorf("subcommand route %q not registered", name)
		}
	}
}

// [REQ:CLI-ROUTE-CRUD-002] Route CRUD API paths
func TestRouteCRUDAPIPaths(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}

	tests := []struct {
		input string
		want  string
	}{
		{"/routes/1", "/api/v1/routes/1"},
		{"/routes/42", "/api/v1/routes/42"},
	}

	for _, tc := range tests {
		got := app.core.APIPath(tc.input)
		if got != tc.want {
			t.Errorf("apiPath(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

// [REQ:CLI-ROUTE-CRUD-003] Flag parsing helpers
func TestParseFlag(t *testing.T) {
	tests := []struct {
		args    []string
		name    string
		wantVal string
		wantOK  bool
	}{
		{[]string{"--port", "8080"}, "port", "8080", true},
		{[]string{"--subdomain", "api", "--port", "3000"}, "port", "3000", true},
		{[]string{"--subdomain", "api"}, "port", "", false},
		{[]string{"--port"}, "port", "", false},
		{[]string{}, "port", "", false},
	}

	for _, tc := range tests {
		val, ok := flags.StringValue(tc.args, tc.name)
		if val != tc.wantVal || ok != tc.wantOK {
			t.Errorf("StringValue(%v, %q) = (%q, %v), want (%q, %v)", tc.args, tc.name, val, ok, tc.wantVal, tc.wantOK)
		}
	}
}

// [REQ:CLI-ROUTE-CRUD-004] Bool flag parsing helper
func TestParseBoolFlag(t *testing.T) {
	tests := []struct {
		args []string
		name string
		want bool
	}{
		{[]string{"--force"}, "force", true},
		{[]string{"--other"}, "force", false},
		{[]string{}, "force", false},
	}

	for _, tc := range tests {
		got := flags.BoolValue(tc.args, tc.name)
		if got != tc.want {
			t.Errorf("BoolValue(%v, %q) = %v, want %v", tc.args, tc.name, got, tc.want)
		}
	}
}
