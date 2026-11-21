package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"ai-gateway/internal/providers"
	"ai-gateway/internal/rolecatalog"
)

type catalogInputs struct {
	roles         json.RawMessage
	features      json.RawMessage
	creditOwners  json.RawMessage
	specs         []providers.ProviderSpec
	resourceRoles map[string]map[string]struct{}
	policies      map[string]json.RawMessage
}

func loadInputs(root string) (catalogInputs, error) {
	read := func(path string) ([]byte, error) { return os.ReadFile(filepath.Join(root, path)) }
	roles, err := read(roleCatalogRelPath)
	if err != nil {
		return catalogInputs{}, err
	}
	catalog, err := rolecatalog.LoadCatalog(filepath.Join(root, roleCatalogRelPath))
	if err != nil {
		return catalogInputs{}, err
	}
	features, err := read("packages/monetization-go/ai-features.generated.json")
	if err != nil {
		return catalogInputs{}, err
	}
	credits, err := read("packages/monetization-go/credit-features.generated.json")
	if err != nil {
		return catalogInputs{}, err
	}
	inputs := catalogInputs{roles: roles, features: features, creditOwners: credits, specs: providers.DefaultProviderSpecs(), resourceRoles: map[string]map[string]struct{}{}, policies: map[string]json.RawMessage{}}
	for _, spec := range inputs.specs {
		policy, err := read("resources/" + spec.ID + "/model-policy.json")
		if err != nil {
			return catalogInputs{}, fmt.Errorf("read model policy for provider %q: %w", spec.ID, err)
		}
		resourceRoleSet, err := parseResourceRoles(policy)
		if err != nil {
			return catalogInputs{}, fmt.Errorf("resources/%s/model-policy.json: %w", spec.ID, err)
		}
		inputs.resourceRoles[spec.ID], inputs.policies[spec.ID] = resourceRoleSet, policy
	}
	if err := validateRoleCandidates(catalog, inputs.resourceRoles); err != nil {
		return catalogInputs{}, err
	}
	if err := validateServiceModelRoles(root, inputs.resourceRoles); err != nil {
		return catalogInputs{}, err
	}
	return inputs, nil
}

func providerEntries(root string, inputs catalogInputs) (map[string]any, error) {
	entries := map[string]any{}
	for _, spec := range inputs.specs {
		if spec.Family != providers.FamilyOllama && spec.Family != providers.FamilyOpenAICompatible {
			continue
		}
		resourceData, err := os.ReadFile(filepath.Join(root, "resources", spec.ID, "resource.json"))
		if err != nil {
			return nil, err
		}
		entry, err := resourceEntry(spec, inputs.policies[spec.ID], resourceData)
		if err != nil {
			return nil, err
		}
		entries[spec.ID] = entry
	}
	return entries, nil
}

func resourceEntry(spec providers.ProviderSpec, policy, resourceData []byte) (map[string]any, error) {
	var resource struct {
		Endpoint string `json:"endpoint"`
		Ports    []struct {
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
		return nil, err
	}
	endpoint := strings.TrimSuffix(resource.Endpoint, "/models")
	if spec.Family == providers.FamilyOllama && len(resource.Ports) > 0 {
		endpoint = fmt.Sprintf("http://127.0.0.1:%d", resource.Ports[0].Host)
	}
	entry := map[string]any{"id": spec.ID, "family": spec.Family, "locality": spec.Locality, "endpoint": endpoint, "policy": json.RawMessage(policy)}
	if len(resource.Credentials.Descriptors) > 0 {
		entry["credential_identity"] = resource.Credentials.Descriptors[0].Identity
		entry["credential_field"] = resource.Credentials.Descriptors[0].Field
	}
	return entry, nil
}

func addSupplierEntries(root string, entries map[string]any) error {
	paths, err := filepath.Glob(filepath.Join(root, "scenarios", "ai-gateway", "config", "suppliers", "*.json"))
	if err != nil {
		return err
	}
	registered := make(map[string]struct{})
	for _, spec := range providers.DefaultProviderSpecs() {
		registered[spec.ID] = struct{}{}
	}
	supplierIDs := make(map[string]string, len(paths))
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if err := addSupplierEntry(root, path, data, entries, registered, supplierIDs); err != nil {
			return err
		}
	}
	return nil
}

