# Domains

## Purpose Of This Document

This is the authoritative ownership map for changes to Browser Automation Studio.

## Domain Inventory

| Domain | Responsibility | Owns Data | Primary Archetype | Source Paths |
| --- | --- | --- | --- | --- |
| Workflow authoring | Visual graph creation and editing | Workflow definitions | mutation | `ui/src`, `api/handlers/ai`, `api/handlers/ai_service`, `api/handlers/capture`, `api/handlers/drills`, `api/handlers/entitlement`, `api/handlers/executions`, `api/handlers/export`, `api/handlers/exports_service`, `api/handlers/observability`, `api/handlers/project_files`, `api/handlers/projects`, `api/handlers/recordings`, `api/handlers/replay_config`, `api/handlers/scenarios`, `api/handlers/schedules`, `api/handlers/schema`, `api/handlers/session_profiles`, `api/handlers/uxmetrics`, `api/handlers/vision_navigation`, `api/handlers/workflows` |
| API platform | Shared HTTP/Connect boundary mechanics, conversion, compatibility, and wiring | Transport and conversion policy; no product records | infrastructure | `api/internal/apierror`, `api/internal/compat`, `api/internal/enums`, `api/internal/paths`, `api/internal/protoconv`, `api/internal/scenarioport`, `api/internal/typeconv`, `api/internal/wire` |
| Compilation | Validate/compile workflows into typed actions | Compiled instructions | validation | `api/automation`, `packages/proto/schemas` |
| Browser execution | Execute typed actions and manage sessions | Browser session state | orchestration | `playwright-driver/src` |
| Recording and evidence | Capture artifacts, timeline, replay packages | Recording/evidence metadata | service | `api/recording`, `playwright-driver/src/recording` |
| Persistence | Route and persist domain state | SQLite tables/artifacts | infrastructure | `api/database`, `api/storage` |

## Domain Details

Compilation is the only authority that turns workflow intent into executable actions. Browser execution accepts only typed actions. Evidence records what occurred and does not revise workflow policy.

The API platform is deliberately a separate infrastructure domain. Handlers own product-facing request policy and may depend on it; platform helpers do not own workflow, recording, execution, or project state and must not contain product policy.

## Shared Concepts

Workflow, action, execution, session, artifact, replay package, and recording are shared names; their wire representation is proto.

## Deferred Domains

Commercial packaging and hosted multi-tenant operation are planned; no document should imply they are already implemented.

## Non-Domains

Test utilities, generated code, and lifecycle scripts support domains but do not define product ownership.

## Cross-References

- [Architecture](ARCHITECTURE.md)
- [Flows](FLOWS.md)
- [Data](DATA.md)
- [Seams](../SEAMS.md)
