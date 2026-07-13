# Product Requirements Document (PRD)

> **Template Version**: 2.0
> **Canonical Reference**: `/scenarios/business-health/docs/reference/canonical-prd-template.md`
> **Validation**: Enforced by Test Genie health-provider phases
> **Policy**: The PRD is the product contract; requirement modules and tests should trace back to the targets defined here.

## 🎯 Overview

- **Purpose**: Test Genie is Vrooli's scenario-local testing command center. It orchestrates phased test execution, preserves immutable evidence, and coordinates verified remediation from completed-run findings.
- **Primary users/verticals**:
  - scenario authors validating local apps before deployment
  - operators auditing execution history, findings, and requirement evidence
  - AI agents performing explicitly selected remediation through Agent Manager
- **Deployment surfaces**: Go API, React UI, Go CLI
- **Value promise**:
  - Replace ad hoc shell-based testing with a deterministic Go-native orchestration layer
  - Make test execution legible through execution history, structured findings, and per-phase artifacts
  - Provide one findings-first control surface for execution, remediation, verification, and requirement evidence
  - Raise the baseline quality bar for every scenario that ships through Vrooli
  - Be the self-describing front door for climbing every scenario's maturity ladder: after any run, surface per phase where each capability stands and the single next move, via the [Phase Capability Contract](docs/concepts/phase-capability-contract.md)

## 🎯 Operational Targets

### 🔴 P0 – Must ship for viability

- [ ] OT-P0-001 | Internal Test Orchestrator | Run Test Genie phases from scenario-local Go packages with explicit presets, phase toggles, artifacts, and requirement sync decisions.
- [ ] OT-P0-002 | Evidence-driven Remediation | Support one Agent Manager-backed remediation job from completed execution findings, verified by a server-owned rerun.

### 🟠 P1 – Should have post-launch

- [x] OT-P1-003 | Remediation Evidence UX | Surface requirement evidence, execution insights, and verified remediation deltas directly in the UI so operators can steer improvements without reading raw logs.

### 🟢 P2 – Future / expansion

- [ ] No committed P2 targets yet. Add future work here only after it has a clear operational owner and a matching requirement module.

## 🧱 Tech Direction Snapshot

- **Preferred stacks / frameworks**:
  - API: Go with scenario-local orchestration packages and HTTP transport adapters
  - UI: React + TypeScript + Vite
  - CLI: Go, thin wrapper over API/runtime capabilities
  - Storage: embedded SQLite for execution and remediation persistence
- **Data + storage expectations**:
  - Execution evidence and remediation job history are persisted in a scenario-local SQLite file
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
  - `agent-manager` for optional evidence-bound remediation workflows
  - health-provider scenarios for descriptor-backed phase validation
- **Operational risks**:
  - Metadata drift between PRD, requirements, and actual tests makes the system harder to trust
  - UI and integration surfaces can drift if the remediation workflow changes without shared test coverage
  - External realtime channels such as agent-manager WebSockets are optional and should not be mistaken for Test Genie's own runtime contract
- **Launch sequencing**:
  1. Keep the Go-native orchestrator, provider-backed phase catalog, and execution/remediation history surfaces stable
  2. Tighten PRD/requirements/test traceability so health findings remain actionable
  3. Maintain verified Agent Manager-backed remediation flows
  4. Expand evidence UX once the underlying telemetry is stable

## 🎨 UX & Branding

- **Look & feel**: Dense operator tooling with clear phase state, execution evidence, and remediation context. The UI should feel like a reliable control plane, not a marketing surface.
- **Accessibility**: Interactive controls must expose stable labels, loading states, and test selectors.
- **Voice & messaging**: Direct and operational. Messages should help an operator decide what happened, why it happened, and what to do next.
- **Branding hooks**: Minimal. Test Genie should align visually with other internal Vrooli operator tools.

## 📎 Appendix

- `OT-P0-001` is the architectural foundation and is expected to stay complete.
- `OT-P0-002` is delivered through findings-first remediation, with Agent Manager completion remaining provisional until Test Genie verifies a rerun.
- `OT-P1-003` remains planned because the current UI only partially exposes remediation history and verified deltas across all scenarios.
- The [Phase Capability Contract](docs/concepts/phase-capability-contract.md) is the SSOT for how phases declare their maturity ladder, North Star, and structured remediation docs, and how the provider-computed per-phase standing reaches the agent at the end of a run. Providers own the ladder + docs; Test Genie only aggregates and renders (guard-tested — no phase-specific knowledge in Test Genie). The provider-conformance phase enforces the contract (advisory first, gating for compliant phases; native phases carry an explicit documented exemption).
- Success means the provider-backed phase catalog remains authoritative, metadata stays truthful enough for standards to be high-signal, and operators can understand execution evidence, remediation state, and next actions without digging through implementation details.
