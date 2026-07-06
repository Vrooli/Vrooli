package conformance_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	conformancev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/conformance"

	"ai-gateway/internal/conformance"
)

func TestScannerFindsUnsafeProviderCoupling(t *testing.T) { // [REQ:AIGW-CONFORMANCE-INVENTORY]
	root := t.TempDir()
	writeFile(t, root, "api/ai.go", `
package api

const base = "http://localhost:11434/api/generate"
const key = "OPENROUTER_API_KEY"
const model = "qwen3:4b"
const dims = 768 // embedding vector size
`)

	report, err := conformance.NewScanner().Scan(context.Background(), conformance.ScanRequest{
		Scenario: "fixture",
		Path:     root,
	})
	require.NoError(t, err)
	require.Equal(t, "blocked-needs-investigation", report.MaturityLevel)
	requireRule(t, report.Findings, "ai.direct_ollama_http")
	requireRule(t, report.Findings, "ai.invalid_provider_secret_env")
	requireRule(t, report.Findings, "ai.concrete_model_slug")
	requireRule(t, report.Findings, "ai.hardcoded_embedding_dimensions")
	requireRule(t, report.Findings, "ai.gateway_not_adopted")
	for _, finding := range report.Findings {
		require.NotContains(t, finding.GetMessage(), "qwen3:4b")
		require.NotContains(t, finding.GetMessage(), "OPENROUTER_API_KEY")
	}
}

func TestScannerRecognizesGatewayAdoptionSignal(t *testing.T) { // [REQ:AIGW-CONFORMANCE-MATURITY] [REQ:AIGW-MIGRATION-REPORTS]
	root := t.TempDir()
	writeFile(t, root, "api/gateway_client.go", `
package api

// Calls ai-gateway with role/profile metadata.
const gateway = "ai-gateway"
`)

	report, err := conformance.NewScanner().Scan(context.Background(), conformance.ScanRequest{
		Scenario: "fixture",
		Path:     root,
	})
	require.NoError(t, err)
	require.Equal(t, "gateway-ready", report.MaturityLevel)
	require.Empty(t, report.Findings)
}

func TestScannerFindsMaturityTaxonomyRules(t *testing.T) { // [REQ:AIGW-CONFORMANCE-MATURITY] [REQ:AIGW-EMBEDDING-GOVERNANCE]
	root := t.TempDir()
	writeFile(t, root, "api/runner.go", `
package api

const command = "resource-ollama generate"
const maxContext = 32768
// ai-gateway-exception reason=temporary owner=platform
`)
	writeFile(t, root, "db/schema.sql", `
CREATE TABLE document_vectors (id TEXT PRIMARY KEY, embedding BLOB);
`)

	report, err := conformance.NewScanner().Scan(context.Background(), conformance.ScanRequest{
		Scenario: "fixture",
		Path:     root,
	})
	require.NoError(t, err)
	requireRule(t, report.Findings, "ai.resource_gateway_missing_role")
	requireRule(t, report.Findings, "ai.hardcoded_context_window")
	requireRule(t, report.Findings, "ai.embedding_metadata_missing")
	requireRule(t, report.Findings, "ai.unreviewed_exception")
}

func TestScannerAllowsReviewedExceptionMetadata(t *testing.T) { // [REQ:AIGW-CONFORMANCE-MATURITY]
	root := t.TempDir()
	writeFile(t, root, "api/runner.go", `
package api

// ai-gateway-exception owner=platform reason=diagnostic expires=2026-12-31 replacement=ai-gateway-profile
const command = "resource-ollama policy roles --json"
`)

	report, err := conformance.NewScanner().Scan(context.Background(), conformance.ScanRequest{
		Scenario: "fixture",
		Path:     root,
	})
	require.NoError(t, err)
	for _, finding := range report.Findings {
		require.NotEqual(t, "ai.unreviewed_exception", finding.GetRuleId())
		require.NotEqual(t, "ai.resource_gateway_missing_role", finding.GetRuleId())
	}
}

func writeFile(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}

func requireRule(t *testing.T, findings []*conformancev1.ConformanceFinding, ruleID string) {
	t.Helper()
	for _, finding := range findings {
		if finding.GetRuleId() == ruleID {
			return
		}
	}
	require.Failf(t, "missing rule", "rule %s not found in %#v", ruleID, findings)
}
