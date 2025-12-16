---
title: "Architecture Audit"
description: "Code organization and architectural alignment with the domain mental model"
category: "technical"
order: 1
audience: ["developers"]
---

# Screaming Architecture Audit: knowledge-observatory

**Date**: 2025-12-16  
**Auditor**: GPT-5.2 (Codex CLI)  
**Status**: IMPROVING — core surfaces are clear; key integrations and entrypoints now better-aligned

---

## Executive Summary

Knowledge Observatory’s domain is crisp: **observe**, **query**, and **health-check** Vrooli’s semantic memory (Qdrant) with optional embedding support (Ollama) and metadata persistence (Postgres). The scenario already presents three clear “surfaces”:

- **API**: semantic search + health/metrics
- **UI**: mission-control dashboard for health + panels
- **CLI**: lightweight operator interface

The main architectural drift was in the API: bootstrap/runtime wiring was mixed with domain handlers, and low-level integrations (resource CLI, Qdrant, Ollama) were pulling configuration directly from the environment at call-sites.

This audit implements a small, behavior-preserving refactor so the API structure “screams” its intent: **entrypoint → server wiring → domain handlers** with explicit integration boundaries.

---

## 1. Domain & Mental Model

### What this scenario exists to do
Provide a **consciousness monitor** over semantic memory:

1. **Semantic Search**: query embeddings → vector search → results
2. **Knowledge Health**: summarize collections and quality signals
3. **Operator Surfaces**: API/CLI/UI for humans and agents to introspect knowledge

### Primary actors
- **Operator / Agent**: runs queries, monitors health
- **Knowledge Base (Qdrant)**: collections + points + payloads
- **Embedding Provider (Ollama)**: query → embedding vector
- **Metadata Store (Postgres)**: system dependency + future metrics persistence

### Core flow (today)
`query → embedding → qdrant search → normalized results`

---

## 2. Current Logical Architecture (as implemented)

### API (Go)
- **Entrypoint**: lifecycle-managed process guard + startup
- **Server**: config, DB connection, router, graceful shutdown
- **Handlers**
  - `/health` (infra readiness: db connectivity)
  - `/api/v1/knowledge/search` (semantic search)
  - `/api/v1/knowledge/health` (collection-level metrics summary)
- **Integrations**
  - `resource-qdrant collections` (collection discovery)
  - Qdrant REST API (per-collection vector search)
  - Ollama embeddings API (query embedding generation)

### UI (React)
- A single dashboard with modal panels; API health polling via `react-query`.

### CLI (bash)
- Operator-friendly wrapper with local config and API autodiscovery.

---

## 3. Physical Structure & “Screaming” Assessment

### What already screams
- Scenario root clearly separates surfaces: `api/`, `ui/`, `cli/`, `test/`, `requirements/`.
- The UI components are named for domain panels (`SearchPanel`, `MetricsPanel`).

### Where it didn’t (before this audit)
- API `main.go` combined lifecycle boot, server wiring, handlers, and low-level configuration access.
- Environment configuration was “implicit dependency injection” scattered across integration methods.
- `resource-qdrant` invocations could block indefinitely when request contexts lacked deadlines.

---

## 4. Changes Implemented (this audit)

### API structure now reflects intent
- `api/main.go` is now a minimal bootstrap (lifecycle guard + start).
- `api/server.go` holds server wiring, config defaults, shared helpers, and integration wrappers.

### Integration boundaries are more explicit
- Centralized fallbacks for `QDRANT_URL`, `OLLAMA_URL`, `OLLAMA_EMBEDDING_MODEL`, and `RESOURCE_QDRANT_CLI`.
- Centralized `resource-qdrant` execution with a default timeout to avoid hanging calls.
- Shared error response helper moved to the server layer.

These changes preserve runtime behavior while making dependencies easier to trace.

---

## 5. Remaining Architectural Gaps (not changed here)

- **Knowledge Graph capability is documented but not implemented** (README/PRD vs API/UI reality). This is a product-surface gap, not just structure.
- **Search result merging has a TODO** (cross-collection sorting/limiting is not global).
- **Collection health metrics are currently placeholder data** (`getCollectionHealth`), which limits how much “observatory” the API can truly provide.

---

## 6. Recommended Next Steps (safe, incremental)

- Add `api/internal/...` packages only when a second domain emerges (e.g., graph generation) to avoid premature abstraction.
- When implementing the Knowledge Graph, create a clear domain module boundary (e.g., `graph/` with request model, builder, and Qdrant traversal integration).
- Replace placeholder collection health with real Qdrant-derived stats in a dedicated integration file (keep handler thin).

