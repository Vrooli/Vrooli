# Product Requirements Document (PRD)

> **Template Version**: 2.0
> **Canonical Reference**: `/scenarios/business-health/docs/reference/canonical-prd-template.md`
> **Validation**: Enforced by `business-health` (the test-genie `business` phase)
> **Policy**: Generated once and treated as read-only (checkboxes may auto-update)
>
> **Vision rewrite — 2026-05-18**: This PRD replaces the original 2026-03-11 PRD.
> The earlier design ("Skills are connected, not executed; validation evaluates
> hand-authored declarative expectations") has been retired. The new model executes
> each skill against a template-pristine golden via agent-manager and treats the
> resulting sandbox diff as the validation signal. See the Appendix → *Why the
> vision changed* for the full rationale.

## 🎯 Overview

- **Purpose**: Validate the coherence of Vrooli's development toolchain — steer
  skills, scenario templates, and development helper tools — by executing them
  against template-pristine "golden" scenarios and comparing the result to an
  expected-diff manifest. A non-empty unexpected diff is a signal: either the
  template is missing something the skill expected (template gap) or the skill
  did something wrong (skill bug). Either way, the toolchain is incoherent and
  someone needs to fix it.
- **Primary users/verticals**:
  - prompt-manager's meta-optimization team (primary consumer — uses results
    to prioritize template improvements and skill bug fixes)
  - Template maintainers (use results to know when a template is missing
    structure that skills expect)
  - Skill authors (use results to verify their skill no-ops on a pristine
    scenario, as a coherence check)
  - Ecosystem-manager (validates the development helper tools it relies on
    still produce sensible output)
- **Deployment surfaces**: Connect-RPC API, React UI, CLI
- **Value promise**:
  - Catch template/skill incoherence before it costs hours of agent iteration
    on real scenarios
  - Measure skill maturity continuously (duration + token cost trending down
    over time = skill is compressing toward CLI-wrapper form)
  - Surface template gaps that are silently being patched up by skills, so the
    template can absorb them and stop paying the per-skill cost
  - Validate that scenario-auditor, test-genie, and scenario-completeness-scoring
    produce sensible output on known-good code — if they don't, the tool is
    wrong, not the scenario

## 🎯 Operational Targets

### 🔴 P0 – Must ship for viability

