# Requirement Traceability

*Generated:* 2025-10-29T02:16:21.931Z

## BAS-FUNC-001 — Persist visual workflows with nodes/edges and folder hierarchy

- **Status:** in_progress
- **Criticality:** P0
- **Category:** foundation

| Type | Reference | Status | Exists | Notes |
| --- | --- | --- | --- | --- |
| test | api/browserless/compiler_test.go | implemented | ⚠️ | Validates compile ordering, cycle detection, and unsupported node handling. |
| test | test/phases/test-unit.sh | not_implemented | ✅ | Add unit coverage for `database.Workflow` persistence & compiler validation. |
| automation | workflows/ui-validation/structure-check.json | not_implemented | ⚠️ | Planned Browserless automation to verify workflow CRUD via API. |

## BAS-FUNC-002 — Stream execution telemetry (screenshots, progress, logs) while runs are active

- **Status:** in_progress
- **Criticality:** P0
- **Category:** execution

| Type | Reference | Status | Exists | Notes |
| --- | --- | --- | --- | --- |
| test | api/browserless/client_test.go | implemented | ✅ | Exercises telemetry persistence, heartbeat emission, retry/backoff attempt metadata, assert artifact storage, and per-step highlight metadata; extend with WebSocket contract tests. |
| test | api/browserless/runtime/session_test.go | not_implemented | ⚠️ | Session manager unit tests emitting mock telemetry. |
| test | ui/tests/executionEventProcessor.heartbeat.test.mjs | implemented | ✅ | Node-based unit test ensuring heartbeat events update progress and heartbeat metadata in the UI store; extend with UI heartbeat state rendering snapshot. |
| test | ui/tests/executionEventProcessor.assertion.test.mjs | implemented | ✅ | Confirms assertion payloads generate readable success/failure logs in the execution stream processor. |
| automation | test/playbooks/executions/telemetry-smoke.json | planned | ⚠️ | JSON workflow export pending; will assert telemetry artefacts once migrated. |
| manual | ui/src/components/ExecutionViewer.tsx | implemented | ✅ | Execution viewer flags heartbeat states (awaiting/delayed/stalled) with contextual messaging. |
| manual | cli/browser-automation-studio | implemented | ✅ | CLI execution watch emits heartbeat lag/stall warnings and recovery notices. |
| automation | test/playbooks/executions/heartbeat-stall.json | planned | ⚠️ | JSON workflow export pending; will validate stalled heartbeat handling. |

## BAS-FUNC-003 — Persist normalized execution artifacts for replay & video rendering

- **Status:** in_progress
- **Criticality:** P1
- **Category:** replay

| Type | Reference | Status | Exists | Notes |
| --- | --- | --- | --- | --- |
| test | api/database/repository_test.go | in_progress | ✅ | Covers execution_steps/execution_artifacts persistence; add replay schema contract tests as renderer hardens. |
| test | api/browserless/client_test.go | implemented | ✅ | Verifies timeline_frame artifact creation, cursor trail metadata propagation, highlight/mask metadata, and retry history fields without live Browserless. |
| test | api/services/exporter_test.go | implemented | ✅ | Ensures replay export packages include frame/asset metadata, transition hints, and resiliency fields for downstream renderers. |
| test | api/services/timeline_test.go | implemented | ✅ | Ensures execution timeline aggregation surfaces screenshot/highlight metadata for replay consumers. |
| automation | test/playbooks/replay/render-check.json | planned | ⚠️ | JSON workflow export pending; will validate replay export once migrated. |
| manual | cli/browser-automation-studio execution render | implemented | ⚠️ | Generates stylised HTML replay packages from export metadata for operator review. |

## BAS-FUNC-004 — AI-generated workflows validated against the execution plan

- **Status:** pending
- **Criticality:** P0
- **Category:** ai-integration

| Type | Reference | Status | Exists | Notes |
| --- | --- | --- | --- | --- |
| test | api/services/workflow_service_ai_test.go | not_implemented | ⚠️ | Add deterministic tests for AI prompt normalization + compiler validation. |
| automation | test/playbooks/ai/generated-smoke.json | planned | ⚠️ | BAS workflow executing AI-generated flows end-to-end. |
