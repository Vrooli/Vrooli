# Product Requirements Document (PRD)

> **Template Version**: 2.0
> **Canonical Reference**: `/scenarios/business-health/docs/reference/canonical-prd-template.md`
> **Validation**: Enforced by `business-health` (the test-genie `business` phase)
> **Policy**: Generated once and treated as read-only (checkboxes may auto-update)

## 🎯 Overview
- **Purpose**: Measure how ready the Vrooli project is for lower-powered (local) coding agents to do software engineering. It aggregates coverage across four projections — **Answer** (architectural questions), **Validate** (what is tested + auto-fixed), **Guide** (the SWE-task→skill space), and **Act** (whether each operation is programmatically invocable, joined live from `program-runtime`) — plus **template & reference-scenario convergence**, prioritizes the gaps, and empirically proves readiness with local-model trials. A thin, read-mostly aggregator that makes the meta-optimization team's *measurement and prioritization* programmatic so judgment can focus on the actual improvement.
- **Primary users/verticals**: (1) AI coding agents — query readiness/focus/gaps to know where to work and how much to trust generated answers; (2) the meta-optimization team / operators steering project self-improvement, who today carry this knowledge as prose and run dozens of CLI commands by hand.
- **Deployment surfaces**: CLI (primary; agent- and operator-facing; typed proto-JSON), API (Connect-RPC), UI (operator scoreboard).
- **Value promise**: Replaces manual, prose-driven scorekeeping with a programmatic, validated, **honest** readiness scoreboard + focus engine. Shrinks the meta-optimization team to agents executing improvements the scoreboard points them at. The honest confidence model is the enabling primitive for documentation-first development with local models — an agent can trust a generated answer exactly to the degree attested, and skip re-reading code to that degree.

## 🎯 Operational Targets

### 🔴 P0 – Must ship for viability
- [ ] OT-P0-001 | Readiness snapshot | `status [--json]` returns per-projection coverage (Answer/Validate/Guide) + denominator-confidence + the latest empirical trend, computed live from each owner's `space --projection` verb joined against the live registries; coverage is computed, never stored; degrades gracefully when an owner is down. _Shipped (`coverage status`) with per-cell live joins for Answer, Validate, and Guide. Honest caveat, tracked in `docs/internal/PROBLEMS.md`: the owner `space --projection` verb is not built yet, so denominators are read via the doc-parse fallback (same `spacedoc` parser)._
- [ ] OT-P0-002 | Actionable focus | `focus [--json]` returns the ranked next-best gaps (impact × importance) across all projections + convergence, each with its qualitative context, so an agent/operator knows where to work without parsing raw numbers. _Shipped (`focus next`)._
- [ ] OT-P0-003 | Honest gaps registry | `gaps [--projection] [--cell]` surfaces every known gap with notes/approaches/context (not just a percentage); cross-cutting/global gaps and explored-but-unbuilt ideas live durably here. _Shipped (`gaps list` / `gaps show` / `gaps note`)._
- [ ] OT-P0-004 | Base-document integrity | Validate the three space-definition documents themselves — every referenced skill/provider/phase exists (no stale/broken refs) and Guide rows are flagged when they do not map to exactly one skill. _Shipped (`coverage validate-docs`), including the `ungraduated_pointer` rule for the Guide→Validate→Answer gradient._

### 🟠 P1 – Should have post-launch
- [ ] OT-P1-001 | Empirical readiness trials | `trials run` exercises a local model on fixture SWE tasks (add-feature/research/comprehend/bugfix + negative cases) via agent-manager's sandboxed runner (opencode + local model), evaluates the produced diff against the fixture oracle in MoM, and records success-rate + tokens + wall-time as a historical trend; tracks "% of Guide tasks with a live gate". _Code complete (runner + evaluator + 5 real fixtures with deterministic oracles); **acceptance pending one operator live-e2e pass** to confirm the diff-apply path on a live local model — see `docs/internal/PROBLEMS.md`._
- [ ] OT-P1-002 | Template & reference convergence | Measure the upstream generators: per-template fitness counts, gold-star generated-golden health/staleness, and the convergence trend — surfacing numbers and flagging candidates only (substrate/nomination decisions stay agentic), feeding focus. _Shipped (`convergence status`). The four lenses are honest **filesystem proxies** (LOC, comment-grep), not semantic analysis; the coordinated-edit lens is marked lower-confidence until validated against a frozen fixture._
- [ ] OT-P1-003 | Condition axis | `condition status` and `condition explain-leg` report owner-measured condition findings whose Answer population is derived from live `NOW` cells; `focus next` ranks those findings and the operator console renders them beside Coverage-oriented gaps. _Code complete with focused API, ranking, CLI, and UI coverage; live substrate certification remains an operator validation step._

