## Steer focus: Measures Adoption

Prioritize **making `scenarios/{{TARGET}}/`'s analytical questions answerable as declared, typed, parameterized measures** on `packages/measures-go`, so `search-hub` can match a natural-language question ("how many backlog items closed this week"), resolve its parameters deterministically, and — for safe read-only measures — return the computed answer with provenance.

Your goal is to make `measures-health validate scenario {{TARGET}}` report the target's provider-owned current and next maturity as unblocked — not to invent new business logic. Measures are a declaration + thin serve layer over computations the scenario already performs.

Required reading:
- **`path:packages/measures-go/README.md`** — the contract API: the three-source `MeasureDeclaration`, the deterministic `time_window` resolver, the serve `Registry`, the auto-exec `Gate`, the `LLMExtractor`/`MeasureComposer` seams, the CQRS substrate, and the contract invariants. This skill is the *judgment* layer; the README is the *API*.
- `path:docs/concepts/MEASURES.md` — the mental model (measure = typed parameterized query; domains/statefulness/waivers; the answer-vs-resolve gate).
- `prompt-manager skill read cli-steer` — the manifest carries the curated half of a measure (`measure` block + `governance`) on the same proto-first binding every command uses.
- `prompt-manager skill read knowledge-observatory-tools` — read/update the scenario's durable docs (§7) through the canonical docs CLI.

Optional reading:
- `prompt-manager skill read api-steer` — the Connect-RPC binding a measure points at; param types/enums/bounds come from *its* request message.
- `prompt-manager skill read storage-steer` — when a measure adopts the event-sourced read-model substrate.
- `prompt-manager skill read scenario-maturity-ladder` — how the EM ladder rung this dimension feeds is read and sized.

