# Act Space — Programmatic Operation Coverage

> **Provenance.** This denominator was authored inside `meta-optimization-manager` before its owner
> existed, and relocated here verbatim when `program-runtime` was created — denominators live with
> their owner, never with the aggregator. Content below is unchanged from that draft; only the
> relocation banner was replaced by this note. The remaining obligations it names are tracked as
> `PRT-P0-007` (serve this file through `space --projection act --json`), `PRT-P0-008`
> (expose the binding-registry RPC for the live numerator), and `PRT-P1-009` (audit this grid
> against that registry and raise the stated confidence above `SKETCH`). The audit is now live;
> the remaining partial cells retain explicit registry reasons.

> **Model & terminology** — the projection model, status legend, and how coverage (the numerator)
> is computed are defined once in `meta-optimization-manager/docs/concepts/COVERAGE-MODEL.md`.
> This document is the *Act* denominator only.

> **Condition** — every cell that resolves `NOW` here puts its **bindings** into the Condition
> population, and `program-runtime` owes each binding's serving, freshness, and exercise signals as
> declared Measures. A registry that reports a four-digit binding count with no invocation data is
> a supply claim carrying no condition evidence: only invocation distinguishes a binding that is
> genuinely callable from one that merely resolves. Per-binding invocation outcome is the
> highest-leverage unbuilt signal in the model, and it is gated on durable retention of the program
> corpus. The model and the required signals are in
> `meta-optimization-manager/docs/concepts/CONDITION-MODEL.md`.

## Purpose

The **denominator** for the *Act* projection: the space of **operation classes** an agent must be
able to invoke **from inside a program**, without leaving the governed surface.

Act measures **effect** supply. Answer/Validate/Guide measure **knowledge** supply — whether the
project can be explained, verified, and taught. Act asks the orthogonal question: when the agent
knows what to do, *can it actually do it* by calling a typed, governed operation rather than
shelling out to an ungoverned tool.

**This is deliberately not a list of every RPC.** The project has ~334 proto services; enumerating
them here would be an inventory, not a denominator, and would rot on contact. A cell is one
*operation class* — a thing an agent needs to accomplish — which may be served by many RPCs. The
mechanical reality (which services exist, which are manifest-bound, which respond) is the
**numerator**, computed live from `program-runtime`'s binding registry and never stored here.

### What COVERED means here

A cell is `COVERED` only when the operation is reachable as a **typed, manifest-bound, governed
call** — i.e. something a program can invoke with validated arguments and a declared `effect`.
Specifically **not** covered:

- the operation exists only as a locally-implemented CLI command (`binding.kind: "local"`), so
  there is no proto method to bind
- the operation requires shelling out to an ungoverned external tool (`git`, `docker`, `psql`,
  `grep`) — per `prompt-manager/docs/concepts/ACTIONS.md`, those must be wrapped first
- the operation is only reachable by reading or writing files by hand

That strictness is the point. A cell that an agent can *only* accomplish by shelling out is a real
gap in the acting surface even when the underlying capability plainly exists.

## This Space

| | |
|---|---|
| Projection | Act |
| Owner | `program-runtime` (owns the binding registry and audit this extends) |
| Denominator confidence | `PARTIAL` — all 28 cells have been audited against the live binding registry. Cells with an unresolved or external owner retain their conservative authored status and carry the registry's explicit reason; they are not silently counted as covered. |
| Sibling spaces | `search-hub/docs/spaces/answer-space.md`, `test-genie/docs/spaces/validate-space.md`, `prompt-manager/docs/spaces/guide-space.md` |
| Legend | `COVERED` a typed, manifest-bound, governed call serves it today · `PARTIAL` the capability exists but is not (or is not confirmed) programmatically invocable — local-only command, unbound RPC, or requires shelling out · `MISSING` no Vrooli-owned operation exists. **The status vocabulary is closed** — `normalizeStatus` coerces any unrecognized token to `MISSING`, so an unaudited cell must be authored `PARTIAL` (the conservative reading: claims nothing, fabricates nothing) and marked **unaudited** in Notes. Never write `?`, `TBD`, or a blank cell: each silently becomes a fabricated gap. |

