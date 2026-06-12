# Product Requirements Document (PRD)

> **Template Version**: 2.0
> **Canonical Reference**: `/scenarios/prd-control-tower/docs/CANONICAL_PRD_TEMPLATE.md`
> **Validation**: Enforced by `prd-control-tower` + `scenario-auditor`
> **Policy**: Generated once and treated as read-only (checkboxes may auto-update)

## 🎯 Overview

- **Purpose**: `proto-health` makes Vrooli's Protocol Buffer contracts drift-proof by validating one scenario at a time and publishing a structured proto-surface fact that downstream quality tools can consume.
- **Primary users/verticals**: Agents building scenarios, test-genie, ecosystem-manager, and later scenario-dependency-analyzer / tech-tree-designer.
- **Deployment surfaces**: Connect API, CLI commands, direct UI, test-genie phase integration, maturity ladder signal.
- **Value promise**: Agents stop guessing which proto files are authoritative, which transport world a scenario uses, and whether committed generated artifacts reflect schema source.

## 🎯 Operational Targets

### 🔴 P0 – Must ship for viability

- [x] OT-P0-001 | Proto standard defined | `packages/proto/STYLE_GUIDE.md` retires `@layer`, defines domain organization, `v<n>/shared/`, supported annotations, and version directory rules.
- [ ] OT-P0-002 | Scenario scaffold documented | Generated `proto-health` scenario has PRD, requirements, architecture, domain, seam, testing, problems, and progress docs that describe the real validator target state.
- [ ] OT-P0-003 | Per-scenario proto validation | `ProtoHealthService.ValidateScenario` validates one scenario and returns stable findings with severity, code, location, message, and suggestion.
- [ ] OT-P0-004 | Proto surface fact | `ProtoHealthService.DescribeScenarioProtos` returns a structured inventory of a scenario's files, services, RPCs, messages, fields, imports, annotations, REST exception payload declarations, and declared transport world.
- [ ] OT-P0-005 | CLI surface | `proto-health validate scenario <name> --json` and `proto-health describe scenario <name> --json` call the generated Connect API and return machine-readable output.

### 🟠 P1 – Should have post-launch

- [ ] OT-P1-001 | Test-genie proto phase | test-genie runs a non-blocking `proto` phase, maps findings to `FINDING_SOURCE_PROTO`, and preserves stable finding IDs.
- [ ] OT-P1-002 | Ecosystem-manager maturity signal | ecosystem-manager consumes proto findings as an R2 `proto-health` soft-boost dimension.
- [ ] OT-P1-003 | Direct UI | The UI lists fleet proto-health status and per-scenario findings/surface facts with loading, error, and empty states.
- [ ] OT-P1-004 | Proto contract audit skill | `proto-contract-audit` steer skill loads through prompt-manager and routes deep API/interoperability decisions to the existing sibling skills.
- [ ] OT-P1-005 | CI gen-sync gate | CI runs `cd packages/proto && make verify-committed-gen` so committed generated artifacts cannot drift from schema sources.

### 🟢 P2 – Future / expansion

- [ ] OT-P2-001 | Dependency-analyzer handoff | scenario-dependency-analyzer consumes `DescribeScenarioProtos` for fleet graph analysis instead of re-reading descriptors itself.
- [ ] OT-P2-002 | Semantic proto reuse search | A future provider can index `ProtoSurface` facts for type reuse suggestions; no search provider ships in v1.
- [ ] OT-P2-003 | Ratchet hardening | Warning-level maturity checks can be promoted after fleet migration evidence, without stored per-scenario baselines in v1.

## 🧱 Tech Direction Snapshot

- Preferred stacks / frameworks: Go API and CLI, Connect-RPC, React/Vite UI, committed `packages/proto` generated artifacts, SQLite only for local scenario state if needed.
- Data + storage expectations: v1 is primarily read-only over repo files and descriptor artifacts; persisted state is not required for validation or surface facts.
- Integration strategy: read the committed fleet descriptor (`packages/proto/gen/descriptor/image.binpb`), shell/consume `buf` for checks it already owns, expose facts through Connect + CLI, then let test-genie and ecosystem-manager consume findings.
- Non-goals / guardrails: no cross-scenario graph computation, no declared-vs-actual dependency drift analysis, no fleet-aware dead-proto detection, no `@layer` enforcement, no hand-editing generated code, no search-hub provider in v1.

## Ecosystem Fit

- **Role**: meta / interface-enabler.
- **Interfaces served**: Programmatic Connect + CLI for test-genie and future analyzers; direct UI for human inspection.
- **Done obligations**: clean reusable Connect contract, CLI commands discoverable through cli-health, and production-ready UI states.
- **Compound-value seam**: `DescribeScenarioProtos` is the stable fact RPC that later scenarios consume instead of re-implementing descriptor analysis.
- **Self-improvement**: advances engineering quality by converting proto conventions into executable checks and maturity signals.
- **Monetization**: not applicable; internal validation/meta scenario.

## 🤝 Dependencies & Launch Plan

- Required resources: local filesystem, `buf`, Go toolchain, existing `packages/proto` generation pipeline.
- Scenario dependencies: test-genie and ecosystem-manager for P1 integration; prompt-manager for the audit skill; downstream dependency-analyzer consumes the fact RPC later.
- Operational risks: existing fleet annotations are legacy-heavy, `gen/descriptor/image.binpb` may lack source info, and generated artifact drift is already possible until CI is wired.
- Launch sequencing: define style guide and docs, scaffold scenario, implement descriptor reader seam, implement validation findings, implement surface fact RPC, add CLI/UI, wire quality loop, self-validate on representative scenarios.

## 🎨 UX & Branding

- Look & feel: quiet operational validator UI, dense but scannable; no marketing hero.
- Accessibility: keyboard navigable, clear loading/error/empty states, testable selectors.
- Voice & messaging: precise findings with prescriptive suggestions; avoid blame language.
- Branding hooks: reuse Vrooli default design kit; visual emphasis on scenario, severity, domain, and transport status.

## Appendix

- Plan: `~/.vrooli/plans/proto-health-scenario-fleet-proto-standardization.md`
- Proto guide: `packages/proto/STYLE_GUIDE.md`
- Descriptor reader precedent: `packages/measures-go/paramschema.go`
- Validator precedent: `scenarios/cli-health/api/internal/services/manifestvalidation/`
- Phase integration precedent: `scenarios/test-genie/api/internal/orchestrator/phases/phase_contracts.go`
