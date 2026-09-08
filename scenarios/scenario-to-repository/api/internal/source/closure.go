package source

import (
	closure "github.com/vrooli/vrooli/scenarios/scenario-dependency-analyzer/api/closure"
	"path/filepath"
	"strings"
)

// AnalyzeClosure delegates to Scenario Dependency Analyzer, the authoritative
// owner of source dependency closure and local-module obligations.
func AnalyzeClosure(root, scenario, sourceDigest string) (Closure, error) {
	return closure.NewResolver().Resolve(root, scenario, sourceDigest)
}

func privatePath(path string) bool {
	base := strings.ToLower(filepath.Base(path))
	return strings.HasPrefix(base, ".env") || strings.HasSuffix(base, ".pem") || strings.HasSuffix(base, ".key") || strings.HasSuffix(base, ".p12") || strings.Contains(filepath.ToSlash(path), ".vrooli/plan-artifacts/")
}
