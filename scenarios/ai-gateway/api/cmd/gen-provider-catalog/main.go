// gen-provider-catalog is the owner of the deployed, credential-free catalog.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"ai-gateway/internal/providers"
)

func main() {
	root := flag.String("root", "../../..", "repository root from api directory")
	output := flag.String("output", "internal/catalog/providers.generated.json", "output path")
	check := flag.Bool("check", false, "verify output without writing")
	flag.Parse()
	if err := run(*root, *output, *check); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(root, output string, check bool) error {
	read := func(path string) (json.RawMessage, error) { return os.ReadFile(filepath.Join(root, path)) }
	roles, err := read("scenarios/ai-gateway/config/inference-role-catalog.json")
	if err != nil {
		return err
	}
	features, err := read("packages/monetization-go/ai-features.generated.json")
	if err != nil {
		return err
	}
	entries := map[string]any{}
	for _, spec := range providers.DefaultProviderSpecs() {
		if spec.Family != providers.FamilyOllama && spec.Family != providers.FamilyOpenAICompatible {
			continue
		}
		policy, err := read("resources/" + spec.ID + "/model-policy.json")
		if err != nil {
			return err
		}
		resourceData, err := read("resources/" + spec.ID + "/resource.json")
		if err != nil {
			return err
		}
		var resource struct {
			Endpoint       string `json:"endpoint"`
			ManagedService struct {
				Port int `json:"port"`
			} `json:"managed_service"`
			Ports []struct {
				Host int `json:"host"`
			} `json:"ports"`
			Credentials struct {
				Descriptors []struct {
					Identity string `json:"logical_id"`
					Field    string `json:"field"`
				} `json:"descriptors"`
			} `json:"credentials"`
		}
		if err := json.Unmarshal(resourceData, &resource); err != nil {
			return err
		}
		endpoint := strings.TrimSuffix(resource.Endpoint, "/models")
		if spec.Family == providers.FamilyOllama && len(resource.Ports) > 0 {
			endpoint = fmt.Sprintf("http://127.0.0.1:%d", resource.Ports[0].Host)
		}
		entry := map[string]any{"id": spec.ID, "family": spec.Family, "locality": spec.Locality, "endpoint": endpoint, "policy": policy}
		if len(resource.Credentials.Descriptors) > 0 {
			entry["credential_identity"] = resource.Credentials.Descriptors[0].Identity
			entry["credential_field"] = resource.Credentials.Descriptors[0].Field
		}
		entries[spec.ID] = entry
	}
	// Suppliers without a local resource wrapper are owned as declarations.
	// A new compatible endpoint needs data here, never a new adapter branch.
	supplierPaths, err := filepath.Glob(filepath.Join(root, "scenarios", "ai-gateway", "config", "suppliers", "*.json"))
	if err != nil {
		return err
	}
	for _, path := range supplierPaths {
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		var entry struct {
			ID       string          `json:"id"`
			Family   string          `json:"family"`
			Endpoint string          `json:"endpoint"`
			Policy   json.RawMessage `json:"policy"`
		}
		if err = json.Unmarshal(data, &entry); err != nil {
			return err
		}
		if entry.ID == "" || filepath.Base(entry.ID) != entry.ID || entry.Family != providers.FamilyOpenAICompatible || entry.Endpoint == "" || len(entry.Policy) == 0 {
			return fmt.Errorf("invalid supplier declaration %s", path)
		}
		if _, exists := entries[entry.ID]; exists {
			return fmt.Errorf("supplier %s has two declaration owners", entry.ID)
		}
		entries[entry.ID] = json.RawMessage(data)
	}
	harnesses := map[string]json.RawMessage{}
	paths, err := filepath.Glob(filepath.Join(root, "resources", "*", "model-policy.json"))
	if err != nil {
		return err
	}
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		var p struct {
			Billing json.RawMessage `json:"billing"`
		}
		if err = json.Unmarshal(data, &p); err != nil {
			return err
		}
		if len(p.Billing) > 0 {
			harnesses[filepath.Base(filepath.Dir(path))] = data
		}
	}
	data, err := json.MarshalIndent(map[string]any{"roles": roles, "features": features, "providers": entries, "harnesses": harnesses}, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	old, _ := os.ReadFile(output)
	if bytes.Equal(old, data) {
		return nil
	}
	if check {
		return fmt.Errorf("provider catalog stale; run make provider-catalog from scenarios/ai-gateway")
	}
	return os.WriteFile(output, data, 0o644)
}
