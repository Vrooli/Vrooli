package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestRoutesFromSourceFindsEveryMethod(t *testing.T) {
	dir := t.TempDir()
	source := `package main
func register(router interface{ HandleFunc(string, any) route }) {
  router.HandleFunc("/health", nil).Methods("GET")
  router.HandleFunc("/items/{id}", nil).Methods("GET", "POST")
}`
	path := filepath.Join(dir, "main.go")
	if err := os.WriteFile(path, []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	routes, err := routesFromSource(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(routes) != 3 || routes[1] != (route{Path: "/items/{id}", Method: "GET"}) || routes[2].Method != "POST" {
		t.Fatalf("routes = %#v", routes)
	}
}

func TestGenerateRejectsRouterMetadataDrift(t *testing.T) {
	dir := t.TempDir()
	routerPath := filepath.Join(dir, "main.go")
	metadataPath := filepath.Join(dir, "endpoints.json")
	outputPath := filepath.Join(dir, "generated.json")
	if err := os.WriteFile(routerPath, []byte(`package main
func register(router interface{ HandleFunc(string, any) route }) {
  router.HandleFunc("/new", nil).Methods("GET")
}`), 0o600); err != nil {
		t.Fatal(err)
	}
	metadata := []byte(`{"$schema":"schema","version":"1","service":"vrooli-onboarding","categories":[],"endpoints":[{"id":"old","path":"/old","method":"GET"}]}`)
	if err := os.WriteFile(metadataPath, metadata, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := generate(routerPath, metadataPath, outputPath); err == nil {
		t.Fatal("expected router/metadata drift to fail generation")
	}
}

func TestGeneratePreservesMetadataAndUsesRouterOrder(t *testing.T) {
	dir := t.TempDir()
	routerPath := filepath.Join(dir, "main.go")
	metadataPath := filepath.Join(dir, "endpoints.json")
	outputPath := filepath.Join(dir, "generated.json")
	if err := os.WriteFile(routerPath, []byte(`package main
func register(router interface{ HandleFunc(string, any) route }) {
  router.HandleFunc("/second", nil).Methods("POST")
  router.HandleFunc("/first", nil).Methods("GET")
}`), 0o600); err != nil {
		t.Fatal(err)
	}
	metadata := []byte(`{"$schema":"schema","version":"1","service":"vrooli-onboarding","categories":["system"],"endpoints":[{"id":"first","path":"/first","method":"GET","summary":"first"},{"id":"second","path":"/second","method":"POST","summary":"second"}]}`)
	if err := os.WriteFile(metadataPath, metadata, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := generate(routerPath, metadataPath, outputPath); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	var result manifest
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatal(err)
	}
	var first struct {
		ID string `json:"id"`
	}
	if len(result.Endpoints) != 2 || json.Unmarshal(result.Endpoints[0], &first) != nil || first.ID != "second" {
		t.Fatalf("generated endpoints = %s", data)
	}
}
