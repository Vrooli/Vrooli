package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"scenario-to-cloud/bundle"
	"scenario-to-cloud/domain"
)

// This exercises the production bundle builder with the J13 owner scenarios
// selected directly. It keeps the packaged-catalog path testable while the
// committed LPBS manifest's transitive closure is reconciled by its owner.
// It is bundle-builder evidence, not a committed-closure or hosted-serving pass.
func TestAC2J13BundleBuilderPackagesDisabledSupplierCatalog(t *testing.T) {
	repoRoot, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatal(err)
	}
	manifest := domain.CloudManifest{
		Version:  "1.0.0",
		Scenario: domain.ManifestScenario{ID: "landing-page-business-suite"},
		Dependencies: domain.ManifestDependencies{
			Scenarios: []string{"ai-gateway", "landing-page-business-suite"},
		},
		Bundle: domain.ManifestBundle{
			Scenarios: []string{"ai-gateway", "landing-page-business-suite"},
		},
	}
	spec, err := bundle.MiniVrooliBundleSpec(repoRoot, manifest)
	if err != nil {
		t.Fatalf("resolve J13 owner-scenario bundle inputs with production builder: %v", err)
	}
	content, err := bundle.ContentManifest(repoRoot, spec)
	if err != nil {
		t.Fatalf("compute production bundle content manifest: %v", err)
	}
	entryHashes := make(map[string]string, len(content))
	for _, entry := range content {
		entryHashes[entry.Path] = entry.SHA256
	}
	bundledFile := func(path string) []byte {
		t.Helper()
		data, err := os.ReadFile(filepath.Join(repoRoot, filepath.FromSlash(path)))
		if err != nil {
			t.Fatal(err)
		}
		digest := sha256.Sum256(data)
		if got, want := entryHashes[path], hex.EncodeToString(digest[:]); got != want {
			t.Fatalf("production bundle manifest hash for %q = %q; want source hash %q", path, got, want)
		}
		return data
	}

	const catalogPath = "scenarios/ai-gateway/api/internal/catalog/providers.generated.json"
	type modelMetadata struct {
		Eligible   *bool `json:"default_eligible"`
		Provenance struct {
			Catalog struct {
				Source     string `json:"source"`
				ObservedAt string `json:"observed_at"`
			} `json:"catalog"`
			Identity struct {
				Confidence string `json:"confidence"`
				Source     string `json:"source"`
				ObservedAt string `json:"observed_at"`
			} `json:"identity"`
		} `json:"provenance"`
	}
	var generated struct {
		Providers map[string]struct {
			Policy struct {
				Provider struct {
					Terms struct {
						Review struct {
							Status string `json:"status"`
						} `json:"review"`
					} `json:"terms"`
					Routes     []json.RawMessage `json:"hosted_routes"`
					Agreements []json.RawMessage `json:"agreements"`
				} `json:"provider"`
				Models map[string]modelMetadata `json:"models"`
			} `json:"policy"`
		} `json:"providers"`
	}
	catalogBytes := bundledFile(catalogPath)
	if err := json.Unmarshal(catalogBytes, &generated); err != nil {
		t.Fatalf("production bundle input has no decodable embedded Gateway catalog: %v", err)
	}
	openRouter, ok := generated.Providers["openrouter"]
	if !ok {
		t.Fatal("bundle catalog omits the existing OpenRouter registry provider")
	}
	if openRouter.Policy.Provider.Terms.Review.Status != "proposed" || len(openRouter.Policy.Provider.Routes) != 0 || len(openRouter.Policy.Provider.Agreements) != 0 {
		t.Fatalf("OpenRouter catalog entry is not proposal-only: %+v", openRouter.Policy.Provider)
	}
	modelIdentityCount := 0
	checkCatalogModel := func(providerID, modelID string, model modelMetadata) {
		t.Helper()
		modelIdentityCount++
		if model.Eligible == nil {
			t.Errorf("bundle catalog model %s/%s has no explicit default_eligible value", providerID, modelID)
		}
		if strings.TrimSpace(model.Provenance.Catalog.Source) == "" || model.Provenance.Catalog.ObservedAt == "" {
			t.Errorf("bundle catalog model %s/%s is missing dated catalog provenance: %+v", providerID, modelID, model.Provenance.Catalog)
		}
		if providerID != "openrouter" && !strings.Contains(model.Provenance.Catalog.Source, "https://") {
			t.Errorf("bundle supplier model %s/%s is missing a direct public source URL: %+v", providerID, modelID, model.Provenance.Catalog)
		}
	}
	verifiedOpenRouterIdentityPages := 0
	unverifiedOpenRouterIdentityPages := 0
	for modelID, model := range openRouter.Policy.Models {
		checkCatalogModel("openrouter", modelID, model)
		identity := model.Provenance.Identity
		if !strings.Contains(identity.Source, "https://openrouter.ai/") || identity.ObservedAt != "2026-11-23" {
			t.Errorf("OpenRouter model %q lacks a dated direct identity-page audit: %+v", modelID, identity)
		}
		if modelID == "black-forest-labs/flux.2-klein" {
			if identity.Confidence != "low" || !strings.Contains(identity.Source, "404") {
				t.Errorf("missing FLUX.2 Klein page should remain explicit unresolved evidence: %+v", identity)
			}
			unverifiedOpenRouterIdentityPages++
			continue
		}
		if identity.Confidence != "high" {
			t.Errorf("OpenRouter model %q direct identity page confidence = %q, want high", modelID, identity.Confidence)
		}
		verifiedOpenRouterIdentityPages++
	}
	if verifiedOpenRouterIdentityPages != 26 || unverifiedOpenRouterIdentityPages != 1 {
		t.Errorf("OpenRouter model identity pages: %d confirmed, %d unresolved; want 26/1", verifiedOpenRouterIdentityPages, unverifiedOpenRouterIdentityPages)
	}

	supplierDir := filepath.Join(repoRoot, "scenarios", "ai-gateway", "config", "suppliers")
	suppliers, err := os.ReadDir(supplierDir)
	if err != nil {
		t.Fatal(err)
	}
	found := 0
	for _, supplier := range suppliers {
		if supplier.IsDir() || filepath.Ext(supplier.Name()) != ".json" {
			continue
		}
		id := strings.TrimSuffix(supplier.Name(), filepath.Ext(supplier.Name()))
		declarationPath := filepath.ToSlash(filepath.Join("scenarios", "ai-gateway", "config", "suppliers", supplier.Name()))
		declaration, err := os.ReadFile(filepath.Join(repoRoot, filepath.FromSlash(declarationPath)))
		if err != nil {
			t.Fatal(err)
		}
		if got := bundledFile(declarationPath); string(got) != string(declaration) {
			t.Errorf("bundle supplier source %q differs from the repository", id)
		}
		provider, ok := generated.Providers[id]
		if !ok {
			t.Errorf("bundle catalog omits supplier %q", id)
			continue
		}
		if provider.Policy.Provider.Terms.Review.Status != "proposed" || len(provider.Policy.Provider.Routes) != 0 || len(provider.Policy.Provider.Agreements) != 0 {
			t.Errorf("supplier %q is executable or not proposal-only: %+v", id, provider.Policy.Provider)
		}
		for modelID, model := range provider.Policy.Models {
			checkCatalogModel(id, modelID, model)
			if model.Eligible == nil || *model.Eligible {
				t.Errorf("supplier %q model %q is missing an explicit disabled flag", id, modelID)
			}
		}
		found++
	}
	if found != 18 {
		t.Fatalf("bundle carried %d supplier declarations, want 18", found)
	}
	if modelIdentityCount != 78 {
		t.Fatalf("bundle carried %d sourced model identities across 19 candidates, want 78", modelIdentityCount)
	}
	for modelID, model := range generated.Providers["apmix"].Policy.Models {
		identity := model.Provenance.Identity
		if identity.Confidence != "high" || identity.ObservedAt != "2026-11-23" || !strings.Contains(identity.Source, "https://business.apmix.ai/") || !strings.Contains(identity.Source, modelID) {
			t.Errorf("production J13 content manifest carries APmix model %q without dated exact-ID identity evidence: %+v", modelID, identity)
		}
	}
	for _, modelID := range []string{"openai/gpt-oss-120b", "openai/gpt-oss-20b"} {
		model, ok := generated.Providers["groqcloud"].Policy.Models[modelID]
		if !ok || model.Eligible == nil || *model.Eligible {
			t.Errorf("bundle catalog does not carry GroqCloud model %q as explicitly disabled", modelID)
		}
	}
	for providerID, modelIDs := range map[string][]string{
		"allgate":      {"gemini-4-argon", "claude-opus-5-5", "gpt-6-sol", "MiniMax-M3", "deepseek-flash"},
		"featherless":  {"mistral-3.5", "qwen3-8b", "minimax-m3"},
		"kilo_gateway": {"anthropic/claude-sonnet-4.6", "openai/gpt-5.4"},
		"indierouter":  {"deepseek-v4-flash", "deepseek-v4.1-flash", "deepseek-v4.1-flash-1m", "gemma-4-26b-a4b-it", "gemma-4-31b-it", "glm-5.3", "glm-5.3-flash", "glm-5.3-flash-1m", "nemotron-3.5-super-vl-120b-a12b", "qwen3.8-27b-fp8", "pplx-decider-v1.1"},
	} {
		for _, modelID := range modelIDs {
			model, ok := generated.Providers[providerID].Policy.Models[modelID]
			if !ok || model.Eligible == nil || *model.Eligible {
				t.Errorf("production bundle catalog does not carry %s model %q as explicitly disabled", providerID, modelID)
			}
		}
	}

	// Go beyond ContentManifest: build a bounded deterministic artifact with
	// the production builder and selected LPBS/Gateway scenarios, then read the
	// catalog and all supplier declarations back from its tar stream. This
	// remains a fixture, not committed-closure or hosted J13 evidence. The
	// fixture uses the production repo contract and the
	// real Gateway catalog/source payloads. The full selected-scenario source
	// tree is large and is independently covered by ContentManifest above; the
	// fixture keeps this cross-scenario HTTP journey small and stable while the
	// archived catalog and declarations remain byte-for-byte production files.
	artifactRoot := t.TempDir()
	copyArtifactFixtureFile := func(rel string) {
		t.Helper()
		data, err := os.ReadFile(filepath.Join(repoRoot, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatalf("read production artifact fixture source %q: %v", rel, err)
		}
		destination := filepath.Join(artifactRoot, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
			t.Fatalf("create production artifact fixture path for %q: %v", rel, err)
		}
		if err := os.WriteFile(destination, data, 0o644); err != nil {
			t.Fatalf("copy production artifact fixture source %q: %v", rel, err)
		}
	}
	copyArtifactFixtureFile(".vrooli/repo-contract.json")
	copyArtifactFixtureFile("scenarios/ai-gateway/.vrooli/service.json")
	copyArtifactFixtureFile("scenarios/landing-page-business-suite/.vrooli/service.json")
	copyArtifactFixtureFile(catalogPath)
	for _, supplier := range suppliers {
		if !supplier.IsDir() && filepath.Ext(supplier.Name()) == ".json" {
			copyArtifactFixtureFile(filepath.ToSlash(filepath.Join("scenarios", "ai-gateway", "config", "suppliers", supplier.Name())))
		}
	}
	artifact, err := bundle.BuildMiniVrooliBundle(artifactRoot, t.TempDir(), manifest)
	if err != nil {
		t.Fatalf("build selected-scenario production J13 bundle artifact: %v", err)
	}
	archive, err := os.Open(artifact.Path)
	if err != nil {
		t.Fatalf("open production bundle artifact: %v", err)
	}
	defer archive.Close()
	artifactHash := sha256.New()
	if _, err := io.Copy(artifactHash, archive); err != nil {
		t.Fatalf("hash production bundle artifact: %v", err)
	}
	if got := hex.EncodeToString(artifactHash.Sum(nil)); got != artifact.Sha256 {
		t.Fatalf("production bundle artifact hash = %s, metadata says %s", got, artifact.Sha256)
	}
	if _, err := archive.Seek(0, io.SeekStart); err != nil {
		t.Fatalf("rewind production bundle artifact: %v", err)
	}
	gz, err := gzip.NewReader(archive)
	if err != nil {
		t.Fatalf("open production bundle gzip stream: %v", err)
	}
	tarReader := tar.NewReader(gz)
	shipped := make(map[string][]byte, found+1)
	wanted := map[string]bool{catalogPath: true}
	for _, supplier := range suppliers {
		if !supplier.IsDir() && filepath.Ext(supplier.Name()) == ".json" {
			wanted[filepath.ToSlash(filepath.Join("scenarios", "ai-gateway", "config", "suppliers", supplier.Name()))] = true
		}
	}
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("read production bundle tar entry: %v", err)
		}
		name := path.Clean(header.Name)
		if !wanted[name] {
			continue
		}
		data, err := io.ReadAll(tarReader)
		if err != nil {
			t.Fatalf("read packaged AC2 file %q: %v", name, err)
		}
		digest := sha256.Sum256(data)
		if got, want := hex.EncodeToString(digest[:]), entryHashes[name]; got != want {
			t.Fatalf("packaged artifact hash for %q = %s, content manifest says %s", name, got, want)
		}
		shipped[name] = data
	}
	if err := gz.Close(); err != nil {
		t.Fatalf("close production bundle gzip stream: %v", err)
	}
	if len(shipped) != len(wanted) {
		t.Fatalf("production bundle artifact carries %d/%d expected catalog/source files", len(shipped), len(wanted))
	}
	if !strings.Contains(string(shipped[catalogPath]), `"providers"`) {
		t.Fatal("production tarball catalog bytes are missing the provider catalog payload")
	}

	// Join the actual catalog bytes from the deterministic archive to the
	// production Gateway HTTP fixture and LPBS authenticated chooser. The
	// nested journey provides isolated database and upstream fixtures only; it
	// does not install routes, approve suppliers, or call a provider.
	catalogFile := filepath.Join(t.TempDir(), "providers.generated.json")
	if err := os.WriteFile(catalogFile, shipped[catalogPath], 0o600); err != nil {
		t.Fatalf("write catalog extracted from production bundle for the LPBS journey: %v", err)
	}
	journey := exec.Command("go", "test", "./journeytest", "-run", "^TestAC2ProductionArtifactCatalogReachesLPBSCreditsChooser$", "-count=1", "-v", "-timeout", "5m")
	journey.Dir = filepath.Join(repoRoot, "scenarios", "landing-page-business-suite", "api")
	journey.Env = append(os.Environ(), "GOWORK=off", "VROOLI_JOURNEY_EMBEDDED_CATALOG_ONLY=1", "VROOLI_JOURNEY_PACKAGED_CATALOG_FILE="+catalogFile)
	var journeyOut, journeyErr bytes.Buffer
	journey.Stdout, journey.Stderr = &journeyOut, &journeyErr
	if err := journey.Run(); err != nil {
		t.Fatalf("artifact-backed Gateway→LPBS chooser journey failed: %v\nstdout:\n%s\nstderr:\n%s", err, journeyOut.String(), journeyErr.String())
	}
	t.Logf("artifact-backed Gateway→LPBS chooser journey: %s", strings.TrimSpace(journeyOut.String()))
}
