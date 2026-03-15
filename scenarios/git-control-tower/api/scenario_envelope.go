package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/gorilla/mux"
)

// =============================================================================
// Scenario Envelope — Enriched metadata from service.json for agent orientation
// =============================================================================

// ScenarioEnvelopeResponse is the JSON payload returned by GET /api/v1/scenarios/{slug}/envelope.
// It contains the subset of service.json data that an AI agent needs to orient itself
// within a scenario: identity, dependencies, and lifecycle commands.
type ScenarioEnvelopeResponse struct {
	Name         string                       `json:"name"`
	DisplayName  string                       `json:"displayName"`
	Description  string                       `json:"description"`
	Path         string                       `json:"path"`
	Tags         []string                     `json:"tags"`
	Dependencies ScenarioEnvelopeDependencies `json:"dependencies"`
	Lifecycle    ScenarioEnvelopeLifecycle    `json:"lifecycle"`
}

// ScenarioEnvelopeDependencies lists the scenario's resource and scenario dependencies
// as name→description maps, so the agent knows what tools are available.
type ScenarioEnvelopeDependencies struct {
	Scenarios map[string]string `json:"scenarios"`
	Resources map[string]string `json:"resources"`
}

// ScenarioEnvelopeLifecycle contains the key lifecycle commands extracted from service.json,
// so the agent can build, test, and verify its work.
type ScenarioEnvelopeLifecycle struct {
	TestCommand  string `json:"testCommand,omitempty"`
	BuildCommand string `json:"buildCommand,omitempty"`
}

// envelopeCacheEntry holds a parsed envelope response and its expiry time.
type envelopeCacheEntry struct {
	data    *ScenarioEnvelopeResponse
	expires time.Time
}

// EnvelopeCache provides thread-safe, TTL-based caching for parsed service.json data.
// This avoids redundant disk reads since service.json changes rarely.
type EnvelopeCache struct {
	mu      sync.Mutex
	entries map[string]envelopeCacheEntry
	ttl     time.Duration
}

// NewEnvelopeCache creates a cache with the given TTL.
func NewEnvelopeCache(ttl time.Duration) *EnvelopeCache {
	return &EnvelopeCache{
		entries: make(map[string]envelopeCacheEntry),
		ttl:     ttl,
	}
}

// Get returns a cached entry if it exists and hasn't expired.
func (c *EnvelopeCache) Get(slug string) (*ScenarioEnvelopeResponse, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	entry, ok := c.entries[slug]
	if !ok || time.Now().After(entry.expires) {
		return nil, false
	}
	return entry.data, true
}

// Set stores an entry with the configured TTL.
func (c *EnvelopeCache) Set(slug string, data *ScenarioEnvelopeResponse) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[slug] = envelopeCacheEntry{
		data:    data,
		expires: time.Now().Add(c.ttl),
	}
}

// =============================================================================
// service.json parsing types (internal — mirrors the on-disk structure)
// =============================================================================

// serviceJSON is the top-level structure of a scenario's .vrooli/service.json.
type serviceJSON struct {
	Service      serviceJSONMeta         `json:"service"`
	Dependencies serviceJSONDependencies `json:"dependencies"`
	Lifecycle    serviceJSONLifecycle    `json:"lifecycle"`
}