## Coverage Grid

| # | Operation | Owner scenario(s) | Status | Notes |
|---|---|---|---|---|
| **Discover** | | | | |
| A1 | Find a capability by intent | `search-hub`, `prompt-manager` | COVERED | The Recall→Discover reflex; both expose Connect services. |
| A2 | Enumerate the fleet (scenarios, resources) with state | `vrooli` project CLI | PARTIAL | `cli/v1/scenario_list.proto` exists; verify the manifest binding vs. a local command. |
| A3 | Read a unit's command contract (commands, args, governance) | `cli-health` | NOW | The 58 manifest-bearing scenarios are checked against the shared proto binding ladder; the live registry supports this contract, while scenarios without manifests remain an explicit fleet coverage boundary. |
| A4 | Resolve a unit's API base URL / port | `api-core/discovery` | PARTIAL | Library-level today, not an invocable operation; a program needs this as a call. |
| **Inspect** | | | | |
| A5 | Read lifecycle status for a unit | `vrooli` project CLI | COVERED | `scenario.status.show` is already a seed Action. |
| A6 | Read logs for a running unit | `vrooli` project CLI | PARTIAL | **Audited.** The owner is a project CLI rather than a manifest-backed scenario; the authored partial status is retained. |
| A7 | Read health / freshness verdicts | `*-health` fleet, `structure-health` | PARTIAL | **Audited.** At least one health binding resolves; the fleet taxonomy remains partial where individual owners are absent. |
| A8 | Read test run results, verdicts, and diffs | `test-genie` | COVERED | `runs.proto` is a bound service; `runs compare` exists. |
| A9 | Read the dependency graph (forward + reverse) | `scenario-dependency-analyzer` | COVERED | `graph.proto`. |
| A10 | Read code facts, symbols, call graph | `code-facts`, `symbol-search`, `go-code-graph`, `typescript-code-graph` | PARTIAL | **Audited.** Some named owners resolve while `symbol-search` has no governed binding; the live result is partial. |
| **Operate** | | | | |
| A11 | Start / stop / restart a unit | `vrooli` project CLI | COVERED | The governed lifecycle path; running binaries directly is forbidden. |
| A12 | Run a test suite and await the verdict | `test-genie` | COVERED | Server-owned runs; `runs wait` is the blocking primitive. |
| A13 | Run setup / build / regenerate artifacts | `vrooli` project CLI | PARTIAL | **Audited.** This is a project-CLI capability without a manifest-backed owner; effect classification remains the governing gap. |
| A14 | Install or govern a dependency | `scenario-dependency-analyzer` | COVERED | The only sanctioned path; raw package managers are forbidden. |
| **Knowledge** | | | | |
| A15 | Query durable memory / work records | `vrooli-memory` | COVERED | |
| A16 | Write a work record, note, or capture | `vrooli-memory`, `swarm-manager` | COVERED | The write side of the learning loop. |
| A17 | Read & write plans, backlog items, goals | `plan-manager`, `swarm-manager` | PARTIAL | Reachable, but `swarm-manager` is not yet reliable enough to depend on. |
| A18 | Read & write requirements / PoRs | `prompt-manager`, per-scenario `requirements/` | PARTIAL | **Audited.** Prompt Manager is present but has no resolved governed binding for this compound owner; filesystem-shaped requirements remain partial. |
| **Delegate & infer** | | | | |
| A19 | Typed inference — classify / extract / judge | `ai-gateway` | NOW | `program-runtime` exposes governed `vrooli.ai.classify`, `vrooli.ai.extract`, and `vrooli.ai.judge` facades over ai-gateway's locally validated inference RPC and catalog roles; the live cell explanation confirms the status. |
| A20 | Spawn a delegated agent run and collect its evidence | `agent-manager` | COVERED | Already consumed programmatically by MoM's `trials` domain. |
| A21 | Read run transcripts, events, and friction findings | `agent-manager` | PARTIAL | Reachable, but ~65% of runs have unknown ownership, so results are not yet trustworthy. |
| **Change** | | | | |
| A22 | Create an isolated workspace over the repo | `workspace-sandbox` | COVERED | Copy-on-write; safety from accidents, not from adversaries. |
| A23 | Apply edits and produce a reviewable diff | `workspace-sandbox` | COVERED | |
| A24 | Promote approved changes to the canonical repo | `workspace-sandbox` | COVERED | Hunk-level approval workflow. |
| A25 | Edit a file in place, outside a sandbox | — | MISSING | **Deliberately missing.** There is no governed in-place edit operation and there should not be one; A22–A24 is the sanctioned path. Recorded so the absence is a stated decision rather than an unnoticed hole. |
| **Measure** | | | | |
| A26 | Read measures for a unit | `measures-health`, per-scenario `measures` | PARTIAL | Declared in `cli-manifest` `Measures`; verify programmatic reachability. |
| A27 | Emit and query platform events | `vrooli-events` | COVERED | Emission is automatic via `api-core/eventbus`; querying is a bound service. |
| A28 | Read the readiness board (coverage / gaps / focus) | `meta-optimization-manager` | COVERED | The board this document feeds. |