### 🟢 P2 – Future / expansion
- [ ] OT-P2-001 | Operator UI dashboard | A React/Vite console rendering the readiness scoreboard, focus list, gaps registry, and trials/convergence trends. _Shipped._
- [ ] OT-P2-002 | Readiness answerable via search | "how ready are we / where should I work" is answerable through search-hub today via cli-health command federation (a query surfaces MoM's `status`/`focus`/`validate-docs` commands). A dedicated MoM provider emitting the inline `AttestedAnswer` envelope is **descoped** as redundant — see `docs/internal/DECISIONS.md` (2026-06-24). Re-attestation is pending healthy-substrate restoration.

## 🧱 Tech Direction Snapshot
- Preferred stacks / frameworks: Go (api + cli), React + Vite + Tailwind (ui), Connect-RPC over proto contracts (`packages/proto/schemas/meta-optimization-manager`), typed proto-JSON CLI output, cli-core `ScenarioApp`, api-core/storage.
- Data + storage expectations: SQLite via api-core/storage. Read-mostly aggregator with minimal owned state — the gaps registry (qualitative), the trials history time-series, a cached convergence/fitness-audit index, and short-TTL cached coverage snapshots. Denominators (the space docs) are NOT owned here; they live with their owner scenarios.
- Integration strategy: thin aggregator — read other scenarios' typed CLI/RPC, never re-implement their measurement. Standard read interface: a `space --projection <p> --json` verb on each denominator owner. Preference: shared workflows > resource CLI > direct API. All reads degrade gracefully.
- Non-goals / guardrails: does NOT re-implement measurement (owners keep it); does NOT do the improvements (agentic, directed by focus); does NOT own the space denominators; does NOT make judgment calls (surfaces numbers + candidates, never the substrate/tiering/nomination/root-cause decision); does NOT absorb the judgment-heavy meta-optimization themes (skill/action lifecycle, friction intake, contrarian review, team/agent audits); coverage % is always paired with a denominator-confidence so the board cannot imply false completeness.

## 🤝 Dependencies & Launch Plan
- Required resources: SQLite (default storage). No heavy resources (no Ollama/Qdrant — this aggregates typed JSON, it does not embed or search).
- Scenario dependencies (all soft / degrade gracefully): `search-hub`, `test-genie`, `prompt-manager`, `completeness-scoring`, `code-facts`, `architecture-cartographer`, `scenario-auditor` (reads); `agent-manager` (trials sandboxed spawn). Each space-owner must expose a `space --projection <p> --json` verb (a shared contract dependency).
- Operational risks: denominator honesty (mandatory recursive denominator-confidence); fan-out latency / partial availability (must degrade, never false-fail); trials cost (gate strictly behind explicit `trials run`, sandboxed); the convergence coordinated-edit walkthrough is new mechanization (mark lower-confidence until proven); readiness is ultimately empirical (the board measures infrastructure-readiness; the trials trend is the real proof).
- Launch sequencing: P0 (coverage/focus/gaps/base-doc — readable from existing RPCs + the new `space` verb) → P1 (trials, convergence) → P2 (UI; readiness-via-search is already covered by cli-health command federation, so no dedicated provider is built). The canonical `docs/concepts/COVERAGE-MODEL.md` must exist before the space docs ship, since they cross-reference it.

## 🎨 UX & Branding
- Look & feel: an operational console (vrooli-default "Vrooli Operational Console" kit), light + dark, dense data tables and scoreboards; status-color semantics for coverage (now / in-reach / missing) and confidence (authoritative / partial / sketch).
- Accessibility: WCAG AA; full keyboard navigation; axe-clean; preserve the template's i18n + accessibility seams.
- Voice & messaging: honest and measured; never imply false completeness; always pair a coverage number with its denominator-confidence; "surfaces, does not decide."
- Branding hooks: vrooli-default operational console; keep the seeded PWA manifest + icons valid until real product branding exists.

## 📎 Appendix
- The three space-definition documents: `search-hub/docs/spaces/answer-space.md`, `test-genie/docs/spaces/validate-space.md`, `prompt-manager/docs/spaces/guide-space.md`.
- Convergence doctrine: `docs/agent-system/TEMPLATE_CONVERGENCE_LOOP.md`, `REFERENCE_PATTERN_FITNESS.md`, `REFERENCE_SCENARIOS.md`.
- Canonical model: `docs/concepts/COVERAGE-MODEL.md`.
