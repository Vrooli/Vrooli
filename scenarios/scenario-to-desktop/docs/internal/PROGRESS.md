# Progress Log

Historical execution details are preserved beneath the protected runtime home:
`plan-artifacts/docs-html-progress-20260908/scenarios/scenario-to-desktop/docs/internal/PROGRESS.md`.
The archive retains exact entries, validation receipts, and unresolved qualifications;
relocation does not resolve a finding or establish current readiness.
The table retains selected dated milestones; consult the archive for the full sequence.

These are dated development milestones, not a current readiness verdict.
Detailed pre-cleanup entries are preserved beneath the protected runtime home:
`plan-artifacts/docs-cleanup-20260907-final-txumdx73/scenarios/scenario-to-desktop/docs/internal/PROGRESS.md`.
Use that original for command transcripts, measurements, and full qualifications.
Hashes and recovery instructions are in the project documentation cleanup record
(`docs/internal/PROGRESS.md`, "Documentation cleanup completion — 2026-09-07").
Current unresolved issues belong in this scenario's problem ledger; the historical
handoff notes below remain follow-up leads until checked against current evidence.

## 2026-09-04 — Outcome-linked learning

Implemented recommendations 1–3. The usage skill captures advice decisions and
verified outcomes through `vrooli-memory learning record`; the improvement board
adds failure recurrence, success effort, and advice-outcome measurements. Contexts,
unresolved tasks, missing evidence, and test provenance remain distinct. Targets
stay null until comparable operator baselines exist.

The shared implementation, scope provisioning receipt, capture/recall verification,
fixture evidence, and known Memory suite limitations are in the
[Vrooli Memory work record](../../../vrooli-memory/docs/internal/PROGRESS.md). This scenario's five-phase validation passed
in Test Genie `20260904-213256-6066e7da`. Fresh runtime explanations and fixtures passed for both
setpoint programs. The skill divergence review resolves missing recall, evidence,
and baselines conservatively; no release gate or evidence floor was weakened.

## Skill and program setup (2026-09-04)

### Summary

Added the scenario-owned usage/improve declaration, three governed read programs,
and the `scenario-to-desktop-usage` memory scope. The usage skill replaces stale
CLI recipes with operation selection and evidence boundaries. The improve skill
retains the complete desktop behavior contract and routes missing measurements
instead of claiming that capture counts or successful reads establish maturity.
This work does not implement the remaining desktop platform adapters.

The usage role is owed because `cli/manifest.json` exists. The improve role is
owed because deployment-manager and scenario-to-android declare this dependency.
No independent feature role was extracted: the inventory supplied no pattern
shared by two agents that requires another skill. Role/rung canon is
`docs/agent-system/SKILL_AUTHORING.md`; authoring followed `skill-set-authoring`,
`skill-authoring-tools`, `improve-skill-authoring`, and `program-runtime`.

### Sensor and program inventory

| Surface | Observation on 2026-09-04 | Interpretation |
|---|---|---|
| `program-runtime bindings condition --scenario scenario-to-desktop --window-seconds 604800` | 48 bindings; five usage-program reads selected by the board; selected reads dormant in the receipt window | Measured, but insufficient serving evidence; never exercise writes to improve this reading |
| `agent-manager.friction-digest` | `prog_f8e33772-d69e-4ff2-a380-addda6354473`: 40 recent runs, truncated window, zero attributable desktop episodes | Sampled observation; cannot establish representative zero friction |
| `measures-health validate scenario scenario-to-desktop` | Passed with seven parameter-tier warnings | Snapshot measures exist; goal-level outcome aggregates below are pending |
| `business-health matrix show scenario-to-desktop --format summary` | 34 requirements, four modules, 18 operational targets; no unproven status claims | Linkage inventory, not implementation completion |
| `evals/*.primary.json` | No declared corpus | No invented floor |
| `performance-baseline.json` | Five cold and five warm historical samples; cohort metadata requires review | Baseline candidate, not a live reading or approved budget |
| `pipeline-inspect` | Three declared read bindings; one selected pipeline; at most five returned task references from a 100-task sample | Diagnosis only; saturation is explicit |
| `evidence-inventory` | Two declared read bindings; at most five returned capture references | Historical metadata, not candidate-artifact proof |
| `setpoint-read` | External binding sensor plus explicit external/pending rows | Six rows match the improve skill; `ok` describes execution only |

