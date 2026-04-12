package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
	repocontract "github.com/vrooli/repo-contract-go"
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
	return parseServiceJSON(data, "", slug)
}

func parseServiceJSON(data []byte, repoRoot, slug string) (*ScenarioEnvelopeResponse, error) {
	var svc serviceJSON
	if err := json.Unmarshal(data, &svc); err != nil {
		return nil, fmt.Errorf("invalid service.json: %w", err)
	}

	tags := svc.Service.Tags
	if tags == nil {
		tags = []string{}
	}

	scenarioPath := fmt.Sprintf("scenarios/%s", slug)
	if strings.TrimSpace(repoRoot) != "" {
		if resolvedPath, err := resolveScenarioPathRelative(repoRoot, slug); err == nil {
			scenarioPath = resolvedPath
		}
	}

	return &ScenarioEnvelopeResponse{
		Name:         svc.Service.Name,
		DisplayName:  svc.Service.DisplayName,
		Description:  svc.Service.Description,
		Path:         scenarioPath,
		Tags:         tags,
		Dependencies: extractDependencies(svc.Dependencies),
		Lifecycle:    extractLifecycle(svc.Lifecycle, slug),
	}, nil
}

// extractDependencies converts serviceJSONDependencies to ScenarioEnvelopeDependencies.
func extractDependencies(deps serviceJSONDependencies) ScenarioEnvelopeDependencies {
	scenarioDeps := make(map[string]string, len(deps.Scenarios))
	for name, dep := range deps.Scenarios {
		scenarioDeps[name] = dep.Description
	}
	resourceDeps := make(map[string]string, len(deps.Resources))
	for name, dep := range deps.Resources {
		resourceDeps[name] = dep.Description
	}
	return ScenarioEnvelopeDependencies{
		Scenarios: scenarioDeps,
		Resources: resourceDeps,
	}
}

// extractLifecycle extracts test and build commands from the lifecycle configuration.
func extractLifecycle(lc serviceJSONLifecycle, slug string) ScenarioEnvelopeLifecycle {
	testCmd := fmt.Sprintf("vrooli scenario test %s", slug)
	if lc.Test != nil && len(lc.Test.Steps) > 0 {
		if last := lc.Test.Steps[len(lc.Test.Steps)-1]; last.Run != "" {
			testCmd = last.Run
		}
	}

	var buildCmd string
	if lc.Setup != nil {
		for _, step := range lc.Setup.Steps {
			if containsBuild(step.Name) && step.Run != "" {
				buildCmd = step.Run
				break
			}
		}
	}

	return ScenarioEnvelopeLifecycle{
		TestCommand:  testCmd,
		BuildCommand: buildCmd,
	}
}

// containsBuild checks if a step name indicates a build step (case-insensitive substring match).
func containsBuild(name string) bool {
	return strings.Contains(strings.ToLower(name), "build")
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

	serviceJSONPath, err := repocontract.ResolveScenarioFile(repoRoot, slug, "service")
	if err != nil {
		http.Error(w, "failed to resolve scenario metadata path", http.StatusInternalServerError)
		return
	}
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

	envelope, err := parseServiceJSON(data, repoRoot, slug)
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
