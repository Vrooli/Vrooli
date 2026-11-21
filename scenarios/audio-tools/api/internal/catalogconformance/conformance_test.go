// Package catalogconformance proves the provider and model catalogs agree.
//
// It deliberately reads the checked-in seeds as JSON instead of importing the
// packages: the catalogs are separate declaration layers, and the test is the
// seam that keeps them from quietly diverging. A provider may reference a model
// ID, and a model may reference a provider ID; neither package should need to
// import the other at runtime to make that true.
package catalogconformance

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"audio-tools/internal/tts"
)

type providerSeed struct {
	Providers []provider `json:"providers"`
}

type provider struct {
	ID           string   `json:"id"`
	AdapterID    string   `json:"adapter_id"`
	Kind         string   `json:"kind"`
	Status       string   `json:"status"`
	Capabilities []string `json:"capabilities"`
	Features     []string `json:"features"`
	EngineIDs    []string `json:"engine_ids"`
	ResourceSlug string   `json:"resource_slug"`
	Actions      []string `json:"actions"`
	Models       []string `json:"models"`
}

type modelSeed struct {
	Models []model `json:"models"`
}

type model struct {
	ID           string   `json:"id"`
	ProviderIDs  []string `json:"provider_ids"`
	EngineIDs    []string `json:"engine_ids"`
	Capability   string   `json:"capability"`
	Languages    []string `json:"languages"`
	License      string   `json:"license"`
	Commercial   string   `json:"commercial_use"`
	LicenseNotes string   `json:"license_notes"`
	Readiness    string   `json:"readiness"`
	Roles        []string `json:"roles"`
	AllowDefault bool     `json:"allow_default"`
}

func loadSeeds(t *testing.T) ([]provider, []model) {
	t.Helper()
	var ps providerSeed
	readJSON(t, filepath.Join("..", "providercatalog", "catalog.seed.json"), &ps)
	var ms modelSeed
	readJSON(t, filepath.Join("..", "modelcatalog", "models.seed.json"), &ms)
	if len(ps.Providers) == 0 {
		t.Fatal("provider catalog is empty")
	}
	if len(ms.Models) == 0 {
		t.Fatal("model catalog is empty")
	}
	return ps.Providers, ms.Models
}

func readJSON(t *testing.T, path string, out any) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if err := json.Unmarshal(raw, out); err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
}

func TestProviderAndModelIDsAreUnique(t *testing.T) {
	providers, models := loadSeeds(t)
	seenP := map[string]bool{}
	for _, p := range providers {
		if p.ID == "" {
			t.Fatal("provider with empty id")
		}
		if seenP[p.ID] {
			t.Fatalf("duplicate provider id %q", p.ID)
		}
		seenP[p.ID] = true
	}
	seenM := map[string]bool{}
	for _, m := range models {
		if m.ID == "" {
			t.Fatal("model with empty id")
		}
		if seenM[m.ID] {
			t.Fatalf("duplicate model id %q", m.ID)
		}
		seenM[m.ID] = true
	}
}

func TestProviderModelReferencesResolve(t *testing.T) {
	providers, models := loadSeeds(t)
	modelIDs := map[string]model{}
	for _, m := range models {
		modelIDs[m.ID] = m
	}
	for _, p := range providers {
		for _, ref := range p.Models {
			m, ok := modelIDs[ref]
			if !ok {
				t.Errorf("provider %q references unknown model %q", p.ID, ref)
				continue
			}
			if !contains(p.Capabilities, m.Capability) {
				t.Errorf("provider %q capabilities %v do not include model %q capability %q", p.ID, p.Capabilities, m.ID, m.Capability)
			}
		}
	}
}

func TestModelProviderReferencesResolve(t *testing.T) {
	providers, models := loadSeeds(t)
	byID := map[string]provider{}
	for _, p := range providers {
		byID[p.ID] = p
	}
	for _, m := range models {
		if len(m.ProviderIDs) == 0 {
			t.Errorf("model %q has no provider_ids", m.ID)
		}
		for _, id := range m.ProviderIDs {
			p, ok := byID[id]
			if !ok {
				t.Errorf("model %q references unknown provider %q", m.ID, id)
				continue
			}
			if !contains(p.Capabilities, m.Capability) {
				t.Errorf("model %q capability %q is not offered by provider %q (%v)", m.ID, m.Capability, id, p.Capabilities)
			}
		}
		if len(m.EngineIDs) > 0 {
			for _, engineID := range m.EngineIDs {
				declared := false
				for _, id := range m.ProviderIDs {
					if p, ok := byID[id]; ok && contains(p.EngineIDs, engineID) {
						declared = true
						break
					}
				}
				if !declared {
					t.Errorf("model %q declares engine %q that none of its providers declare", m.ID, engineID)
				}
			}
		}
	}
}