- [ ] OT-P0-001 | Golden Registry & Regeneration | Maintain a registry of generated goldens (one per template contract in `templates/scenarios/*`). Each entry records: slug, template id, pinned template version, generation metadata, and stable logical root. Validation runs materialize a fresh generated path outside `scenarios/` by default and preserve `golden_slug` for manifests and records.
- [ ] OT-P0-002 | Skill Catalog Sync | Pull the steer-skill catalog from prompt-manager's API. Record each skill's id, current version, and content hash. Refresh on demand; surface drift since last sync.
- [ ] OT-P0-003 | Expected-Diff Manifest | Per `(skill, golden)` tuple, store a manifest describing what the skill is *allowed* to change when run against the golden. Manifest is path-globbed with optional content rules; supports `*` wildcard for "anything goes here." Each manifest is version-pinned to a template version + skill version pair; either changing invalidates the pinning.
- [ ] OT-P0-004 | Skill Validation Run + Diff Evaluation | Given a `(skill, golden)` tuple: spawn an agent-manager run executing the skill against the golden in sandbox mode; await completion; read the sandbox diff; evaluate the diff against the manifest. Produce a verdict: `pass` (diff fits manifest), `unexpected-mutation` (diff exceeds manifest), or `run-failure` (agent-manager run didn't complete cleanly).
- [ ] OT-P0-005 | Tooling Baseline Validation | Given a `(tool, golden)` tuple where tool ∈ {scenario-auditor, test-genie, scenario-completeness-scoring, others as they emerge}: invoke the tool against the golden via its CLI; capture stdout/exit; evaluate against a per-tool expectation (e.g., "zero auditor violations except orientation noise", "test-genie comprehensive preset passes", "completeness score ≥ 96"). Produce a verdict.
- [ ] OT-P0-006 | Validation Record Storage | Persist every validation run (skill or tool) with: tuple identity, timestamps, agent-manager run id (if skill), duration, tokens used, cost estimate, verdict, diff hash, manifest version pinning at run time. Records are append-only and form the basis for trend analysis (P1).
- [ ] OT-P0-007 | Staleness Tracking | Track per-manifest version pinning (template version + skill version). When either drifts ahead of the manifest's pinning, mark the manifest stale. Template-version drift stales all manifests for that golden; skill-version drift stales only that skill's manifest. Stale-ness surfaces as a UI badge and a CLI `--force`/`--yes` requirement; can be cleared without re-running (operator asserts "I've reviewed; not stale anymore").
- [ ] OT-P0-008 | Validation Report API | Connect-RPC service exposing: list goldens, list skills, list `(skill, golden)` manifests, latest verdict per tuple, full run history per tuple, current stale flags. All read-only at this OT (writes covered by 001–007).
- [ ] OT-P0-009 | CLI Interface | Connect-RPC client commands covering the full P0 surface: golden CRUD + regenerate, skill catalog sync, manifest CRUD, validation run trigger (skill + tool), verdict query, staleness ops.
- [ ] OT-P0-010 | UI Dashboard | Per-golden view: grid of `(skill × verdict)` and `(tool × verdict)` with stale badges; click-through to last-run details (manifest, diff, agent-manager run id with link, duration, tokens, cost). Lists all goldens on the index page.

### 🟠 P1 – Should have post-launch

- [ ] OT-P1-001 | Template Version Watcher | DTV checks the active template version (from `templates/scenarios/<template>/template.json::version`) on startup and once daily. When a template's version is ahead of the version pinned in a golden's registry entry, mark all manifests for that golden stale and surface a "template bumped" notification.
- [ ] OT-P1-002 | Skill Maturity Score | Per skill, compute a maturity score from rolling duration + token-cost trends across validation runs. Skills with declining cost over time score higher; skills with rising cost or persistent unexpected-mutation verdicts score lower. Surface in the dashboard.
- [ ] OT-P1-003 | Manifest Convergence Tracking | Manifests can declare a `convergence_target: empty_diff` annotation indicating "this skill should eventually no-op on this golden." Track how many such manifests have actually reached empty-diff and surface "X of N converged" as a top-level scenario-health metric.
- [ ] OT-P1-004 | Run History & Trend Detection | Detect regressions (previously-passing tuple now failing) and tightening (manifest wildcard replaced with explicit path list, then the path list shrinks). Surface both as timeline events in the UI.
- [ ] OT-P1-005 | Coverage Map | For each golden, produce a path-tree map of which skills' last validation runs touched which files/folders. Reveals overlap (multiple skills touching the same path = potential coordination need) and gaps (template paths no skill exercises).
- [ ] OT-P1-006 | Bulk Re-Validation | "Re-run all stale tuples for golden X" and "re-run skill Y across all goldens" as single CLI/UI operations, with concurrency limits to avoid agent-manager queue saturation.

### 🟢 P2 – Future / expansion

- [ ] OT-P2-001 | Cross-Golden Consistency | When the same skill is validated against multiple goldens (different templates), verify the verdicts and mutation patterns are coherent — e.g., a skill that no-ops on react-vite golden but mutates the cli-only golden might indicate template-shape divergence.
- [ ] OT-P2-002 | Per-Skill Cost Budget Alerts | Set a token/cost ceiling per skill; alert when a run exceeds historical baseline by N%. Surfaces skill-bug regressions early (skill suddenly doing 10× more work than usual).
- [ ] OT-P2-003 | Webhook Notifications | Notify on verdict regressions, new stale flags, or convergence milestones via configurable webhooks.
- [ ] OT-P2-004 | Manifest Auto-Generation from Observed Diffs | When a manifest doesn't exist for a `(skill, golden)` tuple, run the validation and propose a manifest seeded from the observed diff (operator reviews + accepts). Bootstraps the manifest set as new skills are added.
- [ ] OT-P2-005 | Input/Output Token Breakdown | Surface input vs output token split per run (requires agent-manager to expose split in `RunSummary` — currently only combined `tokens_used` is exposed; tracked as upstream dependency).

## 🧱 Tech Direction Snapshot

- **Preferred stacks / frameworks**:
  - API: Go + Connect-RPC over proto (template default, per `feedback_proto_connect_always`)
  - UI: React + TypeScript + Vite (template default)
  - CLI: Go, generated Connect-RPC client
  - Storage: SQLite (template default; per-domain `internal/<x>/{sqlite,schema}.{go,sql}` modules)
- **Data + storage expectations**:
  - Goldens registry, skill catalog snapshot, manifests, validation records all in SQLite
  - Skill content and versions fetched from prompt-manager API; only the hash + version are mirrored locally
  - Validation records are append-only and retained indefinitely (small footprint; enables trend analysis)
- **Integration strategy**:
  - **prompt-manager**: read-only API consumer (skill catalog, versions, content hashes). DTV does not call prompt-manager for writes.
  - **agent-manager**: spawn sandboxed runs via Connect-RPC; await completion; read sandbox diff and per-run stats (duration, tokens, cost). Sandbox is the diff capture mechanism — agent-manager already runs in accountability sandbox mode by default (per `project_sandbox_purpose_accountability`).
  - **Development helper tools** (scenario-auditor, test-genie, scenario-completeness-scoring): invoked via their CLIs; capture stdout/exit. No API coupling.
  - **Template generator**: invoked via `vrooli scenario generate` (or equivalent) when materializing or refreshing a generated golden.
- **Non-goals / guardrails**:
  - Does NOT statically analyze skill prose for cross-skill conflicts (out of scope; that's a prompt-manager / skill-authoring concern)
  - Does NOT host a fleet-wide scenario health dashboard (out of scope; that's a separate scenario consuming DTV's API)
  - Does NOT enforce a deploy gate (out of scope; results are advisory until enough data exists to define meaningful thresholds)
  - Does NOT maintain committed golden source trees; normal validation materializes pristine template output into managed generated paths
  - Does NOT execute non-steer skills (workflow skills like `plan-skill-discovery`, `idea-workshop`, `deployment-coordinator` are out of scope — validation is for skills that operate on scenario shape)
  - Does NOT manage skills, templates, or scenarios — it observes them

## 🤝 Dependencies & Launch Plan

- **Required resources**:
  - SQLite (embedded; no separate process)
- **Scenario dependencies**:
  - prompt-manager (API consumer: skill catalog + versions)
  - agent-manager (Connect-RPC consumer: spawn sandboxed runs, fetch run summary + sandbox diff)
  - scenario-auditor (CLI consumer)
  - test-genie (CLI consumer)
  - scenario-completeness-scoring (CLI consumer)
- **Template dependencies**:
  - `templates/scenarios/react-vite` — source template for first generated golden (`reference-react-vite`)
  - Future templates as they mature
- **Operational risks**:
  - Each skill validation run consumes LLM tokens via agent-manager. Cost is bounded by manual triggering, but a "validate all" operation across many skills × many goldens can be expensive. P0-009 / P1-006 must enforce concurrency limits and surface estimated cost before bulk runs.
  - agent-manager's `RunSummary.tokens_used` is combined input+output today; granular cost analysis (P2-005) waits on an upstream enhancement.
  - Golden slugs are durable, but their physical substrates are generated per run by default. Debug retention must be explicit so validation does not pollute `scenarios/` or create a second scenario-sized maintenance surface.
- **Launch sequencing**:
  1. P0 (this OT block): single generated golden (`reference-react-vite`), full skill + tool validation flow end-to-end with CLI and UI
  2. P1: trend analysis, maturity scoring, additional goldens as more templates mature
  3. P2: cost guardrails, cross-golden consistency, auto-manifest seeding

## 🎨 UX & Branding

- **Look & feel**: Developer-tool aesthetic, data-dense. Verdict grid is the dominant element — pass/unexpected-mutation/run-failure/stale shown with both color and icon. Drill-down panels show the actual diff and link to the agent-manager run for full context.
- **Accessibility**: WCAG 2.1 AA. Verdict states distinguishable without color (icons + labels).
- **Voice & messaging**: Diagnostic and specific. "Skill `api-steer` mutated 3 files outside its manifest in run abc123: …" beats "Validation failed."
- **Branding hooks**: Toolchain / coherence theme. The visual progression of a manifest from wildcard → explicit allowlist → empty-diff is the product's narrative — surface it prominently.

## 📎 Appendix

### Why the vision changed

The original PRD (2026-03-11) was built around a declarative-expectations model: per `(skill, reference)` pair, hand-author JSON describing what files/structures the skill expects, plus CLI assertions against tool outputs. The motivation for avoiding skill execution was cost ("LLM tokens are expensive") and safety ("agents could modify references").

Two things changed:

1. **Goldens are now pristine template output.** When a golden is just `vrooli scenario create` output (no manual buildout, regenerated on template bump), "agents modifying it" is no longer a problem — the diff from pristine *is* the signal.
2. **agent-manager's sandbox mode** provides per-run diff traceability cheaply (per `project_sandbox_purpose_accountability` — sandbox's purpose is accountability, not isolation), making "run the skill, read the diff" a viable validation primitive.

Once execution is on the table, hand-authored declarative expectations become a strictly weaker signal than the actual diff. The whole `Structural Expectation` + `CLI Assertion` apparatus (~60% of the old PRD) was retired.

What survived from the old PRD: the reference-registry concept (now "golden registry"), the version-pinning concept (now per-manifest pinning to template + skill versions), and the tooling-baseline idea (now OT-P0-005, largely unchanged).

### Skill exemption seeds (initial expected-mutator manifest set)

Most steer skills should no-op on a maximally-mature golden. The exceptions — skills whose explicit job is to mutate scenario content — get permissive manifests at launch and tighten over time:

- **`progress`** — fills scenario-specific PRD checkboxes and operational-target status. Manifest expects mutations to `PRD.md` and `requirements/*/module.json` checkbox/status fields, nothing else.
- **`bundle-integration-steer`** — opt-in; only relevant for scenarios that ship as gated bundle members. Manifest excluded by default; opt-in per golden when relevant.
- **`progress-continuity-interruption-resilience`** — adds work-resumption mechanisms. Permissive manifest at launch; tighten as the resumption pattern stabilizes.
- **All other steer skills** — empty-diff manifest as the convergence target. Some currently mutate (e.g., `refactor`, `polish`, `test`, `performance`, `ux` mutate when the template isn't yet at maximum maturity). For those, start with a wildcard manifest annotated `convergence_target: empty_diff` so the dashboard can track them moving toward zero.

The `skill-authoring` skill is **not** a steer skill and is out of scope.

### Validation flow

```
operator triggers validation for (skill X, golden Y)
        │
        ▼
DTV checks manifest exists + is not stale (pinned versions match current)
        │
        ▼
DTV calls agent-manager: spawn sandboxed run with skill X against golden Y path
        │
        ▼
agent-manager runs the agent → sandbox accumulates diff
        │
        ▼
DTV polls / awaits run completion
        │
        ▼
DTV fetches run summary (duration, tokens, cost) + sandbox diff
        │
        ▼
DTV evaluates diff against manifest paths/wildcards/content rules
        │
        ▼
DTV persists ValidationRecord with verdict, metrics, diff hash
        │
        ▼
verdict surfaces in CLI output + UI dashboard
```

### Verdict states

| Verdict | Meaning | Operator action |
|---|---|---|
| `pass` | Diff fits within manifest's allowed paths/rules | None — keep going |
| `unexpected-mutation` | Diff includes paths/content the manifest doesn't allow | Investigate: template gap OR skill bug. Either fix the template (and bump version → regenerate golden) or fix the skill |
| `run-failure` | agent-manager run didn't complete cleanly (timeout, error, crash) | Investigate the underlying run; not necessarily a skill or template issue |
| `stale` | Manifest's pinned template-version or skill-version drifted | Re-run validation (versions auto-update on next run) OR clear the stale flag manually if confident |

### Ecosystem integration flow

```
templates/scenarios/<template>  ──┐
                                  │  template version bump
                                  ▼
                  DTV: template-version watcher flags goldens stale
                                  │
                                  ▼
                  operator: regenerate golden(s), commit, clear stale
                                  │
                                  ▼
prompt-manager (skill catalog) ──►│
                                  ▼
                  DTV: trigger validation runs (manual, scoped, or bulk)
                                  │
                                  ▼
                  agent-manager: sandboxed runs, return diff + stats
                                  │
                                  ▼
                  DTV: verdicts, trends, convergence metrics
                                  │
                  ┌───────────────┼───────────────────┐
                  ▼               ▼                   ▼
        meta-optimization   template          skill authors
              team        maintainers        (fix skill bugs,
        (prioritize        (absorb gaps        compress toward
         improvements)      into template)      CLI wrappers)
```
