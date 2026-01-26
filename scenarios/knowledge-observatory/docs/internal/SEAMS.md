# Seams & Architecture Boundaries

## Last Updated
2026-01-26

## Integration Seams
- **Vector store seam**: `ports.VectorStore` enables Qdrant substitution for testing. [CODE: api/internal/ports/ports.go]
- **Embedder seam**: `ports.Embedder` isolates embedding provider. [CODE: api/internal/ports/ports.go]
- **Metadata seam**: `ports.MetadataStore` isolates Postgres-backed metadata. [CODE: api/internal/ports/ports.go]
- **Job store seam**: `ports.JobStore` isolates ingest job queue. [CODE: api/internal/ports/ports.go]

## Responsibility Zones
- **Entry/presentation**: HTTP handlers and UI pages. [CODE: api/server.go] [CODE: ui/src/surfaces/dashboard/DashboardPage.tsx]
- **Coordination/orchestration**: server wiring + service construction. [CODE: api/server.go]
- **Domain rules**: ingest/search/graph services, metric calculations. [CODE: api/internal/services/ingest/service.go] [CODE: api/internal/services/search/service.go] [CODE: api/internal/services/graph/service.go]
- **Integrations/infrastructure**: Qdrant/Ollama/Postgres adapters. [CODE: api/internal/adapters/vectorstore/qdrant.go] [CODE: api/internal/adapters/embedder/ollama.go] [CODE: api/internal/adapters/metadatastore/postgres.go]
- **Cross-cutting concerns**: CORS, logging, health checks. [CODE: api/server.go]

## Decision Points
- **Visibility normalization**: enforced values (`private/shared/global`). [CODE: api/knowledge_schema.go]
- **Search defaults**: limit/threshold normalization. [CODE: api/search.go]
- **Chunking strategy**: chunk size/overlap constraints. [CODE: api/document_ingest.go] [CODE: api/internal/services/ingest/chunking.go]

## Change Axes
- Primary change axis: API contracts for ingest/search/graph.
- Secondary change axis: UI surface composition and data presentation.

## Observability Surface
- Health endpoint for dependency visibility. [CODE: api/server.go]
- Metrics endpoint for quality telemetry. [CODE: api/metrics.go]

## Architecture Clarity Notes
- API wiring is centralized in `api/server.go`.
- Domain services are isolated under `api/internal/services`.

## Exploration Log
- 2025-12-16: Split API bootstrap from server wiring; documented in audit. [DOC: docs/SCREAMING_ARCHITECTURE_AUDIT.md]