The runtime's contract catalog loads at startup. Direct `library run` reads the
scenario source files; library search needs the refreshed catalog. Registration
uses the scenario-owned files and `.vrooli/service.json`, with no library-row edits.

### Filed obligations

Reuse these references on the next cycle. Measurement workers use
`prompt-manager skill read measures-adoption`. Filing is not completion.

| Row or missing operation | Work reference |
|---|---|
| desktop-behavior | `chore/scenario-to-desktop-measure-desktop-behavior` |
| runtime-performance | `chore/scenario-to-desktop-measure-runtime-performance` |
| pipeline-performance | `chore/scenario-to-desktop-measure-pipeline-performance` |
| engineering-quality | `chore/scenario-to-desktop-measure-engineering-quality` |
| Governed target/profile/matrix operations | `fix/scenario-to-desktop-validation-bindings-w1` |
| Server-owned pipeline wait | `fix/scenario-to-desktop-pipeline-wait-w1` |

Preserve and consult existing emulator/Bridge work before proposing transport
changes, including `execute/adopt-vrooli-emulator-in-deployment-flows`,
`execute/vrooli-emulator-remote-node-backend`,
`execute/macos-real-device-validation-over-bridge`, and
`execute/windows-real-device-validation-over-bridge`.

External defect: Scenario QA `knw-1788554309150137139` records the governed
measures-health validation response's unresolved native-report type. The binding
is declared and callable, but program `prog_04fc107e-22dc-4cf9-9168-a2c90230edd9`
failed decoding `ScenarioCoverageReport`. Its CLI remains usable. The desktop
board does not substitute an invented reading for this failed probe.

### Findings: skill validation

| Capability | Primary path | Verification | Failure path |
|---|---|---|---|
| Build and inspect | Mode decision, typed pipeline CLI, pipeline-inspect | Exact scenario/pipeline identity and owner stage status | Typed read errors and stage diagnosis |
| Locate evidence | evidence-inventory, journey CLI, validation UI | Counts, references, and separately verified matrix identity | Unknown/stale/wrong-target evidence retained |
| Regulate the ramp | setpoint-read and external friction program | Same six rows, explicit validity and baseline gaps | First-read filing and work-ladder routing |
| Learn across attempts | Declared memory scope and task/work records | Created scope; before/after evidence references | Failed bindings recorded separately from domain outcomes |

Initial Major findings were repaired: the old skill excluded bundled mode while
prescribing it, advertised unsupported `--wait` and deploy flags, duplicated the
CLI manual, and lacked rung labels, the learning spine, and matrix ownership.

Divergence probe: attempted two compliant executions for (1) newest versus exact
pipeline selection, (2) pending-sensor routing versus code execution, and (3)
close-out with unavailable native targets. The final text selects exact IDs for
continuation, files/routes missing sensors before implementation, and prohibits
a full-maturity claim with required unavailable cells. No divergence remained on
these probed instructions. No remaining Critical or Major authoring finding.

Contract integrity: primary skill flows use human CLI output. JSON is confined
to explicitly justified evidence linkage and program envelopes. Programs use
only declared read bindings; no shell, filesystem, media fetch, inference, or
delegation exists in their execution paths. No non-Vrooli primary command is
required by the skills. Program/skill references and cited command help were
checked against the local registry and descriptors.

### Prose Retirement Map

| Instruction / Gate | Disposition (Keep/Collapse/Delete) | Rationale | Prerequisite contract | Risk |
|---|---|---|---|---|
| Mode, identity, release, and ownership decisions | Keep | Judgment and authority remain necessary | Canonical desktop evidence contract | Wrong-platform or wrong-artifact claims |
| Repeated pipeline and capture joins | Collapse | Programs return bounded attributable observations | pipeline-inspect and evidence-inventory contracts | Program success confused with release approval |
| Copied flags and build/signing recipes | Collapse | Current help owns the command surface | Verified CLI help and typed descriptors | Historical flags drift |
| Unconditional clean/wait and obsolete deploy examples | Delete | Current behavior and task authority do not support them | pipeline run help; Deployment Manager release workflow | Destructive rebuild or unsupported call |
| Improvement target definitions | Keep | Missing measurements cannot be inferred | PRD, provider targets, evidence contract | Invented floors or suppressed unavailable cells |

### Recommendations and notes

1. **A (Recommended):** Use the registered skills and three programs; execute the
   filed measurement/binding work through the owning implementation workflows.
   Verify each new sensor with attributable before/after evidence. **B:** Retain
   manual UI/CLI operation while those obligations remain open; preserve the same
   evidence boundaries and do not claim automated matrix orchestration.

