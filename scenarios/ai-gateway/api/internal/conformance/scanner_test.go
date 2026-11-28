package conformance_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
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

func TestScannerRejectsUndeclaredLiteralGatewayRoleAndAcceptsCatalogRole(t *testing.T) {
	repo := t.TempDir()
	writeFile(t, repo, "scenarios/ai-gateway/config/inference-role-catalog.json", `{"roles":{"summarize.faithful":{},"classify.fast":{}}}`)
	caller := "package caller\nimport \"github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference\"\n"
	writeFile(t, repo, "scenarios/demo/api/caller.go", caller+"var request = RunRequest{Role: \"summarize.faithful\"}\nconst defaultGatewayRole = \"summary.magic\"\n")

	report, err := conformance.NewScanner().Scan(context.Background(), conformance.ScanRequest{
		Scenario: "demo",
		Path:     filepath.Join(repo, "scenarios", "demo"),
	})
	require.NoError(t, err)
	var roleFindings []*conformancev1.ConformanceFinding
	for _, finding := range report.Findings {
		if finding.GetRuleId() == "ai.gateway_role_unreachable" {
			roleFindings = append(roleFindings, finding)
		}
	}
	require.Len(t, roleFindings, 1)
	require.Equal(t, "api/caller.go:4", roleFindings[0].GetPath())

	writeFile(t, repo, "scenarios/demo/api/unrelated.go", "package caller\n// an unrelated LPBS contract references ai-gateway discovery\nvar role = map[string]string{\"role\": \"chat.default\"}\n")
	report, err = conformance.NewScanner().Scan(context.Background(), conformance.ScanRequest{
		Scenario: "demo",
		Path:     filepath.Join(repo, "scenarios", "demo"),
	})
	require.NoError(t, err)
	for _, finding := range report.Findings {
		if finding.GetPath() == "api/unrelated.go:3" {
			t.Fatalf("unrelated role-shaped field was treated as Gateway call: %+v", finding)
		}
	}
}

func TestScannerAuditsRoleConstantsInGenericGatewayRouteCallers(t *testing.T) {
	repo := t.TempDir()
	writeFile(t, repo, "scenarios/ai-gateway/config/inference-role-catalog.json", `{"roles":{"classify.fast":{},"embedding.default":{}}}`)
	writeFile(t, repo, "scenarios/demo/api/inference.go", `package api
import routingconnect "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/routing/routing_v1connect"

const ClassificationRole = "classify.routing"
func execute(client routingconnect.RoutingServiceClient) {
    _, _ = client.ExecuteRoute(nil, nil)
}

func TestSignalInboxGenericRouteRolesAreCatalogReachable(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", "..", "..", "..", "signal-inbox"))
	require.NoError(t, err)
	report, err := conformance.NewScanner().Scan(context.Background(), conformance.ScanRequest{
		Scenario: "signal-inbox",
		Path:     root,
	})
	require.NoError(t, err)
	for _, finding := range report.Findings {
		require.NotEqual(t, "ai.gateway_role_unreachable", finding.GetRuleId(), "%s: %s", finding.GetPath(), finding.GetMessage())
	}
}
`)
	report, err := conformance.NewScanner().Scan(context.Background(), conformance.ScanRequest{
		Scenario: "demo",
		Path:     filepath.Join(repo, "scenarios", "demo"),
	})
	require.NoError(t, err)
	var findings []*conformancev1.ConformanceFinding
	for _, finding := range report.Findings {
		if finding.GetRuleId() == "ai.gateway_role_unreachable" {
			findings = append(findings, finding)
		}
	}
	require.Len(t, findings, 1)
	require.Equal(t, "api/inference.go:4", findings[0].GetPath())
}

func TestScannerRejectsImplicitSharedRerankRoleOnGatewayAdapter(t *testing.T) {
	repo := t.TempDir()
	writeFile(t, repo, "scenarios/ai-gateway/config/inference-role-catalog.json", `{"roles":{"summarize.faithful":{}}}`)
	writeFile(t, repo, "scenarios/demo/api/reranker.go", `package api
import "search-hub/internal/aigateway"
func makeReranker() { _ = aisearch.NewLLMRerankerWithRunner("", gatewayRunner) }
func gatewayRunner() { _ = aigateway.NewTextGenerator(resolve, client, "caller.source") }
`)
	report, err := conformance.NewScanner().Scan(context.Background(), conformance.ScanRequest{
		Scenario: "demo",
		Path:     filepath.Join(repo, "scenarios", "demo"),
	})
	require.NoError(t, err)
	var findings []*conformancev1.ConformanceFinding
	for _, finding := range report.Findings {
		if finding.GetRuleId() == "ai.gateway_role_implicit" {
			findings = append(findings, finding)
		}
	}
	require.Len(t, findings, 1)
	require.Equal(t, "api/reranker.go:3", findings[0].GetPath())
}