type supplierDeclaration struct {
	ID       string          `json:"id"`
	Family   string          `json:"family"`
	Locality string          `json:"locality"`
	Endpoint string          `json:"endpoint"`
	Policy   json.RawMessage `json:"policy"`
}

func addSupplierEntry(root, path string, data []byte, entries map[string]any, registered map[string]struct{}, supplierIDs map[string]string) error {
	var entry supplierDeclaration
	if err := json.Unmarshal(data, &entry); err != nil {
		return fmt.Errorf("invalid supplier declaration %s: %w", path, err)
	}
	if err := validateSupplierDeclaration(path, entry); err != nil {
		return err
	}
	if _, exists := registered[entry.ID]; exists {
		return fmt.Errorf("supplier %s duplicates registered provider declaration", entry.ID)
	}
	if previous, exists := supplierIDs[entry.ID]; exists {
		return fmt.Errorf("supplier %s is declared by both %s and %s", entry.ID, previous, filepath.Base(path))
	}
	if _, exists := entries[entry.ID]; exists {
		return fmt.Errorf("supplier %s has two declaration owners", entry.ID)
	}
	if err := validateNoResourceDeclaration(root, entry.ID); err != nil {
		return err
	}
	supplierIDs[entry.ID] = filepath.Base(path)
	entries[entry.ID] = json.RawMessage(data)
	return nil
}

func validateSupplierDeclaration(path string, entry supplierDeclaration) error {
	if err := validateSupplierIdentity(path, entry); err != nil {
		return err
	}
	if err := validateSupplierPolicy(path, entry.Policy); err != nil {
		return err
	}
	return nil
}

func validateSupplierIdentity(path string, entry supplierDeclaration) error {
	if !isValidSupplierID(entry.ID) || !isValidSupplierMetadata(entry) {
		return fmt.Errorf("invalid supplier declaration %s", path)
	}
	return nil
}

func isValidSupplierID(id string) bool {
	return id != "" && strings.TrimSpace(id) == id && filepath.Base(id) == id && id != "." && id != ".." && !strings.ContainsAny(id, `/\\`)
}

func isValidSupplierMetadata(entry supplierDeclaration) bool {
	validFamily := entry.Family == providers.FamilyOpenAICompatible
	validLocality := entry.Locality == "local" || entry.Locality == "remote"
	return validFamily && validLocality && strings.TrimSpace(entry.Endpoint) != ""
}

func validateSupplierPolicy(path string, policy json.RawMessage) error {
	var policyObject map[string]json.RawMessage
	policyIsObject := json.Unmarshal(policy, &policyObject) == nil && policyObject != nil
	if len(policy) == 0 || !json.Valid(policy) || !policyIsObject {
		return fmt.Errorf("invalid supplier declaration %s", path)
	}
	return nil
}

func validateNoResourceDeclaration(root, supplierID string) error {
	resourcePath := filepath.Join(root, "resources", supplierID, "resource.json")
	if _, err := os.Stat(resourcePath); err == nil {
		return fmt.Errorf("supplier %s duplicates resource declaration %s", supplierID, resourcePath)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("check supplier %s resource declaration: %w", supplierID, err)
	}
	return nil
}

func loadHarnesses(root string) (map[string]json.RawMessage, error) {
	paths, err := filepath.Glob(filepath.Join(root, "resources", "*", "model-policy.json"))
	if err != nil {
		return nil, err
	}
	harnesses := map[string]json.RawMessage{}
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		var policy struct {
			Billing json.RawMessage `json:"billing"`
		}
		if err := json.Unmarshal(data, &policy); err != nil {
			return nil, err
		}
		if len(policy.Billing) > 0 {
			harnesses[filepath.Base(filepath.Dir(path))] = data
		}
	}
	return harnesses, nil
}

func renderCatalog(inputs catalogInputs, entries map[string]any, harnesses map[string]json.RawMessage) ([]byte, error) {
	data, err := json.MarshalIndent(map[string]any{"roles": inputs.roles, "features": inputs.features, "credit_feature_owners": inputs.creditOwners, "providers": entries, "harnesses": harnesses}, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

func writeCatalog(output string, data []byte, check bool) error {
	old, _ := os.ReadFile(output)
	if bytes.Equal(old, data) {
		return nil
	}
	if check {
		return fmt.Errorf("provider catalog stale; run make provider-catalog from scenarios/ai-gateway")
	}
	return os.WriteFile(output, data, 0o644)
}
