# Plan: Scenario Dependency Analyzer — Direct Ollama + Qdrant Integration

## Context / Goal

Today, `scenario-dependency-analyzer` uses `resource-qdrant` as a *combined* embedding+search tool by shelling out to:
- `resource-qdrant embed <text>` (which internally calls Ollama to produce an embedding)
- `resource-qdrant search --collection scenario_embeddings --vector <embedding> --output json`

Goal: keep using the **existing resources** (Ollama + Qdrant), but have `scenario-dependency-analyzer` call them **directly** (HTTP) from Go:
- embeddings via **Ollama** (`resource-ollama` service)
- vector search/upsert via **Qdrant** (`resource-qdrant` service)

This removes reliance on `exec.Command("resource-qdrant", ...)`, improves testability, and makes responsibilities explicit.

Non-goal: route this through `knowledge-observatory` (this plan is specifically “direct resources”).

## Current State (Exact Behavior)

File: `scenarios/scenario-dependency-analyzer/api/internal/app/prediction.go`

Function: `findSimilarScenariosQdrant(description string, existingScenarios []string)`
- Calls `resource-qdrant embed <description>` and treats stdout as the embedding “vector”.
- Calls `resource-qdrant search --collection scenario_embeddings --vector <embedding> --limit 5 --output json`.
- Parses the JSON into:
  - `QdrantSearchResults{ matches: []QdrantMatch }`
  - where `QdrantMatch` contains `scenario_name`, `score`, `resources`, `description`, `metadata`.
- Filters matches where `score > 0.7`.
- Note: `existingScenarios []string` is currently unused.

## Design Principles / “Screaming Architecture” Requirements

- The domain concept is **“Similarity search for proposed scenarios”**, not “call resource-qdrant CLI”.
- Explicit boundaries:
  - Domain service computes similarity output.
  - Adapters talk to Ollama/Qdrant over HTTP.
- Strong testing seams:
  - Domain service tests should not need real Qdrant/Ollama.
  - Adapter tests should use `httptest.Server` to validate request shapes and parse responses deterministically.
- No new dependencies without explicit permission.

## Proposed Architecture

### Domain Service
Create a small domain service responsible for the “find similar scenarios” use case.

Suggested shape:
- `SimilarityService.FindSimilar(ctx, description, limit) ([]QdrantMatch, error)`

### Ports (Interfaces)
Introduce interfaces for clean seams (names illustrative):

- `type Embedder interface { Embed(ctx context.Context, text string) ([]float64, error) }`
- `type VectorIndex interface {`
  - `Search(ctx context.Context, collection string, vector []float64, limit int) ([]VectorMatch, error)`
  - `Upsert(ctx context.Context, collection string, points []VectorPoint) error`
  - `EnsureCollection(ctx context.Context, name string, vectorSize int) error`
  - `}`

Domain returns the existing `QdrantMatch` shape (or a domain-native match shape and map in the handler layer).

### Adapters

**Ollama Embedder (HTTP)**
- Uses:
  - `OLLAMA_URL` default `http://localhost:11434`
  - `OLLAMA_EMBEDDING_MODEL` default aligned with org convention (e.g. `nomic-embed-text`)
- Calls:
  - `POST ${OLLAMA_URL}/api/embeddings` with `{model, prompt}`
- Returns:
  - embedding `[]float64`

**Qdrant Index (HTTP)**
- Uses:
  - `QDRANT_URL` default `http://localhost:6333`
  - optional `QDRANT_API_KEY`
  - `SCENARIO_EMBEDDINGS_COLLECTION` default `scenario_embeddings`
- Calls:
  - `POST ${QDRANT_URL}/collections/${collection}/points/search`
  - `PUT ${QDRANT_URL}/collections/${collection}/points?wait=true` (upsert)
  - `PUT ${QDRANT_URL}/collections/${collection}` (create collection when missing)
- Parses Qdrant native response (`result[]`) into match structs.

## Data Model: “scenario_embeddings” as This Scenario’s Responsibility

Today, `scenario_embeddings` is assumed to exist and be populated, but this scenario doesn’t ensure that.

Make it first-class:

### Indexing Strategy
- Point ID: stable by scenario name (string).
- Vector: embedding of a canonical text:
  - `${display_name}\n${description}` (or `${name}\n${description}` fallback)
- Payload:
  - `scenario_name` (string)
  - `description` (string)
  - `resources` ([]string) derived from declared resources (or detected + declared; choose one and document)
  - `tags` ([]string)
  - `updated_at` (RFC3339 string)

### When to Index
- On “scan/analyze” of scenarios (where metadata is already being loaded and persisted):
  - upsert embeddings for each scenario encountered
  - optionally throttle or batch to avoid overload

### Collection Creation
- If `scenario_embeddings` is missing:
  - create it using vector size = `len(embedding)` from first embed call
  - distance: cosine

## Implementation Steps

1. Add config helpers for `OLLAMA_URL`, `OLLAMA_EMBEDDING_MODEL`, `QDRANT_URL`, `QDRANT_API_KEY`, and `SCENARIO_EMBEDDINGS_COLLECTION`.
2. Add new internal package(s) for ports + adapters + service (keep small, avoid deep nesting).
3. Replace `exec.Command("resource-qdrant", "embed"...` with `Embedder.Embed(...)`.
4. Replace `exec.Command("resource-qdrant", "search"...` with `VectorIndex.Search(...)`.
5. Ensure `scenario_embeddings` creation/upsert happens during analysis/scan (or at least lazily on first use).
6. Keep old CLI path behind a temporary flag for rollback:
   - `USE_RESOURCE_QDRANT_CLI=true` uses the current exec path
   - default to direct HTTP
7. Update docs (`README.md`/PRD/PROBLEMS) to reflect direct resource integration.

## Testing Plan

### Unit tests (fast, hermetic)
- `SimilarityService`:
  - uses fake `Embedder` and fake `VectorIndex`
  - verifies:
    - embedding called with expected text
    - search called with correct collection/limit
    - score filtering `> 0.7` maintained (or adjust and update callers)

### Adapter tests (httptest)
- Ollama adapter:
  - `httptest.Server` asserting request JSON + returning known embedding
  - test error cases: non-200, invalid JSON, missing embedding field
- Qdrant adapter:
  - `httptest.Server` asserting search body shape
  - return Qdrant-style `result[]` payload; ensure correct parsing into matches

### Optional integration test (guarded)
- Only run if env set (e.g. `INTEGRATION=1`) and resources available.

## Risk / Mitigations

- **Embedding model mismatch**: standardize with env var + clear default; ensure collection vector size matches model output.
- **Collection missing**: lazy create on first use; also consider explicit init during startup.
- **Performance**: indexing all scenarios may be expensive; batch + cache embeddings if needed (postponed until observed).
- **Rollback**: keep CLI-based implementation behind a feature flag until stable.

## Acceptance Criteria

- No `exec.Command("resource-qdrant", ...)` used for runtime similarity matching.
- Proposed scenario analysis still returns similar patterns when Qdrant+Ollama are available.
- Unit tests cover domain behavior; adapter tests validate request/response parsing without real resources.
- Scenario remains compatible with the existing Vrooli resource lifecycle (no direct docker/ports assumptions beyond resource URLs).