func TestScannerFailsClosedWhenGatewayTaskRoleCatalogIsInvalid(t *testing.T) {
	repo := t.TempDir()
	writeFile(t, repo, "scenarios/ai-gateway/config/inference-role-catalog.json", `{broken`)
	writeFile(t, repo, "scenarios/demo/api/caller.go", `package api
import "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
var request = inference.RunRequest{Role: "summarize.faithful"}
`)
	_, err := conformance.NewScanner().Scan(context.Background(), conformance.ScanRequest{
		Scenario: "demo",
		Path:     filepath.Join(repo, "scenarios", "demo"),
	})
	require.ErrorContains(t, err, "decode AI Gateway task-role catalog")
}

func TestConcreteModelTagGateBlocksRuntimeLiteralsAndIgnoresEvidence(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "api/model.go", `package api
const selectedModel = "qwen3.5:9b"
// An old example: claude-3.5-sonnet
const runner = "claude-code"
`)
	writeFile(t, root, "api/model_test.go", `package api
const fixtureModel = "gpt-4o-mini"
`)
	writeFile(t, root, "api/testdata/request.json", `{"model":"llama3.2:3b"}`)
	writeFile(t, root, "api/registry.seed.json", `{"model":"gpt-4o"}`)
	writeFile(t, root, "docs/selection.md", `Example: claude-3.5-sonnet`)

	report, err := conformance.NewScanner().Scan(context.Background(), conformance.ScanRequest{
		Scenario: "fixture",
		Path:     root,
	})
	require.NoError(t, err)
	var slugFindings []*conformancev1.ConformanceFinding
	for _, finding := range report.Findings {
		if finding.GetRuleId() == "ai.concrete_model_slug" {
			slugFindings = append(slugFindings, finding)
		}
	}
	require.Len(t, slugFindings, 1)
	require.Equal(t, "high", slugFindings[0].GetSeverity())
	require.Equal(t, "api/model.go:2", slugFindings[0].GetPath())
}

func TestConcreteModelTagGateAllowsOnlyValidatedRegistryAndKnownProviderReferences(t *testing.T) {
	repo := t.TempDir()
	writeFile(t, repo, "scenarios/demo/api/internal/modelcatalog/models.seed.json", `{"schema_version":"1.0.0","models":[{"id":"qwen3:1.7b","notes":"unregistered qwen3:9b"}]}`)
	writeFile(t, repo, "scenarios/demo/api/internal/modelcatalog/catalog.go", `const SupportedSchemaVersion = "1.0.0"\nfunc Load(raw []byte) { c.Validate() }\n//go:generate render\nvar legacyDefaultExceptions = map[string]string{"qwen3:1.7b": "preserve existing role"}`)
	writeFile(t, repo, "scenarios/demo/api/internal/modelcatalog/validate.go", `func (c *Catalog) ValidationIssues() {}`)
	writeFile(t, repo, "scenarios/demo/api/internal/modelcatalog/catalog_test.go", `func TestEmbeddedSeedIsValid() {}\nfunc TestDocsMatchRenderer() {}\nfunc TestSummarizeRecordsMirrorResourceOllamaPolicy() { _ = "model-policy.json"; _ = "resource-ollama"; _ = "Fallbacks []string"; _ = "Model     string"; _ = "summarize.default" }`)
	writeFile(t, repo, "scenarios/demo/api/internal/providercatalog/catalog.go", `func Load() { r.validate() }\nfunc (r *Registry) validate() error {}`)
	writeFile(t, repo, "scenarios/demo/api/internal/providercatalog/catalog.seed.json", `{"models":["qwen3:1.7b","qwen3:9b"]}`)
	writeFile(t, repo, "scenarios/demo/api/internal/providercatalog/conformance_test.go", `func TestEveryProviderModelReferenceResolves() { _ = models.ByID(modelID) }`)
	writeFile(t, repo, "scenarios/demo/api/internal/providercatalog/render.go", `func RenderTS() { _ = "catalog.seed.json" }`)
	writeFile(t, repo, "scenarios/demo/api/internal/providercatalog/render_test.go", `func TestGeneratedTSProviderCatalogIsNotStale() { _ = RenderTS() }`)
	writeFile(t, repo, "scenarios/demo/ui/src/features/providers/providerCatalog.generated.ts", `models: ["qwen3:1.7b", "qwen3:9b"]`)
	writeFile(t, repo, "scenarios/demo/requirements/02-post-launch/module.json", `{"description":"qualified qwen3:1.7b route"}`)
	writeFile(t, repo, "scenarios/demo/api/runtime.go", `package api\nconst selectedModel = "qwen3:1.7b"`)

	report, err := conformance.NewScanner().Scan(context.Background(), conformance.ScanRequest{
		Scenario: "demo",
		Path:     filepath.Join(repo, "scenarios", "demo"),
	})
	require.NoError(t, err)
	var slugPaths []string
	for _, finding := range report.Findings {
		if finding.GetRuleId() == "ai.concrete_model_slug" {
			slugPaths = append(slugPaths, finding.GetPath())
		}
	}
	require.Equal(t, []string{
		"api/internal/modelcatalog/models.seed.json:1",
		"api/internal/providercatalog/catalog.seed.json:1",
		"api/runtime.go:1",
		"ui/src/features/providers/providerCatalog.generated.ts:1",
	}, slugPaths)
}

