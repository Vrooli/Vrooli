# Guide Space — SWE Task & Skill Coverage

> **Model & terminology** — the projection model, status legend, and how coverage (the numerator)
> is computed are defined once in `meta-optimization-manager/docs/concepts/COVERAGE-MODEL.md`
> _(planned)_. This document is the *Guide* denominator only.

## Purpose

The **denominator** for the *Guide* projection: the space of software-engineering tasks an agent
must handle in this project, each mapped to the prompt-manager skill(s) that guide it and whether
an empirical gate exercises it. Per-skill **health/efficiency** (the numerator) is computed live
from `prompt-manager graph health`, not stored here.

Scope: **SWE tasks only.** Marketing/monetization/business skills (`brand-manager`,
`funnel-builder`, `seo-optimizer`, the monetization/marketing classifiers, `landing-page-*`) are
deliberately out of the local-coding-readiness scope.

## This Space

| | |
|---|---|
| Projection | Guide |
| Owner | `prompt-manager` (owns the skill graph this extends) |
| Denominator confidence | `SKETCH` — the SWE-task taxonomy is a first cut (judgment-defined), and the skill inventory was not exhaustively enumerated; full population deferred to the initiative. |
| Sibling spaces | `search-hub/docs/spaces/answer-space.md`, `test-genie/docs/spaces/validate-space.md` |
| Legend | `COVERED` a skill guides it · `PARTIAL` scattered/partial skills · `MISSING` no skill · Empirical gate: `—` none yet (built in the empirical-gate phase). |

## Coverage Grid