func TestEngineIDsAreUniqueAndBidirectional(t *testing.T) {
	providers, models := loadSeeds(t)
	engineOwner := map[string]string{}
	for _, p := range providers {
		for _, engineID := range p.EngineIDs {
			if other, dup := engineOwner[engineID]; dup && other != p.ID {
				t.Fatalf("engine %q is declared by both %q and %q", engineID, other, p.ID)
			}
			engineOwner[engineID] = p.ID
		}
	}
	// Every provider engine id must be represented by at least one model that
	// names that provider. An engine with weights but no model record is a
	// provenance hole.
	for engineID, providerID := range engineOwner {
		found := false
		for _, m := range models {
			if contains(m.EngineIDs, engineID) && contains(m.ProviderIDs, providerID) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("engine %q (provider %q) has no model record naming both", engineID, providerID)
		}
	}
}

func TestActiveLocalProvidersDeclareResourceAndMeaning(t *testing.T) {
	providers, _ := loadSeeds(t)
	for _, p := range providers {
		if p.Status != "active" {
			continue
		}
		if len(p.Capabilities) == 0 {
			t.Errorf("active provider %q has no capabilities", p.ID)
		}
		if len(p.Features) == 0 {
			t.Errorf("active provider %q has no features", p.ID)
		}
		if p.Kind == "local_resource" {
			if p.ResourceSlug == "" {
				t.Errorf("active local provider %q has no resource_slug", p.ID)
			}
			if len(p.Actions) == 0 {
				t.Errorf("active local provider %q has no lifecycle actions", p.ID)
			}
		}
	}
}

func TestDefaultRolesRespectLicenceAndReadiness(t *testing.T) {
	_, models := loadSeeds(t)
	for _, m := range models {
		if !contains(m.Roles, "default") {
			continue
		}
		// A default role normally requires explicit commercial permission.
		// allow_default is the auditable exception: a runtime-primary model can
		// carry the default role before its licence is verified, and
		// CommercialDefaultFor still refuses to select it. The exception is only
		// legal with license_notes naming the uncertainty.
		if m.Commercial != "yes" && !m.AllowDefault {
			t.Errorf("model %q is a default but commercial_use=%q (license %q) without allow_default", m.ID, m.Commercial, m.License)
		}
		if m.Commercial != "yes" && m.AllowDefault && m.LicenseNotes == "" {
			t.Errorf("model %q is a default with commercial_use=%q and allow_default but no license_notes", m.ID, m.Commercial)
		}
		switch m.Readiness {
		case "available":
		case "candidate", "experimental", "watch":
			if !m.AllowDefault {
				t.Errorf("model %q is %s and a default without allow_default", m.ID, m.Readiness)
			}
		default:
			t.Errorf("model %q has unknown readiness %q", m.ID, m.Readiness)
		}
	}
}

func TestModelsDeclareProvenance(t *testing.T) {
	_, models := loadSeeds(t)
	for _, m := range models {
		if m.License == "" {
			t.Errorf("model %q has no license", m.ID)
		}
		if m.Commercial == "" {
			t.Errorf("model %q has no commercial_use", m.ID)
		}
		if len(m.Languages) == 0 {
			t.Errorf("model %q has no languages", m.ID)
		}
	}
}

func TestProviderModelCapabilityCoverage(t *testing.T) {
	providers, models := loadSeeds(t)
	// Every model must be referenced by at least one provider that offers its
	// capability. This catches an orphaned model (no way to run it) and a
	// provider that advertises a capability with no weights.
	for _, m := range models {
		referenced := false
		for _, p := range providers {
			if contains(p.Models, m.ID) && contains(p.Capabilities, m.Capability) {
				referenced = true
				break
			}
		}
		if !referenced {
			t.Errorf("model %q (%s) is not referenced by any provider offering that capability", m.ID, m.Capability)
		}
	}
}

func contains(haystack []string, needle string) bool {
	for _, v := range haystack {
		if v == needle {
			return true
		}
	}
	return false
}

// TestEngineOwnersAreStable is a light determinism guard: the engine-owner map
// is used above, and a map iteration order must never influence a failure
// message that agents compare across runs.
func TestEngineOwnersAreStable(t *testing.T) {
	providers, _ := loadSeeds(t)
	owners := make([]string, 0)
	for _, p := range providers {
		for _, engineID := range p.EngineIDs {
			owners = append(owners, engineID+"="+p.ID)
		}
	}
	sort.Strings(owners)
	if len(owners) == 0 {
		t.Fatal("no engine declarations found")
	}
}

// sttEngineManifest is the subset of api/internal/sttengine/manifest.json this
// gate needs. Declaring it locally keeps the cross-catalog test independent of
// the sttengine package while still reading the checked-in source of truth.
type sttEngineManifest struct {
	Engines []struct {
		ID       string `json:"id"`
		Resource string `json:"resource"`
	} `json:"engines"`
}