Initial verification: contract JSON schema checks, clean runtime name-resolution
explanations, all eight declared fixtures in fresh test-provenance sessions, and
26 isolated behavioral assertions passed. Assertions covered malformed inputs,
identity mismatch, missing reads, omitted protobuf zeros, sample saturation,
kernel failure classification, and unavailable setpoint rows. These isolated
checks do not claim native platform or end-to-end release validation.

Final Test Genie run `20260904-204358-95dedc12` passed all five requested phases:
programs, skill-set, docs, business, and structure. Programs and skill-set report
L2 (their ceilings); business and structure report L3. The final contract fixture
replay also passed all eight fixtures, including the explicit 100-task bound.
Registry read `prog_a07d1813-ee3f-4ef5-9868-d5f5cf7f354f` confirms both role IDs;
library search returns all three scenario contracts after lifecycle refresh.

The passing docs phase retains advisory reference debt. Nine marked-path warnings
refer to existing repository-relative files in the new skills, contrary to the
marked-reference contract; Scenario QA `knw-1788554700046604331` records this
validator issue. Other warnings include historical command snippets, derived
counts, and documentation-contract coverage. No reference qualifier, validator,
or quality floor was weakened to hide those findings.

## Changelog

| Date       | Author           | Change % | Description |
|------------|------------------|----------|-------------|
| 2026-08-17 | Codex | Reduced the desktop lint surface without changing behavior: removed an unused journey-test field, replaced an unnecessary command wrapper with `exec.CommandContext`, and used the canonical workflow-artifact conversion. | Archived |
| 2026-08-17 | Codex | Non-regression cleanup: `ElectronSession.Target()` now uses `proto.Clone` instead of shallow-copying a protobuf message with mutex state. | Archived |
| 2026-08-09 | Codex | Produced fresh canonical Linux Electron evidence for both `scenario-to-desktop` and `secrets-manager`: pipelines `4d992263-9aee-63b1-42a6-56bb4e8d97ab` and `dbcbe219-cf34-3f6a-1506-d08c70a0d3f0` passed protocol smoke and desktop journeys, with persisted H.264 captures `28e189cd-4d3b-4191-93cd-e0bee23fc1b8` and `470e2b2c-eb82-4285-826d-7b63c87afda6`. | Archived |
| 2026-08-09 | Codex | Produced the primary mutating Electron video deliverable from fresh pipeline `9b0aaa42-7067-d9e3-8a6c-8f1c8935056e`: BAS leased-desktop evidence passed 1/1 through authenticated loopback CDP, the visible result reported `leased writes: 3`, and persisted capture `048f1780-3eec-4548-a46b-61468664800d` is a 44.2-second H.264 MP4. | Archived |
| 2026-08-09 | Codex | Captured and reviewed a real Secrets Manager Electron AppImage video through the live-desktop recorder and BAS dashboard case: 30-second H.264 MP4, 1920x1080, persisted capture `7fef7af1-216b-415d-a17c-e97aee34a95d`, passed Workflow Health run `a7311bcb-4358-4858-beeb-4f3f2b5bf186`, and zero primary-storage requests/writes. | Archived |

## Current State

The launch-performance baseline is durable at
`docs/internal/performance-baseline.json`. It contains ten measured Linux
Xvfb-compatible Hello Desktop runs on one artifact and host identity: five
cold-designated and five warm-designated. Warm process-to-splash p95 is 651 ms;
cold p95 is 1,745 ms because of one first-launch outlier. The 1-second
process-to-splash budget remains advisory and is not promoted to a release gate
until another baseline confirms the variance.

- **Overall Status**: Baseline native journey proven; provider-specific release claims remain evidence-gated
- **Bundled Mode**: Pipeline and supervisor contracts are implemented; offline/resource-native claims require the selected dependency journey
- **Thin Client Mode**: Supported deployment mode; live Tier 1 route evidence depends on the target server fixture
- **Tests**: Fresh native pipeline passed; latest scenario Test Genie run passed all 20 phases (2026-07-28)
- **Completeness Score**: 93/100 (`nearly_ready`, 2026-07-28); this score does not promote environment-gated provider claims

## Architecture Notes

The scenario-to-desktop codebase follows "screaming architecture" principles effectively:

### Domain Modules (api/)
- `build/` - Desktop application build orchestration
- `bundle/` - Runtime bundling and packaging
- `distribution/` - S3/R2 upload and artifact distribution
- `generation/` - Template generation and scenario analysis
- `pipeline/` - Multi-stage deployment orchestration
- `preflight/` - Runtime validation before packaging
- `records/` - Desktop build record persistence
- `scenario/` - Scenario metadata and desktop status
- `signing/` - Code signing configuration and generation
- `smoketest/` - Smoke testing for built applications
- `state/` - Scenario state persistence
- `system/` - Wine service and system-level operations
- `tasks/` - Task orchestration (investigate/fix workflows)
- `telemetry/` - Deployment telemetry collection
- `toolexecution/` - Tool Discovery and Execution Protocol
- `toolhandlers/` - Tool handler HTTP layer
- `toolregistry/` - Tool registration and discovery

### Shared Infrastructure (api/shared/)
- `errors/` - Domain error types
- `http/` - HTTP utilities (CORS, response helpers, middleware)
- `packaging/` - Package file locating utilities
- `store/` - Generic store interfaces
- `validation/` - Input validation

### Root-Level Files (Cross-Cutting Concerns)
Files in api/ root are intentionally cross-cutting and don't belong in a single domain module:
- `handlers_docs.go`, `handlers_probe.go`, `handlers_icon_preview.go` - HTTP handlers for docs/health/preview
- `handlers_tasks.go` - Task orchestration handlers
- `proxy_hints.go`, `proxy_utils.go` - Proxy URL detection and normalization
- `adapters.go` - Cross-domain adapter interfaces (connects build, generation, pipeline stores)
- `utils.go` - Minimal utility functions (detectVrooliRoot)
- `ports.go` - Port allocation constants

The root package is now minimal - types live in domain packages (e.g., `generation.DesktopConfig`).

## Known Issues

### Security Audit Findings (All False Positives)
- **AUTH-002** at signing/generation/electron_builder.go:24 - Template string for env var reference, not hardcoded password
- **PATH-001** findings - Build-controlled paths, not user input
- **HTTP-002** CORS wildcards - Intentional for tool protocol handlers

### Standards Violations
- 100 violations total (50 medium, 46 low, 4 info)
- Most are PRD linkage issues (requirements missing operational target mappings)
- Some are configuration warnings in service.json lifecycle setup

### Lint Status
- **TypeScript/JavaScript**: 0 issues (clean)
- **Go**: 33 issues (mostly errcheck warnings in test files, 2 unused functions)

## Historical handoff leads retained during condensation

These clauses are preserved from dated entries; later work may have superseded
them. They do not assert that an old failure or gap still exists.

- **2026-07-28 — +0.0%**: Deferred work is tracked separately for multi-framework generation, LAN/public authentication, and secrets-manager/Vault dependency health; no shared-package defect was found during the proto migration.

- **2026-07-23 — +0.2%**: The declared 85% global floor remains enforced and unresolved.

- **2026-07-23 — +0.2%**: All remaining consumers now use `BundleSection`, `BundleSectionHandle`, and `BundleResult` directly; the TypeScript compiler now resolves the same `@/*` alias as Vite.

- **2026-01-15 — +0%**: Error Semantics Completion - Converted all remaining ad-hoc error handling in UI api.ts to use structured ApiError class.

- **2026-01-16 — +0%**: Idempotency & Replay Safety Hardening (UI Layer) - Extended idempotency support from backend to UI layer: (1) Added `idempotency_key` field to `PipelineConfig` TypeScript interface in `api.ts`; (2) Created idempotency key generation utilities in `pipeline-utils.ts` (`generateIdempotencyKey()`, `generateUniqueIdempotencyKey()`, `getSessionId()`, `resetSessionId()`) with session-scoped keys that are stable within a page session but unique across sessions; (3) Updated `pipelineStore.ts` with `isSubmitting` flag and `currentIdempotencyKey` state for double-submission prevention; (4) Added in-flight request guards to `runStage()`, `runFullPipeline()`, and `resumePipeline()` - if already submitting, returns existing pipeline ID instead of creating duplicate; (5) Added `resetForRetry()` action that resets session ID to allow explicit retries with fresh idempotency keys; (6) Added `selectIsSubmitting` and `selectIsBusy` selectors for UI button guards; (7) Documented full idempotency seam architecture in SEAMS.md with design philosophy, implementation details, remaining gaps, and future enhancements.