func TestConcreteModelTagGateDoesNotTrustUnvalidatedRegistrySeed(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "api/internal/modelcatalog/models.seed.json", `{"schema_version":"1.0.0","models":[{"id":"qwen3:1.7b"}]}`)
	writeFile(t, root, "api/internal/providercatalog/catalog.seed.json", `{"models":["qwen3:1.7b"]}`)
	report, err := conformance.NewScanner().Scan(context.Background(), conformance.ScanRequest{Scenario: "fixture", Path: root})
	require.NoError(t, err)
	var slugPaths []string
	for _, finding := range report.Findings {
		if finding.GetRuleId() == "ai.concrete_model_slug" {
			slugPaths = append(slugPaths, finding.GetPath())
		}
	}
	require.Equal(t, []string{"api/internal/modelcatalog/models.seed.json:1", "api/internal/providercatalog/catalog.seed.json:1"}, slugPaths)
}

func TestScannerSkipsDocumentationTestsAndLargeEvidenceBeforeOpening(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "api/main.go", `package api
const endpoint = "https://openrouter.ai/api/v1/chat/completions"
`)
	writeFile(t, root, "api/client_test.go", `package api
const fixture = "https://openrouter.ai/api/v1/chat/completions"
`)
	writeFile(t, root, "api/testdata/fixture.json", `{"model":"gpt-4o"}`)
	writeFile(t, root, "api/registry.seed.json", `{"model":"gpt-4o"}`)
	writeFile(t, root, "docs/large-evidence.json", `{"payload":"`+strings.Repeat("x", 5*1024*1024)+`"}`)
	writeFile(t, root, "docs/provider.md", `https://openrouter.ai/api/v1/chat/completions`)
	writeFile(t, root, "tmp/baseline/providers-list.json", `{"model":"anthropic/claude-3.5-sonnet"}`)

	report, err := conformance.NewScanner().Scan(context.Background(), conformance.ScanRequest{
		Scenario: "fixture",
		Path:     root,
	})
	require.NoError(t, err)
	var directProviderFindings []*conformancev1.ConformanceFinding
	for _, finding := range report.Findings {
		if finding.GetRuleId() == "ai.direct_openrouter_http" {
			directProviderFindings = append(directProviderFindings, finding)
		}
	}
	require.Len(t, directProviderFindings, 1)
	require.Equal(t, "api/main.go:2", directProviderFindings[0].GetPath())
}

func TestScannerFindsOpenRouterAPIHostWithoutAPISubdomain(t *testing.T) { // [REQ:AIGW-CONFORMANCE-INVENTORY]
	root := t.TempDir()
	writeFile(t, root, "api/client.ts", `
const endpoint = "https://openrouter.ai/api/v1/chat/completions";
`)

	report, err := conformance.NewScanner().Scan(context.Background(), conformance.ScanRequest{
		Scenario: "fixture",
		Path:     root,
	})
	require.NoError(t, err)
	requireRule(t, report.Findings, "ai.direct_openrouter_http")
}

func TestScannerFindsProviderClientImports(t *testing.T) { // [REQ:AIGW-CONFORMANCE-BOUNDARY]
	root := t.TempDir()
	writeFile(t, root, "api/client.go", `
package api

import resourceollama "github.com/vrooli/resource-ollama/client"
`)

	report, err := conformance.NewScanner().Scan(context.Background(), conformance.ScanRequest{
		Scenario: "fixture",
		Path:     root,
	})
	require.NoError(t, err)
	requireRule(t, report.Findings, "ai.non_gateway_provider_client_import")
}