## Numerator Contract (for `program-runtime`)

`program-runtime` owns both sides of this projection:

1. **Denominator** — this file, moved to `docs/spaces/act-space.md`, served by
   `api-core/spacecli` as `space --projection act --json`.
2. **Numerator** — `BindingRegistryService.ResolveActCells`, which returns a
   verdict only after checking the live callable registry and binding
   completeness. `meta-optimization-manager` consumes this typed RPC.

## Live audit snapshot — 2026-08-07

The post-repair live registry recomputation reports **20 NOW, 7 IN-REACH, and
1 MISSING** across all 28 Act cells: coverage ratio **0.7142857143**. This is
unchanged from `repair-baseline/coverage-before.json`, so the semantic repair
changed the trustworthiness of the binding census without changing the Act
denominator result. The denominator remains `PARTIAL`: all 28 rows were audited
against the live registry, but external, unbound, or partly represented owners
retain conservative statuses with explicit reasons. `focus next --limit 60`
reports eight Act gaps, exactly matching the seven in-reach cells plus the one
missing cell. Live explanations confirm A3 and A19 as `NOW`.

Suggested live-join rule, mirroring the sibling projections: a cell is `NOW` only when **every**
operation it names resolves to a manifest-bound Connect method whose binding generates cleanly and
whose declared `effect` is satisfiable under the caller's grant. A cell whose operations are
partially bound is `IN-REACH`; a cell that cannot be resolved at all keeps its authored status
rather than being fabricated as `MISSING`.

## Known Limitations

- **The grid is audited but partial.** Every value above has been checked against the live registry.
  Cells whose owners are external, unbound, or only partly represented retain their conservative
  authored status and carry the registry reason. Served confidence is derived from the 28/28 audit
  coverage and reports `PARTIAL`; it does not turn unresolved capabilities into fabricated gaps or
  covered operations.
- **The taxonomy is a first cut.** 28 operation classes across 7 groups is a starting shape. Expect
  splits and merges once real programs are observed — the strongest future signal is telemetry from
  actual programs, which reveals the operations agents *try* to invoke and cannot.
- **`cli/manifest.json` coverage bounds everything.** With 58 of 128 scenarios carrying a manifest,
  a large share of the fleet has no bindable surface at all. Act cannot exceed that ceiling, and
  raising it is likely the highest-leverage single action this projection will surface.
