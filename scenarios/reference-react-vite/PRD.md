# Product Requirements Document (PRD)

> **Template Version**: 2.0
> **Canonical Reference**: `/scenarios/prd-control-tower/docs/CANONICAL_PRD_TEMPLATE.md`
> **Validation**: Enforced by `prd-control-tower` + `scenario-auditor`
> **Policy**: Generated once and treated as read-only (checkboxes may auto-update)

## 🎯 Overview

- **Purpose**: Serve as the canonical golden reference implementation for the react-vite scenario template. This scenario demonstrates what a fully-developed, production-ready React+Vite application looks like when it adheres to all applicable steer skills. It is not meant to be deployed as a product — it exists solely as a ground-truth test bed for the development-toolchain-validator to validate steer skill interoperability, development tooling correctness, and quality infrastructure calibration.
- **Primary users/verticals**:
  - development-toolchain-validator (primary consumer — maps skill expectations against this reference)
  - prompt-manager meta optimization team (uses this as the "known-good" target to validate skill improvements)
  - AI agents authoring steer skills (reference implementation of what the skill's guidance produces when fully applied)
  - Human developers (concrete example of Vrooli best practices for react-vite scenarios)
- **Deployment surfaces**: Go API, React UI, CLI (full scenario — exercising the complete toolchain)
- **Value promise**:
  - Provide ground truth: if development-toolchain-validator reports violations on this reference, the tool is wrong, not the reference
  - Demonstrate every architectural pattern that steer skills describe — from API domain organization to test co-location to CLI structure
  - Enable calibration of scoring tools: scenario-completeness-scoring should rate this 96+ ("Production Ready")
  - Ensure scenario-auditor produces zero violations and test-genie passes all phases
  - Serve as the integration test for the entire development ecosystem

## 🎯 Operational Targets

### 🔴 P0 – Must ship for viability

- [ ] OT-P0-001 | API Domain Organization | Implement a Go API with screaming-architecture organization: endpoints grouped by bounded context/domain, not by HTTP verbs or technical layers. At least 3 distinct domain modules demonstrating the pattern.
- [ ] OT-P0-002 | API Error Consistency | All API endpoints return errors in a consistent shape: error code (machine-oriented), human message, structured details, and request/correlation ID. No endpoint-specific error formats.
- [ ] OT-P0-003 | API Health & Lifecycle | Standard /health endpoint with dependency checks, graceful shutdown with connection draining, and proper signal handling (SIGTERM/SIGINT).
- [ ] OT-P0-004 | Storage Layer | PostgreSQL integration with repository pattern: clear separation between handlers (orchestrate) and data access (repository). Schema initialization via migration files. Environment-variable-driven connection configuration.
- [ ] OT-P0-005 | CLI as API Wrapper | Go CLI that is a thin wrapper over the API. Every API endpoint has a corresponding CLI command. CLI uses the same error shapes as the API. Installed via install.sh to ~/.vrooli/bin.
- [ ] OT-P0-006 | React UI Foundation | React + TypeScript + Vite UI with: component-based architecture, proper routing (at least 3 routes), loading/error/empty states for all data-fetching components, and data-testid attributes on interactive elements.
- [ ] OT-P0-007 | Test Architecture | Co-located unit tests alongside production code (api/*_test.go, ui/src/**/*.test.tsx). Centralized test helpers/mocks. Testcontainers setup for database tests. At least 80% structural test coverage across API and UI.
- [ ] OT-P0-008 | Documentation Set | Complete docs/ directory following documentation-health steer: manifest.json, QUICKSTART.md, concepts/ARCHITECTURE.md, reference/api-endpoints.md, reference/cli-commands.md, internal/SEAMS.md, internal/PROBLEMS.md, internal/PROGRESS.md. Bidirectional DOC:/[CODE:] references.
- [ ] OT-P0-009 | Service Configuration | Fully populated .vrooli/service.json with correct port ranges (API 15000-19999, UI 35000-39999), complete lifecycle (setup/develop/test/stop), health checks, and resource declarations.
- [ ] OT-P0-010 | Scenario-Auditor Compliance | Zero violations from `scenario-auditor audit reference-react-vite`. All rules across all categories (api, config, ui, testing, go, typescript, makefile) pass.
- [ ] OT-P0-011 | Test-Genie All-Pass | All 11 test-genie phases pass: structure, standards, dependencies, lint, docs, smoke, unit, integration, playbooks, business, performance.
- [ ] OT-P0-012 | Completeness Score 96+ | `scenario-completeness-scoring score reference-react-vite` returns 96 or higher (Production Ready classification).

### 🟠 P1 – Should have post-launch

- [ ] OT-P1-001 | Interoperability Patterns | Demonstrate proto-first or typed contract patterns between UI and API. Request/response shapes as structured messages, not ad-hoc JSON. Consistent serialization.
- [ ] OT-P1-002 | Pagination & Filtering | At least one list endpoint with proper pagination (cursor or offset), filtering, and sorting. Consistent pattern across all list endpoints.
- [ ] OT-P1-003 | Security Headers | CORS configuration, CSP headers, rate limiting middleware. Input validation at API boundaries. No hardcoded secrets.
- [ ] OT-P1-004 | Accessibility Compliance | WCAG 2.1 AA compliance: semantic HTML, keyboard navigation, ARIA attributes, sufficient color contrast. Lighthouse accessibility score 90+.
- [ ] OT-P1-005 | Error Boundary UI | React error boundaries at route and component level. Graceful error display with retry options. No unhandled promise rejections in console.
- [ ] OT-P1-006 | Integration Tests | Bats-based CLI integration tests. API integration tests using httptest. At least one end-to-end flow tested across API, CLI, and UI.
- [ ] OT-P1-007 | Requirements Traceability | Complete requirements/index.json with all operational targets mapped. Tests tagged with [REQ:ID] comments linking to requirements.
- [ ] OT-P1-008 | iframe-bridge Integration | UI properly integrates with Vrooli's iframe-bridge for orchestration. ui-smoke test passes with browserless screenshot capture.

### 🟢 P2 – Future / expansion

- [ ] OT-P2-001 | Performance Benchmarks | API response times documented. UI Lighthouse performance score 90+. Build time benchmarks for trend tracking.
- [ ] OT-P2-002 | Async Job Pattern | At least one endpoint demonstrating the job pattern: start + status + result endpoints with observable progress.
- [ ] OT-P2-003 | Cross-Scenario Communication | Demonstrate calling another scenario's API as a dependency, showing inter-scenario communication patterns.
- [ ] OT-P2-004 | Design System Usage | UI uses a consistent component library with theme support (light/dark). Typography, spacing, and color scales follow a systematic approach.
- [ ] OT-P2-005 | Makefile Best Practices | Canonical Makefile with all expected targets: start, stop, test, build, fmt, lint, check, logs. Lifecycle targets use vrooli CLI.

## 🧱 Tech Direction Snapshot

- **Preferred stacks / frameworks**:
  - API: Go with standard library HTTP server, gorilla/mux for routing
  - UI: React + TypeScript + Vite, Tailwind CSS
  - CLI: Go with cli-core integration
  - Storage: PostgreSQL via direct SQL (no ORM)
- **Data + storage expectations**:
  - PostgreSQL for all persistent data
  - Schema migrations in initialization/postgres/
  - Environment-variable-driven configuration (no hardcoded connection strings)
- **Integration strategy**:
  - This scenario is self-contained — it does not depend on other scenarios for its core functionality
  - It demonstrates integration patterns that other scenarios should follow
  - The "domain" is deliberately simple (e.g., a task/note management app) so the focus stays on architectural quality, not business complexity
- **Non-goals / guardrails**:
  - Does NOT need to solve a real business problem — the "product" is a demonstration vehicle
  - Does NOT need unique features — it needs exemplary architecture
  - Does NOT need to be deployed publicly — it runs locally for validation
  - Does NOT implement complex business logic — keeps domain simple so architectural patterns are clearly visible
  - Business logic should be just complex enough to meaningfully exercise all steer skill patterns (multiple domain modules, pagination, async jobs, etc.)

## 🤝 Dependencies & Launch Plan

- **Required resources**:
  - PostgreSQL (primary data store)
- **Scenario dependencies**:
  - None for runtime (self-contained by design)
  - development-toolchain-validator (consumes this scenario as a reference — but not a code dependency)
- **Operational risks**:
  - Must be actively maintained as steer skills evolve — when steer skills change, this reference must be updated to match
  - Risk of becoming stale if the meta optimization team doesn't prioritize updates
  - Risk of over-engineering the "demo domain" — keep business logic minimal
- **Launch sequencing**:
  1. Scaffold with simple domain (e.g., task/project management)
  2. Implement API with domain organization pattern
  3. Add storage layer, CLI, and UI
  4. Achieve test-genie all-pass and auditor zero-violations
  5. Connect to development-toolchain-validator as first reference

## 🎨 UX & Branding

- **Look & feel**: Clean, professional, minimal. The UI should be visually polished but simple — it demonstrates UI patterns, not visual creativity.
- **Accessibility**: WCAG 2.1 AA compliance. This is a reference implementation — accessibility must be exemplary.
- **Voice & messaging**: Neutral, technical. The UI content (e.g., form labels, error messages) should follow best practices for clarity and consistency.
- **Branding hooks**: None — this is a development reference, not a branded product.

## 📎 Appendix

### Why This Scenario Exists

The Vrooli ecosystem has 45+ steer skills that guide AI agents during scenario development. Each skill focuses on one architectural dimension (API design, storage patterns, CLI structure, testing architecture, etc.). While individual skills are improved in isolation by prompt-manager's meta optimization team, there is no mechanism to verify that all skills' guidance is **mutually consistent** when applied to the same scenario.

This reference scenario solves that problem by being a **concrete, fully-developed implementation** that all applicable steer skills should converge on. It serves as:

1. **Ground truth for tooling**: If scenario-auditor reports violations here, the auditor rule is wrong. If test-genie fails a phase here, the test phase logic is wrong. If completeness scoring gives a low score, the scoring model is miscalibrated.

2. **Integration test for steer skills**: If two steer skills' guidance produces contradictory structures, this reference cannot satisfy both — the conflict becomes visible and testable.

3. **Living documentation**: Instead of abstract guidance, developers and agents can see exactly what "follow the api-steer" looks like in practice.

### Design Principles

1. **Simple domain, exemplary architecture**: The business logic (task/project management or similar) should be trivially simple. The value is in the architectural patterns, not the features.

2. **Every pattern must be exercised**: If a steer skill describes a pattern (e.g., "use the Job pattern for async work"), this reference must include at least one concrete example of it.

3. **Fully exercisable by tooling**: Every lifecycle command, every test phase, every auditor rule must be exercisable against this scenario. No stubs, no "TODO" implementations.

4. **Maintenance is a feature**: This scenario will be updated as steer skills evolve. The development-toolchain-validator detects when skill content changes (drift detection), signaling that this reference may need updates.

### Steer Skills Applicable to This Reference

The following steer skills (as of creation) are expected to be relevant. Not all need to be connected to development-toolchain-validator immediately — connections should be added as configurations are defined.

**Core architectural steers**: api-steer, storage-steer, cli-steer, interoperability-steer, unit-testing-architecture-steer

**Quality and design steers**: documentation-health, screaming-architecture-audit, react-coherence, react-stability, code-cleanup, refactor, domain-compression, cognitive-load-reduction

**Testing steers**: test, e2e-testing, performance, security

**UX steers**: ux, experience-architecture-audit, navigation-integrity-audit, polish

**Specialized steers**: error-semantics-recovery-path-design, failure-topography-and-graceful-degradation, boundary-of-responsibility-enforcement, seam-discovery-and-enforcement, invariant-discovery-and-enforcement
