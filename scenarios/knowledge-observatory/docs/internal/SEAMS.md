# Seams & Architecture Boundaries

## Last Updated
2026-01-26

## Integration Seams
- **Vector store seam**: `ports.VectorStore` enables Qdrant substitution for testing. [CODE: api/internal/ports/ports.go]
- **Embedder seam**: `ports.Embedder` isolates embedding provider. [CODE: api/internal/ports/ports.go]
- **Metadata seam**: `ports.MetadataStore` isolates Postgres-backed metadata. [CODE: api/internal/ports/ports.go]
- **Job store seam**: `ports.JobStore` isolates ingest job queue. [CODE: api/internal/ports/ports.go]
- **Scenario filesystem seam**: doc health service encapsulates filesystem access to scenario docs. [CODE: api/internal/services/dochealth/service.go]
- **Documentation search seam**: docsearch service owns file/text/unified search over documentation roots. [CODE: api/internal/services/docsearch/service.go]
- **Documentation explorer seam**: explorer service builds scenario doc trees with warnings. [CODE: api/internal/services/explorer/tree.go]
- **Documentation viewer seam**: viewer service owns safe document loading + reset operations. [CODE: api/internal/services/viewer/service.go]
- **Documentation deep search seam**: deepsearch service orchestrates agent-powered doc discovery + job tracking. [CODE: api/internal/services/deepsearch/service.go]
- **Deep search job store seam**: Postgres adapter persists deep search job state. [CODE: api/internal/adapters/deepsearchstore/postgres.go]
- **Agent-manager seam**: deep search agent client isolates agent-manager HTTP calls and supports fixed base URLs for tests. [CODE: api/internal/adapters/agentmanager/deepsearch_client.go] [CODE: api/internal/adapters/agentmanager/client.go]
- **Prompt-manager seam**: prompt-manager client retrieves skill content for deep search. [CODE: api/internal/adapters/promptmanager/client.go]

## Responsibility Zones
- **Entry/presentation**: HTTP handlers and UI pages. [CODE: api/server.go] [CODE: ui/src/surfaces/dashboard/DashboardPage.tsx]
- **Coordination/orchestration**: server wiring + service construction. [CODE: api/server.go]
- **Domain rules**: ingest/search/graph services, metric calculations. [CODE: api/internal/services/ingest/service.go] [CODE: api/internal/services/search/service.go] [CODE: api/internal/services/graph/service.go]
- **Documentation standards**: docschema package owns documentation layout validation and reset rules. [CODE: api/internal/docschema/types.go] [CODE: api/internal/docschema/validation.go]
- **Documentation health API**: handlers and service for scenario validation/reset. [CODE: api/docs_health.go] [CODE: api/internal/services/dochealth/service.go]
- **Documentation search API**: handlers + docsearch service for file/text/unified search. [CODE: api/docs_search.go] [CODE: api/internal/services/docsearch/service.go]
- **Documentation explorer API**: handlers + explorer service for scenario listing and doc tree. [CODE: api/docs_explorer.go] [CODE: api/docs_search.go] [CODE: api/internal/services/explorer/tree.go]
- **Documentation viewer API**: handlers + viewer service for content and reset. [CODE: api/docs_viewer.go] [CODE: api/internal/services/viewer/content.go]
- **Documentation deep search API**: handlers + deep search service for agent-backed search. [CODE: api/docs_deep_search.go] [CODE: api/internal/services/deepsearch/service.go]
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
- 2025-12-16: Split API bootstrap from server wiring; documented in audit. [DOC: docs/internal/SCREAMING_ARCHITECTURE_AUDIT.md]
