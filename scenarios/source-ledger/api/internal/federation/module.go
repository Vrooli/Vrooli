package federation

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"

	"source-ledger/internal/module"
)

var descriptorMu sync.Mutex

// ProviderIDForScope retains the legacy naming helper for callers that need to
// explain historical registry rows. Live registration uses source-ledger.scopes
// and passes the selected scope as a query facet.
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
	if strings.TrimSpace(scope) == "" {
		return fmt.Errorf("scope is required")
	}
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
	// Scope selection is now a query facet carried by the single
	// source-ledger.scopes descriptor. Creating a scope never mutates the
	// descriptor or grows the registry; the policy registry remains the source
	// of truth and the router passes the facet at query time.
	_ = document
	_ = providers
	return nil
}

func Module() module.Module { return module.Empty("federation") }
