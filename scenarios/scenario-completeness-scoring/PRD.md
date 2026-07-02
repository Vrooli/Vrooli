# Product Requirements Document (PRD)

> **Template Version**: 2.0
> **Canonical Reference**: `/scenarios/business-health/docs/reference/canonical-prd-template.md`
> **Validation**: Enforced by `business-health` (the test-genie `business` phase)
> **Policy**: Generated once and treated as read-only (checkboxes may auto-update)

## 🎯 Overview

Purpose: The fast, cached, focused scenario status layer for the Vrooli fleet. Answers "what is the current state of scenario X and what should I focus on next?" in under one second, from one command, with honest staleness labels. Reads only cached on-disk artifacts (test phase results, run index, requirements registry, service manifest, UI sources) — it never runs tests and never blocks on other services. Complements test-genie (the slow, fresh, deep validation layer that produces the artifacts) and ecosystem-manager (the live control loop that computes maturity on live findings). All three share the same maturity vocabulary and gate predicates via packages/maturity-go, and the same digest/freshness verdict logic via packages/freshness-go, so their answers agree by construction.

Target users: AI agents working on scenarios (session-start orientation), ecosystem-manager (planned metric source for importance-aware scheduling), test-genie (report supplement block shells this CLI), swarm-manager (scenario catalog completeness enrichment), and human operators (CLI + thin status dashboard UI).

Deployment surfaces: Connect-RPC API + CLI (programmatic-first; the CLI core path works with zero services running) + thin React status dashboard. Internal tooling tier — local stack only.

Value proposition: One command, <1s warm response, zero network calls on the core path, with every number labeled by the tree digest it was computed against, plus per-phase fresh/stale/unknown verdicts and a copy-pastable refresh command — turning scenario orientation from a multi-tool investigation into a single authoritative read.

## 🎯 Operational Targets

Operational targets are measurable outcomes; checkboxes may auto-update based on validation.

### 🔴 P0 – Must ship for viability
- [ ] OT-P0-001 | Score Read Path | `scenario-completeness-scoring score get <scenario> [--json]` returns in <1s warm with zero network calls on the core path, computed purely from cached filesystem artifacts of the target scenario.
- [ ] OT-P0-002 | Maturity Headline | Output leads with the maturity rung R0–R4 computed via the shared maturity-go ladder gate predicates over per-dimension error/open counts decoded from cached phase-results findings, labeled "as of digest td:…" — never presented as ecosystem-manager's live state.
- [ ] OT-P0-003 | Composite Score | 0–100 composite with classification band and per-dimension breakdown (quality / coverage / quantity / UI signal groups), each line showing observed counts, thresholds, and awarded points.
- [ ] OT-P0-004 | Recommendations | Prioritized recommendations with estimated point impact plus a phased action plan ("do X, worth ~Y points") surfaced in every score response.
- [ ] OT-P0-005 | Staleness Honesty | A freshness block computed via freshness-go (current tree digest vs digests stamped on recorded runs in coverage/runs.index.json) showing per-phase fresh/stale/unknown and a copy-pastable suggested refresh command; never-tested scenarios degrade to "unknown", not fake-fresh.
- [ ] OT-P0-006 | Resilient Collection | Every signal collector is registered behind a circuit breaker — a failing or malformed source disables that collector, redistributes its weight, and surfaces the degradation in output; malformed input must never crash the score path.

### 🟠 P1 – Should have post-launch
- [ ] OT-P1-001 | Importance Enrichment | Best-effort importance display line composed from scenario-dependency-analyzer centrality and swarm-manager recent-activity, with a hard 1s combined budget, silently omitted when sources are unreachable; the core score path never touches the network.
- [ ] OT-P1-002 | Consumer Contract | Stable proto ScoreService (GetScore) consumed by the test-genie report supplement (2s budget, silent skip) and discoverable via cli-health/ui-health manifests.
- [ ] OT-P1-003 | Status Dashboard UI | Thin read-only dashboard rendering the same GetScore payload (score, rung, breakdown, freshness, recommendations) with loading/error/empty states per template standards.

### 🟢 P2 – Future / expansion
- [ ] OT-P2-001 | Fleet Bulk View | List latest persisted scores across all scenarios through `ScoreService.ListScores` and `score list`, with server-side sort/filter/pagination over score snapshots so catalog consumers such as swarm-manager can use the proto contract without legacy JSON compatibility; expose fleet snapshot aggregates as measures for federation.
- [ ] OT-P2-002 | What-If Analysis | Simulate metric changes and report projected score delta (port from the old implementation only if it stays cheap on the new signal set).

## 🧱 Tech Direction Snapshot