func TestBrowserAutomationStudioHasNoHighConformanceFindings(t *testing.T) { // [REQ:AIGW-CONFORMANCE-BAS]
	root := browserAutomationStudioRoot(t)
	report, err := conformance.NewScanner().Scan(context.Background(), conformance.ScanRequest{
		Scenario: "browser-automation-studio",
		Path:     root,
	})
	require.NoError(t, err)
	for _, finding := range report.Findings {
		require.NotEqual(t, "high", finding.GetSeverity(), "%s: %s", finding.GetPath(), finding.GetMessage())
	}
}

func TestAudioToolsGovernedModelCatalogIsNotMistakenForCallerSelection(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", "..", "..", "..", "audio-tools"))
	require.NoError(t, err)
	report, err := conformance.NewScanner().Scan(context.Background(), conformance.ScanRequest{Scenario: "audio-tools", Path: root})
	require.NoError(t, err)
	for _, finding := range report.Findings {
		if finding.GetRuleId() == "ai.concrete_model_slug" {
			require.NotContains(t, finding.GetPath(), "api/internal/modelcatalog/models.seed.json")
			require.NotContains(t, finding.GetPath(), "api/internal/modelcatalog/catalog.go")
			require.NotContains(t, finding.GetPath(), "api/internal/providercatalog/catalog.seed.json")
			require.NotContains(t, finding.GetPath(), "requirements/02-post-launch/module.json")
			require.NotContains(t, finding.GetPath(), "ui/src/features/providers/providerCatalog.generated.ts")
		}
	}
}

func TestBrowserAutomationStudioDoesNotRetainProviderRegistryOrMediaExecutor(t *testing.T) { // [REQ:AIGW-CONFORMANCE-BAS]
	root := browserAutomationStudioRoot(t)
	registry := filepath.Join(root, "playwright-driver", "src", "ai", "vision-client", "model-registry.ts")
	_, err := os.Stat(registry)
	require.ErrorIs(t, err, os.ErrNotExist, "concrete provider model registry must not return to the BAS driver")

	err = filepath.WalkDir(filepath.Join(root, "playwright-driver", "src"), func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || filepath.Ext(path) != ".ts" {
			return nil
		}
		contents, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		require.NotContains(t, string(contents), "MediaExecutor", path)
		require.NotContains(t, string(contents), "calculateCost", path)
		return nil
	})
	require.NoError(t, err)
}

func TestBrowserAutomationStudioDoesNotReadProviderSecrets(t *testing.T) { // [REQ:AIGW-CONFORMANCE-BAS]
	root := browserAutomationStudioRoot(t)
	report, err := conformance.NewScanner().Scan(context.Background(), conformance.ScanRequest{
		Scenario: "browser-automation-studio",
		Path:     root,
	})
	require.NoError(t, err)
	for _, finding := range report.Findings {
		require.NotEqual(t, "ai.invalid_provider_secret_env", finding.GetRuleId(), "%s: %s", finding.GetPath(), finding.GetMessage())
	}
}

func browserAutomationStudioRoot(t *testing.T) string {
	t.Helper()
	// Go runs package tests from their source directory, including with
	// -trimpath, where runtime.Caller no longer contains a filesystem path.
	root, err := filepath.Abs(filepath.Join("..", "..", "..", "..", "browser-automation-studio"))
	require.NoError(t, err)
	return root
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

func TestScannerTreatsResourceRuntimeBuildAsMetadata(t *testing.T) { // [REQ:AIGW-CONFORMANCE-MATURITY]
	root := t.TempDir()
	writeFile(t, root, "api/encoder.go", `package api
const resourceRuntimeBuild = "resource-ollama-gateway/0.1.0+ollama-server/0.30.10"
`)
	report, err := conformance.NewScanner().Scan(context.Background(), conformance.ScanRequest{Scenario: "fixture", Path: root})
	require.NoError(t, err)
	for _, finding := range report.Findings {
		require.NotEqual(t, "ai.resource_gateway_missing_role", finding.GetRuleId())
	}
}

func TestScannerRejectsEmptyReviewedExceptionMetadata(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "api/runner.go", `package api
// ai-gateway-exception owner= reason= expires= replacement=
const command = "resource-ollama policy roles --json"
`)

	report, err := conformance.NewScanner().Scan(context.Background(), conformance.ScanRequest{Scenario: "fixture", Path: root})
	require.NoError(t, err)
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
