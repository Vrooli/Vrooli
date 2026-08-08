package federation

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"source-ledger/internal/module"
)

var descriptorMu sync.Mutex

// ProviderIDForScope returns the stable Search Hub leaf owned by source-ledger
// for a scope. agent-memory keeps the fleet-facing name used by the plan; all
// other scopes are namespaced beneath source-ledger.scope so two ledgers can
// never claim the same provider group.
func ProviderIDForScope(scope string) string {
	scope = strings.TrimSpace(scope)
	if scope == "agent-memory" {
		return "source-ledger.agent-memory"
	}
	return "source-ledger.scope." + scope
}

// AppendScopeProvider materializes a search-hub descriptor for a newly
// registered scope. The committed agent-memory provider is the template; each
// generated provider routes the same recall RPC with its scope fixed in the
// request body, so federated discovery never crosses ledgers.
func AppendScopeProvider(path, scope string) error {
	descriptorMu.Lock()
	defer descriptorMu.Unlock()
	raw, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read search descriptor: %w", err)
	}
	var document map[string]any
	if err := json.Unmarshal(raw, &document); err != nil {
		return fmt.Errorf("parse search descriptor: %w", err)
	}
	providers, ok := document["providers"].([]any)
	if !ok || len(providers) == 0 {
		return fmt.Errorf("search descriptor has no provider template")
	}
	providerID := ProviderIDForScope(scope)
	for _, rawProvider := range providers {
		if provider, ok := rawProvider.(map[string]any); ok && provider["provider_id"] == providerID {
			return nil
		}
	}
	template, ok := providers[0].(map[string]any)
	if !ok {
		return fmt.Errorf("search descriptor provider template is malformed")
	}
	templateBytes, err := json.Marshal(template)
	if err != nil {
		return err
	}
	var provider map[string]any
	if err := json.Unmarshal(templateBytes, &provider); err != nil {
		return err
	}
	provider["provider_id"] = providerID
	provider["provider_group"] = "source-ledger"
	provider["description"] = "Scoped durable memory ledger for " + scope + "."
	delete(provider, "tests")
	if endpoint, ok := provider["endpoint"].(map[string]any); ok {
		if httpJSON, ok := endpoint["http_json"].(map[string]any); ok {
			httpJSON["body_template"] = fmt.Sprintf(`{"query":"{{query}}","limit":{{limit}},"scope":%q}`, scope)
		}
	}
	document["providers"] = append(providers, provider)
	encoded, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return err
	}
	encoded = append(encoded, '\n')
	// Replace the descriptor atomically so search-register never observes a
	// partially-written document while a new scope is being created.
	dir := filepath.Dir(filepath.Clean(path))
	tmp, err := os.CreateTemp(dir, ".search.json.*")
	if err != nil {
		return fmt.Errorf("create temporary search descriptor: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(0o644); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("set search descriptor permissions: %w", err)
	}
	if _, err := tmp.Write(encoded); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write search descriptor: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close search descriptor: %w", err)
	}
	if err := os.Rename(tmpName, filepath.Clean(path)); err != nil {
		return fmt.Errorf("replace search descriptor: %w", err)
	}
	return nil
}

func Module() module.Module { return module.Empty("federation") }