func loadSTTEngines(t *testing.T) map[string]string {
	t.Helper()
	var manifest sttEngineManifest
	readJSON(t, filepath.Join("..", "sttengine", "manifest.json"), &manifest)
	if len(manifest.Engines) == 0 {
		t.Fatal("sttengine manifest declares no engines")
	}
	out := make(map[string]string, len(manifest.Engines))
	for _, engine := range manifest.Engines {
		if engine.ID == "" {
			t.Fatal("sttengine manifest has an engine with no id")
		}
		if _, dup := out[engine.ID]; dup {
			t.Fatalf("sttengine manifest declares engine %q twice", engine.ID)
		}
		out[engine.ID] = engine.Resource
	}
	return out
}

func TestProviderEnginesMatchSTTEngineManifest(t *testing.T) {
	providers, _ := loadSeeds(t)
	manifest := loadSTTEngines(t)

	owned := map[string]string{}
	for _, p := range providers {
		if p.Status != "active" {
			continue
		}
		for _, engineID := range p.EngineIDs {
			if engineID == "" {
				t.Errorf("provider %q declares an empty engine id", p.ID)
				continue
			}
			if other, dup := owned[engineID]; dup && other != p.ID {
				t.Errorf("engine %q is declared by both %q and %q", engineID, other, p.ID)
			}
			owned[engineID] = p.ID
			resource, ok := manifest[engineID]
			if !ok {
				t.Errorf("provider %q declares engine %q that sttengine does not ship", p.ID, engineID)
				continue
			}
			if p.ResourceSlug != "" && p.ResourceSlug != resource {
				t.Errorf("provider %q maps engine %q to resource %q but the manifest says %q",
					p.ID, engineID, p.ResourceSlug, resource)
			}
		}
	}

	for engineID := range manifest {
		if _, ok := owned[engineID]; !ok {
			t.Errorf("sttengine ships engine %q but no active provider owns it", engineID)
		}
	}
}

func TestModelEngineIDsAreShippedBySTTEngineManifest(t *testing.T) {
	_, models := loadSeeds(t)
	manifest := loadSTTEngines(t)
	for _, m := range models {
		for _, engineID := range m.EngineIDs {
			if _, ok := manifest[engineID]; !ok {
				t.Errorf("model %q declares engine %q that sttengine does not ship", m.ID, engineID)
			}
		}
	}
}

// TestActiveLocalSTTModelsNameAnEngine closes the "model with weights but no
// selectable stream path" hole: an active local STT adapter must resolve to at
// least one engine that the same model's providers own.
func TestActiveLocalSTTModelsNameAnEngine(t *testing.T) {
	providers, models := loadSeeds(t)
	byID := map[string]provider{}
	for _, p := range providers {
		byID[p.ID] = p
	}
	for _, m := range models {
		if m.Capability != "stt" {
			continue
		}
		for _, providerID := range m.ProviderIDs {
			p, ok := byID[providerID]
			if !ok || p.Kind != "local_resource" || p.Status != "active" {
				continue
			}
			if len(m.EngineIDs) == 0 {
				t.Errorf("model %q is served by active local provider %q but declares no engine id", m.ID, providerID)
				continue
			}
			owned := false
			for _, engineID := range m.EngineIDs {
				if contains(p.EngineIDs, engineID) {
					owned = true
					break
				}
			}
			if !owned {
				t.Errorf("model %q engine ids %v are not owned by provider %q (%v)",
					m.ID, m.EngineIDs, providerID, p.EngineIDs)
			}
		}
	}
}

// TestTTSAdapterMappingsMatchProviderCatalog ties the code-side voice catalog to
// the data-side provider catalog. A new TTS provider adapter must declare a
// canonical-voice mapping, and a mapping cannot name an adapter that no
// provider declares. The same test runs VerifyCatalog so a mapping cannot omit
// a canonical voice.
func TestTTSAdapterMappingsMatchProviderCatalog(t *testing.T) {
	providers, _ := loadSeeds(t)
	declared := map[string]bool{}
	for _, p := range providers {
		if p.AdapterID != "" {
			declared[p.AdapterID] = true
		}
	}
	mapped := map[string]bool{}
	for _, entry := range tts.CanonicalAdapterMappings {
		mapped[entry.TierProvider] = true
	}
	for id := range declared {
		if !mapped[id] {
			t.Errorf("provider adapter %q has no canonical-voice mapping in internal/tts", id)
		}
	}
	for id := range mapped {
		if !declared[id] {
			t.Errorf("voice mapping %q is not a declared provider adapter_id", id)
		}
	}
	if err := tts.VerifyCatalog(tts.CanonicalAdapterMappings); err != nil {
		t.Fatalf("voice catalog incomplete: %v", err)
	}
}
