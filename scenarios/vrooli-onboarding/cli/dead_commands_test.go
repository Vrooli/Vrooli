package main

import (
	"encoding/json"
	"os"
	"path/filepath"
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
