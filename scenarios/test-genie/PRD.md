# Product Requirements Document (PRD)

> **Template Version**: 2.0
> **Canonical Reference**: `/scenarios/business-health/docs/reference/canonical-prd-template.md`
> **Validation**: Enforced by `scenario-auditor`
> **Policy**: The PRD is the product contract; requirement modules and tests should trace back to the targets defined here.

## 🎯 Overview

- **Purpose**: Test Genie is Vrooli's scenario-local testing command center. It discovers scenarios, orchestrates phased test execution, stores queue and execution history, and provides AI-assisted generation/fix workflows for expanding test coverage.
- **Primary users/verticals**:
  - scenario authors validating local apps before deployment
  - operators auditing queue health, execution history, and requirement coverage
  - AI agents generating or repairing tests through controlled delegation
- **Deployment surfaces**: Go API, React UI, Go CLI
- **Value promise**:
  - Replace ad hoc shell-based testing with a deterministic Go-native orchestration layer
  - Make test execution legible through queue telemetry, execution history, and per-phase artifacts
  - Provide a single control surface for local test generation, execution, fixing, and requirement sync
  - Raise the baseline quality bar for every scenario that ships through Vrooli

## 🎯 Operational Targets

### 🔴 P0 – Must ship for viability

- [x] OT-P0-001 | Internal Test Orchestrator | Run Test Genie phases from scenario-local Go packages with explicit presets, phase toggles, artifacts, and requirement sync decisions.
- [x] OT-P0-002 | AI Suite Generation | Support AI-assisted suite generation and repair workflows through CLI, API, and the operator UI without requiring manual shell orchestration.

### 🟠 P1 – Should have post-launch

- [ ] OT-P1-003 | Vault & Coverage UX | Surface requirement coverage, execution insights, and vault/test-gap guidance directly in the UI so operators can steer improvements without reading raw logs.

### 🟢 P2 – Future / expansion

- [ ] No committed P2 targets yet. Add future work here only after it has a clear operational owner and a matching requirement module.

## 🧱 Tech Direction Snapshot

- **Preferred stacks / frameworks**:
  - API: Go with scenario-local orchestration packages and HTTP transport adapters
  - UI: React + TypeScript + Vite
  - CLI: Go, thin wrapper over API/runtime capabilities
  - Storage: embedded SQLite for queue and execution persistence
- **Data + storage expectations**:
  - Queue requests and execution history are persisted in a scenario-local SQLite file
  - Scenario requirements, coverage artifacts, and phase logs remain scenario-local on disk
  - Lifecycle-managed environment variables remain the only supported runtime configuration surface
- **Integration strategy**:
  - Test execution stays scenario-local and deterministic
  - Optional agent workflows delegate to `agent-manager`
  - Test Genie consumes Vrooli lifecycle metadata instead of inventing separate runtime discovery paths
- **Non-goals / guardrails**:
  - Do not reintroduce shell-script orchestration as the primary execution path
  - Do not add duplicate control surfaces for the same workflow in CLI, UI, and API
  - Do not treat optional agent-manager capabilities as hard runtime dependencies for core test execution

## 🤝 Dependencies & Launch Plan

- **Required resources**:
  - PostgreSQL
- **Scenario dependencies**:
  - `agent-manager` for optional AI-assisted generation/fix workflows
  - `scenario-auditor` for standards validation
- **Operational risks**:
  - Metadata drift between PRD, requirements, and actual tests makes the system harder to trust
  - UI and integration surfaces can drift if the generation/fix workflow changes without shared test coverage
  - External realtime channels such as agent-manager WebSockets are optional and should not be mistaken for Test Genie's own runtime contract
- **Launch sequencing**:
  1. Keep the Go-native orchestrator, provider-backed phase catalog, and queue/history surfaces stable
  2. Tighten PRD/requirements/test traceability so standards remain actionable
  3. Restore and harden AI-assisted generation/fix flows
  4. Expand vault and coverage UX once the underlying telemetry is stable

## 🎨 UX & Branding

- **Look & feel**: Dense operator tooling with clear phase state, queue health, and execution context. The UI should feel like a reliable control plane, not a marketing surface.
- **Accessibility**: Interactive controls must expose stable labels, loading states, and test selectors.
- **Voice & messaging**: Direct and operational. Messages should help an operator decide what happened, why it happened, and what to do next.
- **Branding hooks**: Minimal. Test Genie should align visually with other internal Vrooli operator tools.

## 📎 Appendix

- `OT-P0-001` is the architectural foundation and is expected to stay complete.
- `OT-P0-002` is still in progress because the AI generation workflow exists, but the delegation/control surfaces are not yet consistently reliable enough to treat as finished.
- `OT-P1-003` remains planned because the current UI only partially exposes coverage and vault guidance.
- Success means the provider-backed phase catalog remains authoritative, metadata stays truthful enough for standards to be high-signal, and operators can understand queue state, execution state, and next actions without digging through implementation details.