Preferred stacks: Go Connect-RPC API following react-vite template v1.0.1 standards — proto-first contracts under `packages/proto/schemas/scenario-completeness-scoring/v1`, screaming architecture under `api/internal/<domain>`. Go CLI over generated Connect clients. React + Vite + Tailwind for the status dashboard UI. Domains: `signals` (collector registry over cached artifacts: requirements registry + sync metadata, phase-results findings decoded to proto ArchitectureFinding and mapped through the maturity-go dimension vocabulary, service manifest, UI heuristics), `freshness` (tree-digest computation + runs.index.json verdicts via freshness-go), `scoring` (signal assembly → ladder rung + composite + classification + recommendations + action plan).

Preferred storage: SQLite is owned by the scoring domain for digest-deduplicated `score_snapshots`. The background sweeper is the single writer; `GetScore` remains a pure read, and fleet-shaped reads (`ListScores`, UI fleet table, measures) must be O(query) over persisted state rather than O(fleet) recomputation.

Integration strategy: Shared pure-logic packages (maturity-go, freshness-go) — shared code, not shared services. Optional network enrichment (dep-analyzer, swarm-manager) is best-effort with hard budgets. The old REST/gorilla-mux API and legacy JSON field names are NOT preserved (greenfield rule; consumers re-point to the new proto contract).

Non-goals: No lifecycle event ledger, no bespoke search-hub provider registration beyond measures federation, no scenario-qa findings bridge, no per-scenario freshness phase configuration (anti-gaming operator decision), no service calls from ecosystem-manager's control loop into this scenario, and no legacy bulk-JSON compatibility shim. Digest scope is scenario-dir-only in v1 (documented limitation: edits to shared packages/* do not stale-ify dependent scenarios).

## 🤝 Dependencies & Launch Plan

Required resources: None beyond the standard template runtime. The core path is filesystem-only and requires no running services.

Scenario dependencies: Optional/best-effort only — scenario-dependency-analyzer (centrality score) and swarm-manager (recent activity) for the importance enrichment line; both degrade gracefully to silent omission when unreachable.

Operational risks: (1) Stale cached artifacts misread as current state — mitigated by the digest label and per-phase verdicts being the headline, not a footnote. (2) Phase-results schema drift — the decoder is written against sampled live files with an EM-format fallback; malformed input trips the collector circuit breaker rather than crashing. (3) Scoring a scenario that has never been tested — produces unknown verdicts and a degraded-but-honest output with clear "no recorded runs" messaging.

Launch sequencing: This scenario ships standalone first with the full P0 core path (score read, maturity headline, composite, recommendations, staleness, resilient collection). test-genie report supplement integration and ecosystem-manager importance scheduling integrate against the stable proto contract afterwards. P1 dashboard UI and importance enrichment follow core path stabilization. P2 bulk view and what-if analysis are queued for post-stabilization.

## 🎨 UX & Branding

User experience: Dense, scannable status surfaces optimized for quick orientation. CLI output leads with the rung and composite score, followed by per-dimension breakdown, freshness block, and recommendations — ordered to answer the most urgent question first. The dashboard mirrors this hierarchy: rung badge at the top, breakdown table, freshness indicators, and action list below. Loading, error, and empty states are explicitly handled per template standards so consumers never see a blank or silent failure.

Visual design: Vrooli-default design kit. Status-color semantics (rung colors, classification bands, freshness badge states: fresh/stale/unknown) are reserved exclusively for their defined states and never repurposed for decorative use. Typography follows template defaults; dense tabular layout for breakdown lines. PWA install surface: retain the seeded `ui/public/site.webmanifest`, `apple-icon-180.png`, `favicon-196.png`, and maskable manifest icons as valid placeholders; replace generic icons when final product branding is confirmed.

Accessibility: Template accessibility floors enforced — axe-clean landmark structure, full keyboard navigation, i18n for all user-facing strings (en/ar/ja), and cimode-stable automated tests via `data-testid` selectors. Voice and messaging: factual and terse; every number carries its provenance (digest label, thresholds, awarded points); recommendations are phrased as concrete next actions with explicit point impact ("fix X, worth ~Y points"). No branding hooks beyond template defaults.

## 📎 Appendix

- Shared packages: `packages/maturity-go` (ladder gate predicates, maturity vocabulary), `packages/freshness-go` (tree-digest computation, staleness verdict logic).
- Proto contract location: `packages/proto/schemas/scenario-completeness-scoring/v1` — ScoreService plus MeasuresService RPCs.
- Cached artifact sources consumed by the `signals` domain: `coverage/runs.index.json` (run index), phase-results findings files (decoded to ArchitectureFinding proto), requirements registry, service manifest, UI heuristics.
- Digest scope limitation (v1): scenario directory only; edits to `packages/*` do not stale-ify dependent scenario digests — documented known limitation for v1.
