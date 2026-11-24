package cliutil

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAntigravityModelSourceUsesTemporarySettingsOverlay(t *testing.T) {
	home := t.TempDir()
	settingsPath := filepath.Join(home, ".gemini", "antigravity-cli", "settings.json")
	if err := os.MkdirAll(filepath.Dir(settingsPath), 0700); err != nil {
		t.Fatal(err)
	}
	original := []byte(`{"modelProvider":"google","colorScheme":"tokyo night","customModelsConfig":{"customModels":[{"modelName":"gemini-api://https://example.invalid/models/other","label":"other"}]}}`)
	if err := os.WriteFile(settingsPath, original, 0600); err != nil {
		t.Fatal(err)
	}
	source := ModelSource{ID: "Vrooli credits", Kind: "vrooli_credits", Model: "fixture-model", BaseURL: "https://door.example/api/v1/ai", SecretEnv: "VROOLI_SESSION_KEY"}
	got, cleanup, err := prepareAntigravitySourceHome(source, []string{"HOME=" + home, "VROOLI_SESSION_KEY=synthetic-secret"})
	if err != nil {
		t.Fatal(err)
	}
	tempHome := environmentValue(got, "HOME")
	if tempHome == home {
		t.Fatal("Antigravity source must not edit the user's settings home")
	}
	data, err := os.ReadFile(filepath.Join(tempHome, ".gemini", "antigravity-cli", "settings.json"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) == string(original) {
		t.Fatal("custom endpoint was not projected into the temporary settings")
	}
	if string(data) == "" || len(data) == 0 {
		t.Fatal("temporary settings are empty")
	}
	var overlay map[string]any
	if err := json.Unmarshal(data, &overlay); err != nil {
		t.Fatal(err)
	}
	if overlay["modelProvider"] != "gemini" || overlay["colorScheme"] != "tokyo night" {
		t.Fatalf("source settings did not preserve user preferences: %#v", overlay)
	}
	if strings.Contains(string(data), "synthetic-secret") {
		t.Fatal("API key was written to settings")
	}
	unchanged, err := os.ReadFile(settingsPath)
	if err != nil || string(unchanged) != string(original) {
		t.Fatal("user settings were modified", err)
	}
	cleanup()
	if _, err := os.Stat(tempHome); !os.IsNotExist(err) {
		t.Fatalf("temporary home remains after cleanup: %v", err)
	}
}