Read first when present (prior findings — continue, don't restart):
- `scenarios/{{TARGET}}/docs/internal/SEAMS.md` — the measures-serve boundary, if already declared.
- `scenarios/{{TARGET}}/docs/internal/PROBLEMS.md` — deferred measure work (e.g. a `/stats` migration in flight).

> Universal authoring/quality bars (intent statement, convergence patterns, anti-gaming framing, the agent memory loop) are canon in `path:docs/agent-system/SKILL_AUTHORING.md` and are not restated here.

---

### 1. Scope Boundaries

**In scope** (anchored to `scenarios/{{TARGET}}/`):
- declaring measures: the manifest `measure` block + `governance` effect, the proto binding, the `measures-go` serve `Registry`.
- choosing a measure's substrate (plain SQL aggregate / live call / CQRS event-sourced read-model).
- classifying which stateful domains are expected, covered, or waived; writing waiver reasons.
- refactoring an existing monolithic `/stats` path into granular measures and deleting it (§6).
- recording the measures-serve seam + deferred work in `docs/internal/SEAMS.md` / `PROBLEMS.md`.

**Out of scope** (hand off):
- the manifest command surface / CLI ergonomics → `cli-steer`.
- the Connect-RPC request/response contract a measure binds to → `api-steer`.
- the event-log / repository storage architecture itself → `storage-steer`.
- the central index, the search-hub provider, and routing → owned by the `measures-health` scenario; `{{TARGET}}` only *serves* measures and is harvested.
- changing *what* the scenario computes, or adding features → not a measures concern.

---

### 2. Measures, in one screen

A **measure** is a named, typed, parameterized analytical query (`<domain>.<name>`, e.g. `backlog.completed`), **assembled from three sources joined on the manifest `binding`** (see README "The model"):

```
MeasureDeclaration = { manifest `measure` block }   ← curated prose: intent, questions[], result, defaults
                   ⊕ { manifest `governance` }       ← effect (read|write|destructive), run_eligible
                   ⊕ { proto request-message schema } ← param types/enums/bounds (NEVER duplicated in manifest)
```

Two contract rules drive everything below; the rest are in the README's "Contract invariants":
- **A required param that can't be resolved abstains** (`needs[]`) — never a guess. Wrong numbers are worse than no number.
- **`write`/`destructive` measures never auto-execute**, at any confidence. The auto-exec `Gate` fires only on `read` + `run_eligible` + confidence ≥ θ.

Measures are **not** prompt-manager *actions* (agent-discovered command intents) and **not** operational/Prometheus telemetry — they are a distinct analytical-query class. Do not merge them.

---

### 3. Provider-Owned Maturity

Run the provider CLI before manual judgment. The provider's default human output is the single source of truth for current local maturity, next level, blockers, global impact grouping, and recommended skill IDs:

```bash
measures-health validate scenario {{TARGET}}
measures-health validate scenario {{TARGET}} --probe
```

Use this skill to interpret and fix the findings the CLI reports. Do not restate or override the provider's `.vrooli/maturity.json` ladder in skill prose; if the ladder looks wrong, fix `measures-health` or its maturity spec.

---

### 4. Decide what each domain needs

**First, detect the scenario's mode** — it decides where stateful domains are declared:

- **Conformant** — `packages/proto/schemas/{{TARGET}}/v1/domain/` exists (screaming architecture). That folder is the **authoritative SSOT**: EXPECTED stateful domains derive from its `*.proto` files. Declaring a *new* stateful domain via the manifest `measures.domains[]` here is an **ERROR** (`measures.illegal-domain-declaration`) — add a `v1/domain/<d>.proto` instead. (Down-grade overrides marking a folder domain `stateful:false`, and `measures.omitted[]` waivers, stay legal.)
- **Fallback** — no `v1/domain/` folder (a flat `v1/<entity>/`-layout / template scenario). You **must** declare stateful domains via the `measures.domains[]` crutch, and the scenario carries a standing `measures.architecture-fallback` **INFO** advisory nudging it to adopt `v1/domain/`. The crutch is transitional, not a parallel SSOT — plan the upgrade.

Then walk this table **per domain**. It produces a deterministic classification (expected/covered/waived) and a substrate choice.

| Question | If YES | If NO |
|---|---|---|
| Is the domain a persisted entity type? (conformant: a `v1/domain/<d>.proto`; fallback: a `measures.domains[]` entry marked `stateful:true`.) | Stateful candidate → continue. | Not expected (config/structural). Conformant: a folder proto that isn't countable → down-grade it `stateful:false` with a reason. |
| Do its rows accumulate countable / historical value over time? | Must be **covered** → choose a substrate below. | **Waive** in `measures.omitted[]` with a reason; stop. (Ephemeral inbox, live derived feed.) |
| Do many measures fold over one shared event stream? | CQRS event-sourced **read-model** substrate (`EventLog`+`Projection`+`ReadModel`). | Continue. |
| Does a single `COUNT(…) WHERE ts ∈ [from,to)` answer it? | Plain **SQL aggregate** in the compute func. | A **live external call** in the compute func. |

> **Substrate cross-check (anti-honor-system, layout-agnostic).** Independently of the proto/manifest signal, `measures-health` detects persisted countable entities from evidence — a SQL `CREATE TABLE` with a `created_at` column. A detected entity that is neither covered nor waived raises `measures.undeclared-substrate` (**WARNING**). So a fallback scenario can't simply *omit* a real entity from `measures.domains[]` to dodge the gate — declare it or waive it.

Then, per answerable **question family**, author one measure = one binding. If you find yourself adding a `category`/`kind` switch to one endpoint, you are rebuilding the monolith — split it. Phrase 3+ natural, varied `questions[]`: the central index embeds them, so they are what semantic match scores against.

> **Anti-gaming (measures-specific).** A waiver added to hide a domain that *should* be covered is suppression-shaped → EM `gameguard` zeroes the credit. A measure declared without a working endpoint is caught by `--probe` as ERROR. Waive only the genuinely non-measurable, with a real reason; serve before you declare. (General anti-gaming framing: canon.)

---

### 5. Author + serve

- **Bind + declare.** Put the curated prose (`intent`, `questions[]`, `result`, param `default`/`values_source`) in the manifest `measure` block; put types/enums/bounds in the **proto request message** (never duplicate them). Use the canonical `time_window` param type for any date so resolution is deterministic. Worked example: `path:templates/scenarios/react-vite/cli/manifest.json` (the `notes count` command). Write `governance.effect` honestly — the gate trusts it.
- **Serve via one `Registry` + one spec table.** Register each declaration with its compute func; if you also expose a typed Connect service, back both surfaces from the **same `computeFn`** so a measure and its RPC can never report different numbers (the swarm-manager dogfood pattern). The registry name **must equal** the manifest `<domain>.<command>` or `--probe`'s `/execute` 404s. Provenance (`executed_query` + `computed_at`) is mandatory; the helper stamps it. Resolve windows from the request's explicit `now`/`loc` — never an ambient clock.

See README "Serving measures" + "The CQRS substrate" for the exact wiring.

---

### 6. Refactor away a monolith (greenfield — delete it)

If `{{TARGET}}` already serves a monolithic `/stats` blob (`GET /stats?category=…`, a giant `StatsResponse`, a CLI `stats` group), measures **replace** it — there is no flag and no shim:

1. expand the measure set to **parity** with what the blob served (counts first, then `table`/`series` + rate measures, each its own binding + block);
2. migrate the UI / consumers onto `/measures/execute` (or the central index);
3. **delete** the old `internal/stats/`, the `GET /stats` route, and the CLI `stats` group — verify with `git grep`;
4. add a `measures` CLI group if consumers need one.

A *purely additive* measures layer (new measures reading the domain's data directly, blob left standing) is a legitimate **intermediate** state when the blob serves richer shapes than your first measures — but it is not done. Record the owed deletion + migration in `PROBLEMS.md`; do not let it become permanent dead weight.

---

### 7. Memory & durable-doc output anchors

Per the agent memory loop (canon), findings accumulate in the scenario's durable docs — **never a standalone `*_AUDIT.md`**:

- **`scenarios/{{TARGET}}/docs/internal/SEAMS.md`** — the "Measures Serve Boundary" seam: the `Registry` is the single point where a declaration meets its computation; the test double is a fake compute func.
- **`scenarios/{{TARGET}}/docs/internal/PROBLEMS.md`** — deferred measure work (a `/stats` migration in flight, a domain stuck below full tier and why, a waiver that needs revisiting).

Write the provider-reported current/next maturity and blockers so the next agent continues rather than re-discovers.

---

### 8. Output expectations

You **may**: add `measure` blocks + `governance` to `cli/manifest.json`; add a `measures-go` serve `Registry` + compute funcs; add measure RPCs/bindings; delete an obsolete `/stats` path (§6); update `SEAMS.md`/`PROBLEMS.md`.

You **must**: keep param types in proto (not the manifest); abstain (`needs[]`) on unresolved required params; keep `write`/`destructive` non-auto-executing; stamp provenance; reach `probePassed: true` before claiming a domain covered.

You **must NOT**: duplicate param types into the manifest; use a `category` switch in place of N measures; mark a mutating measure `effect: read`; waive a domain that should be covered; read an ambient clock in a compute func.

Per-change ritual: fix all lint/type/test issues in touched files (incl. pre-existing), `vrooli scenario restart {{TARGET}}`, verify `/health` + UI, `gofumpt -w`, then `swarm-manager records create --kind execute`.

---

### 9. Troubleshooting & Edge Cases

- **`--probe` skips / reports not-run.** The scenario wasn't reachable. `vrooli scenario restart {{TARGET}}`, confirm `/health`, re-probe. The central index harvests at boot — if you just added measures, restart `measures-health` before `search-hub query` (L5) returns a hit.
- **`/execute` 404 for a declared measure.** The serve-registry name ≠ the manifest `<domain>.<command>`. Make them identical.
- **`measure.schema_unread` WARNING (degraded).** cli-health couldn't read the proto descriptor; param tiers fall back to manifest-only. Ensure the descriptor exists / `MEASURES_DESCRIPTOR_PATH` is set; non-fatal but blocks full-tier grading. (The shared scanner is `github.com/vrooli/measures-go/manifestscan` — one canonical `Parse`/`Assemble`/`GradeTier`/`DescriptorSchemaReader`.)
- **`measures.illegal-domain-declaration` ERROR (conformant mode).** The scenario has a `v1/domain/` folder, so it's the SSOT — but `measures.domains[]` adds a *new* stateful domain with no proto there. Fix: add `packages/proto/schemas/{{TARGET}}/v1/domain/<d>.proto` and remove the manifest entry. (A down-grade `stateful:false` of an existing folder domain, or an `measures.omitted[]` waiver, is legal.)
- **`measures.architecture-fallback` INFO (fallback mode).** Standing advisory whenever a scenario has no `v1/domain/` folder and derives stateful domains from the `measures.domains[]` crutch (e.g. a `v1/<entity>/`-layout template scenario — the react-vite template does this for `notes`). Non-blocking; the real fix is adopting screaming architecture (`v1/domain/`), after which the override disappears.
- **`measures.undeclared-substrate` WARNING.** The substrate cross-check found a persisted countable entity (a SQL `CREATE TABLE` with `created_at`) that no measure covers and no waiver excuses — regardless of proto layout. Declare a measure for it or waive it in `measures.omitted[]`. (You can't silently leave it out of `measures.domains[]` to dodge the gate.)
- **A scenario with no `cli/manifest.json`** (hand-rolled CLI) must author one as the measures SSOT — `measures-health` reads `scenarios/{{TARGET}}/cli/manifest.json`.