| # | SWE task | Guiding skill(s) | Status | Gate | Notes |
|---|---|---|---|---|---|
| **Understand** | | | | | |
| G1 | Explore / understand a codebase | `explore` + the Answer projection | COVERED | — | Answer providers reduce the read cost. |
| G2 | Audit architecture / alignment | `screaming-architecture-audit`, `architecture-scope`, `boundary-of-responsibility-enforcement` | COVERED | — | |
| G3 | Find seams / testability | `seam-discovery-and-enforcement` | COVERED | — | |
| G4 | Clarify intent / decisions / invariants | `intent-clarification`, `decision-boundary-extraction`, `invariant-discovery-and-enforcement`, `concept-vocabulary-unification` | COVERED | — | |
| **Plan** | | | | | |
| G5 | Author an implementation plan | `implementation-plan-authoring`, `plan-skill-discovery` | COVERED | — | |
| G6 | Brainstorm / workshop an idea | `idea-workshop` | COVERED | — | |
| G7 | Scope a change | `feature-scope`, `bugfix-scope`, `refactor-scope`, `audit-scope`, `platform-scope` | COVERED | — | |
| **Build / change** | | | | | |
| G8 | Build / repurpose a scenario | `ecosystem-fit`, `scenario-generation`, `scenario-capability-extraction` | COVERED | — | |
| G9 | Add / modify a CLI command | `cli-steer` | COVERED | — | |
| G10 | Add / modify a proto / RPC | `api-steer`, `interoperability-steer`, `proto-contract-audit` | COVERED | — | |
| G11 | Add / modify UI | `react-coherence`, `react-stability`, `experience-architecture-audit`, `polish` | COVERED | — | |
| G12 | Storage / DB work | `storage-steer` | COVERED | — | |
| G13 | Bundle / integration | `bundle-integration-steer` | COVERED | — | |
| **Test** | | | | | |
| G14 | Write / strengthen tests | `e2e-testing`, `seam-discovery-and-enforcement` | COVERED | — | A unit-test-architecture skill is referenced in manifests; verify/inventory in the initiative. |
| **Debug / fix** | | | | | |
| G15 | Debug a non-obvious issue | `scientific-debugging` | COVERED | — | |
| G16 | Fix quality / lint / types | `quality-health`, `code-cleanup`, `cognitive-load-reduction` | COVERED | — | |
| G17 | Refactor | `refactor`, `refactor-scope` | COVERED | — | |
| **Harden (audits)** | | | | | |
| G18 | Error handling / graceful degradation | `error-semantics-recovery-path-design`, `failure-topography-and-graceful-degradation` | COVERED | — | |
| G19 | Idempotency / replay safety | `idempotency-replay-safety-hardening` | COVERED | — | |
| G20 | Performance | `performance` | COVERED | — | |
| G21 | Security | `security` | COVERED | — | |
| G22 | Change-resilience / control surfaces | `change-axis-and-evolution-resilience-audit`, `control-surface-tunable-levers-design`, `signal-and-feedback-surface-design` | COVERED | — | |
| **Docs / requirements** | | | | | |
| G23 | Documentation health | `documentation-health`, `documentation-search`, `knowledge-observatory-tools`, `spec-sync` | COVERED | — | |
| G24 | Requirements / traceability | `requirements-traceability-steer` | COVERED | — | |
| G25 | Maturity / readiness review | `scenario-maturity-ladder`, `scenario-readiness-review`, `scenario-improvement-campaign` | COVERED | — | |
| **Dependencies / deploy** | | | | | |
| G26 | Dependency / package work | `platform-package-consumption-audit`, `platform-package-hardening` | PARTIAL | — | + scenario-dependency-analyzer flows. |
| G27 | Deploy / publish | `deployment-coordinator`, `scenario-to-cloud`, `scenario-to-desktop`, `cross-platform-readiness` | COVERED | — | |
| **Learning loop** | | | | | |
| G28 | Report a bug / friction | `report-bug`, `report-friction`, `conversation-friction-analysis` | COVERED | — | |
| G29 | Capture a capability | `capability-extraction`, `scenario-capability-extraction` | COVERED | — | |
| G30 | Author / improve a skill | `skill-authoring` (+ variants), `skill-validation`, `skill-improvement-suggestions`, `reference-pattern-fitness` | COVERED | — | The meta loop: skills improving skills. |
| **Candidate gaps (`SKETCH`)** | | | | | |
| G31 | Concurrency / race hardening | _(adjacent: `idempotency-replay-safety-hardening`)_ | MISSING | — | No dedicated skill; adjacent to G19. |
| G32 | Observability / telemetry wiring | _(none)_ | MISSING | — | Pairs with Validate V18. |
| G33 | Internationalization (i18n) | _(partial: `react-coherence`)_ | PARTIAL | — | No dedicated skill. |

## Known Gaps & Approaches

- **Empirical gates** — the `Gate` column is uniformly `—`: no task category yet has a local-model empirical gate. Closing this column *is* the empirical-gate phase (tasks generated from these rows, run via agent-manager + workspace-sandbox). "% of tasks with a gate" is the recursive Guide-coverage metric.
- **Skill efficiency** — `Status: COVERED` only means a skill exists; whether it is *programmatic* (command-backed) vs prose-heavy is the live `prompt-manager graph health` numerator + the `--programmatic-home` graduation tracking. Prose-heavy skills are the prose→programmatic backlog (`PROMOTION_LADDER`).
- **Candidate gaps (G31–G33)** — judgment-heavy; the initiative owns whether they get dedicated skills.

## Sources Of Truth

- `prompt-manager graph health` (`ScoreAllWithConfig`) — per-skill health heuristic (edges, code-usage, content-length).
- `prompt-manager skill list` / `discover` — the live skill inventory.
- `skill update --programmatic-home` — graduation tracking; `PROMOTION_LADDER.md` — prose→CLI→Action lifecycle.
- `AGENTS.md` situational-skill-loading table — the seed task→skill map this extends.

## Cross-References

- `meta-optimization-manager/docs/concepts/COVERAGE-MODEL.md` — canonical model + legend _(planned)_.
- `search-hub/docs/spaces/answer-space.md`, `test-genie/docs/spaces/validate-space.md` — sibling denominators.