type serviceJSONMeta struct {
	Name        string   `json:"name"`
	DisplayName string   `json:"displayName"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
}

type serviceJSONDependencies struct {
	Scenarios map[string]serviceJSONDep `json:"scenarios"`
	Resources map[string]serviceJSONDep `json:"resources"`
}

type serviceJSONDep struct {
	Description string `json:"description"`
}

type serviceJSONLifecycle struct {
	Test  *serviceJSONLifecyclePhase `json:"test"`
	Setup *serviceJSONLifecyclePhase `json:"setup"`
}

type serviceJSONLifecyclePhase struct {
	Steps []serviceJSONStep `json:"steps"`
}

type serviceJSONStep struct {
	Name string `json:"name"`
	Run  string `json:"run"`
}

// =============================================================================
// Parsing logic (exported for testing)
// =============================================================================

// ParseServiceJSON reads raw service.json bytes and produces a ScenarioEnvelopeResponse.
// The slug is used to build the relative path and as a fallback for the test command.
func ParseServiceJSON(data []byte, slug string) (*ScenarioEnvelopeResponse, error) {
	var svc serviceJSON
	if err := json.Unmarshal(data, &svc); err != nil {
		return nil, fmt.Errorf("invalid service.json: %w", err)
	}

	// Ensure non-nil slices/maps for clean JSON output.
	tags := svc.Service.Tags
	if tags == nil {
		tags = []string{}
	}

	scenarioDeps := make(map[string]string, len(svc.Dependencies.Scenarios))
	for name, dep := range svc.Dependencies.Scenarios {
		scenarioDeps[name] = dep.Description
	}

	resourceDeps := make(map[string]string, len(svc.Dependencies.Resources))
	for name, dep := range svc.Dependencies.Resources {
		resourceDeps[name] = dep.Description
	}

	// Extract test command: use the last step in lifecycle.test (typically the actual test runner).
	testCmd := fmt.Sprintf("vrooli scenario test %s", slug)
	if svc.Lifecycle.Test != nil && len(svc.Lifecycle.Test.Steps) > 0 {
		last := svc.Lifecycle.Test.Steps[len(svc.Lifecycle.Test.Steps)-1]
		if last.Run != "" {
			testCmd = last.Run
		}
	}

	// Extract build command: first setup step whose name contains "build".
	var buildCmd string
	if svc.Lifecycle.Setup != nil {
		for _, step := range svc.Lifecycle.Setup.Steps {
			if containsBuild(step.Name) && step.Run != "" {
				buildCmd = step.Run
				break
			}
		}
	}

	return &ScenarioEnvelopeResponse{
		Name:        svc.Service.Name,
		DisplayName: svc.Service.DisplayName,
		Description: svc.Service.Description,
		Path:        fmt.Sprintf("scenarios/%s", slug),
		Tags:        tags,
		Dependencies: ScenarioEnvelopeDependencies{
			Scenarios: scenarioDeps,
			Resources: resourceDeps,
		},
		Lifecycle: ScenarioEnvelopeLifecycle{
			TestCommand:  testCmd,
			BuildCommand: buildCmd,
		},
	}, nil
}

// containsBuild checks if a step name indicates a build step (case-insensitive substring match).
func containsBuild(name string) bool {
	for i := 0; i+5 <= len(name); i++ {
		if (name[i] == 'b' || name[i] == 'B') &&
			(name[i+1] == 'u' || name[i+1] == 'U') &&
			(name[i+2] == 'i' || name[i+2] == 'I') &&
			(name[i+3] == 'l' || name[i+3] == 'L') &&
			(name[i+4] == 'd' || name[i+4] == 'D') {
			return true
		}
	}
	return false
}

// =============================================================================
// HTTP handler
// =============================================================================

// handleScenarioEnvelope serves enriched scenario metadata for agent orientation.
// It reads and caches the scenario's .vrooli/service.json file.
func (s *Server) handleScenarioEnvelope(w http.ResponseWriter, r *http.Request) {
	slug := mux.Vars(r)["slug"]
	if slug == "" {
		http.Error(w, "missing scenario slug", http.StatusBadRequest)
		return
	}

	// Check cache first.
	if cached, ok := s.envelopeCache.Get(slug); ok {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(cached); err != nil {
			log.Printf("ERROR: encoding cached envelope for %q: %v", slug, err)
		}
		return
	}

	// Resolve the service.json path via the repo root.
	repoRoot := s.git.ResolveRepoRoot(r.Context())
	if repoRoot == "" {
		http.Error(w, "unable to determine repository root", http.StatusInternalServerError)
		return
	}

	serviceJSONPath := filepath.Join(repoRoot, "scenarios", slug, ".vrooli", "service.json")
	data, err := os.ReadFile(serviceJSONPath)
	if err != nil {
		if os.IsNotExist(err) {
			http.Error(w, fmt.Sprintf("scenario %q not found (no service.json)", slug), http.StatusNotFound)
			return
		}
		log.Printf("ERROR: reading service.json for %q: %v", slug, err)
		http.Error(w, "failed to read scenario metadata", http.StatusInternalServerError)
		return
	}

	envelope, err := ParseServiceJSON(data, slug)
	if err != nil {
		log.Printf("ERROR: parsing service.json for %q: %v", slug, err)
		http.Error(w, "failed to parse scenario metadata", http.StatusInternalServerError)
		return
	}

	// Cache for future requests.
	s.envelopeCache.Set(slug, envelope)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(envelope); err != nil {
		log.Printf("ERROR: encoding envelope for %q: %v", slug, err)
	}
}
