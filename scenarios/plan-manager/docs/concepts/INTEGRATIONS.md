# Integrations — Plan Manager

## Purpose Of This Document

The dependency contract for plan-manager: which Vrooli resources and scenarios it
composes, which third-party services it touches, and how each failure degrades.
The governing principle is **compose, don't own** — plan-manager stands on
existing substrate (code references, staleness, baselines, validation) rather than
re-implementing it, and every dependency is soft and degrades gracefully.

## Dependency Inventory

| Dependency | Kind | Purpose | Hard/Soft |
|---|---|---|---|
| `~/.vrooli` home store (SQLite) | Resource | Durable plan persistence (server-independent) | Required |
| runtime-home `plans` entry | Resource | Durable rendered markdown mirrors for file-addressable operator workflows | Soft repairable |
| `search-hub` | Scenario | Reference **discovery** at authoring — the Answer projection; hits routed by locator shape into reviewable `[CODE:]/[DOC:]/[REQ:]` candidates | Soft |
| `code-facts` | Scenario | Resolve `[CODE:]`/`[REQ:]` references at **validation** where its current surface can provide evidence | Soft |
| `git-control-tower` | Scenario | Producer-owned collection capture, wait, extension, and diff; Plan Manager reads typed state | Soft |
| `cli-health` | Scenario | Validate authored `cli:` command references in plans without executing them | Soft |
| git / freshness engine | Platform | Per-reference drift is git-sourced today; freshness engine remains scenario-artifact scoped | Soft |
| `test-genie` / `scenario-validation` | Scenario | Validation results consumed (not owned) | Soft |
| `prompt-manager` | Scenario | Relevant-context skill/action discovery for authoring and setup guidance | Soft |
| `meta-optimization-manager` | Scenario | Velocity signal sink (trials) | Soft |
| `agent-manager` | Scenario | Run-id attribution contract; owns prose-handoff capture | Soft |
| `scenario-qa` | Scenario | Downstream sink for `log` **bug_report** entries (issue tracker) | Soft |
| `swarm-manager` | Scenario | Downstream sink for `log` **record** entries (`records create`) | Soft |

Future consumers to be **inverted** (they will depend on plan-manager, not the
reverse): `swarm-manager` `phased-plan-drain`. Project hygiene plan checks are
already a Plan Manager consumer: hygiene calls `ReconcilePlans`. Root
`vrooli plans` has been retired; plan lifecycle and inspection now use the
`plan-manager` CLI directly.

**Boundary vocabulary alignment (consumer-inversion contract).** Plan Manager's
change boundary deliberately uses Swarm Manager's `acceptance_allow` /
`acceptance_deny` field names (Swarm Manager's backlog model explicitly rejects a
generic `scope` field). This means a future Swarm Manager → Plan Manager inversion
can pass a backlog item's `acceptance_allow`/`acceptance_deny` **directly** into a
Plan Manager `change_boundary` with no vocabulary translation. The two derive the
same affected-scenario set from the same globs (`scenarios/<name>/...`). The shared
glob/scenario logic is duplicated minimally today (`api/internal/planmodel/boundary.go`
mirrors `swarm-manager/internal/pathutil`); extracting a shared package is a
follow-up once a second cross-scenario consumer needs it.

## Vrooli Resources

- **SQLite home store** — the only required resource. Rooted at `~/.vrooli` (see
  [`DATA.md`](DATA.md)) so plans persist independent of the server process. No
  heavy resources (no Ollama/Qdrant) — plan-manager aggregates structured data and
  composes other scenarios' typed CLI/RPC; it does not embed or search.
- **Runtime-home plans mirror directory** — durable markdown projections under
  the repo-contract `plans` entry. These files are operator-facing and backup
  material, but repairable from SQLite; a missing/stale mirror degrades reads
  until Plan Manager regenerates it.
- **Plan Manager CLI** — the direct control surface for plan lifecycle,
  inspection, import, archive, authoring, and render workflows.
- **Root `vrooli hygiene` plans provider** — a consumer of `ReconcilePlans`.
  It reports Plan Manager reconcile outcomes separately from applied fixes so
  no-op results (`skipped_duplicate`, `already_canonical`) are not counted as
  mutations. Static scan fallback is advisory only and is used only when Plan
  Manager is unavailable or times out.

## Scenario Dependencies

All soft; each is reached through a seam and degrades to a marked gap rather than
failing the flow:

- **search-hub** — direct authoring discovery for records, docs, skills, and
  code/requirement context. Plan Manager recommends
  `search-hub query "<intent>" --type record,doc,skill`, but does not mirror the
  output into candidate state. The agent/operator inspects native confidence and
  attribution, then submits durable `[CODE:]/[DOC:]/[REQ:]` locators or context
  items. If down/empty: use manual locator entry or `NO_CODE_REFS:` (never a
  fabricated reference).
- **code-facts** — resolves reference evidence at validation where its current
  surface can express the locator. If down: references remain recorded; validation
  falls back to filesystem resolution for CODE/DOC refs and marks unresolved gaps
  honestly.
- **git-control-tower** — owns collection capture/diff operation lifecycle at
  execution start and validation/DoD. Authoring records intent only; the runner
  renders exact GCT start commands, GCT owns native wait/recovery/Agent Manager
  parking, and Plan Manager performs a single typed `baseline-sync` or
  `validate sync` read. An incomplete collection is a visible gate, never a pass.
- **cli-health** — validates authored `cli:` marked references in plan/phase
  text. If down: command validation reports UNKNOWN and never fabricates a pass.
  Plan Manager owns plan policy and authoring feedback; CLI Health owns command
  existence, argument-shape confidence, suggestions, and guidance.
- **git / freshness engine** — Phase 0 substrate recon found `vrooli scenario
  freshness <scenario> --json` reports scenario-artifact staleness, not
  per-reference code drift. v1 therefore uses filesystem existence as the floor
  and `git diff --numstat <anchor> -- <ref>` for LIGHTLY_STALE refinement when an
  anchor SHA is available. If that refinement is unavailable: present references
  remain FRESH and missing references are still DEFINITELY_STALE.
- **test-genie / scenario-validation** — Test Genie owns its native run waits;
  Plan Manager records supplied run identities and synchronizes terminal typed
  snapshots without parsing output or re-implementing project validation.
- **prompt-manager / search-hub / cli-health** — relevant-context candidate
  discovery for authoring and setup guidance. Prompt Manager owns the curated
  skill/action discovery contract; Search Hub contributes broad recall candidates
  across records/docs/skills; CLI Health contributes command-surface discovery.
  This Guide/setup flow is distinct from the search-hub Answer projection that
  feeds references. Authoring stores discovered setup in a curated discovery
  batch; the author applies the batch by taking useful shortlist items and
  sweeping the rest. If a source is down: the batch carries degraded probe notes
  and leaves the author free to supply context explicitly.
- **meta-optimization-manager** — velocity sink. If down: velocity is retained
  locally and emit is retried/skipped; no flow blocks.
- **agent-manager** — provides the run-id attribution contract
  (the verified identity-token run id) used to dedup handoff actions. **Owns the prose
  final-handoff capture** (it has the transcript); plan-manager links to it by
  reference but never reads transcripts itself.
- **scenario-qa / swarm-manager** — downstream sinks for `log` `bug_report` and
  `record` entries. See [Downstream log forwarding](#downstream-log-forwarding)
  below.

### Downstream log forwarding

The `log` domain owns downstream submission of bug reports and reusable records
**internally**, behind seams (`BugReporter` → scenario-qa, `RecordWriter` →
`swarm-manager records create`). This is a hard contract: an agent **must never
be told to run an external scenario CLI** (`scenario-qa`, `swarm-manager`, or a
bug-filing skill) from the plan workflow — it records the entry through
`plan-manager log bug-add` / `log record-add`, and Plan Manager forwards it.

Production wires live local HTTP adapters behind those seams. Bug reports are
translated into Prompt Manager knowledge writes for `team:scenario-qa` under
`bug-inbox/code-defect/<slug>`, with report-bug attribution and Plan Manager
provenance in the payload. Records are translated into Swarm Manager
`/api/v1/records` creates with a deterministic `[planlog-entry:<id>]` marker in
the trigger and Plan Manager provenance in the narrative fields. Both adapters
lookup by deterministic provenance before creating, so `plan-manager log sync
<id>` can retry without duplicating downstream artifacts.

Downstream failure is **never fatal**: discovery/network/5xx unavailability
leaves the local entry `pending`, downstream rejection leaves it `sync_failed`,
the state surfaces in list/status summaries, and the entry is retried with
`plan-manager log sync <id>` once the downstream is reachable. Explicit pending
stubs remain available as degraded/test fallbacks.

### The handoff ownership split

plan-manager owns the **structured, canonical** handoff (assembled from in-flow
captured state). The agent's free-text **prose** final handoff is a catch-all for
whatever the agent did not route through a command; capturing it requires reading
the run transcript, which the orchestration layer owns:

```
agent-manager run (owns transcript) ──▶ swarm-manager operating mode (chains loops)
        └─ detect + capture prose handoff, reconcile via run-id attribution,
           optionally one bounded local-model diff vs plan-manager's canonical
           record → operator-triaged suggestions, pass to the next loop
plan-manager: structured handoff only — linked to the plan/phase by reference
```

## Third-Party Services

None. plan-manager makes no external network calls. Any local-model extraction
used at the orchestration layer runs through existing Vrooli runners
(agent-manager), not a direct third-party API in this scenario.

## Failure Modes

- **A composed scenario is down** → that input degrades to a marked gap; the flow
  continues and surfaces the gap honestly (never a false "validated").
- **Home store unavailable** → reads/writes fail loudly; this is the one required
  dependency. The store is process-independent, so a downed API server does not
  count as "store unavailable".
- **Rendered mirror unavailable/stale** → render/get surfaces degraded metadata
  and repairs from SQLite when possible. The stale file is never parsed back into
  the canonical structured plan.
- **Velocity sink down** → velocity retained locally; emit retried later.
- **Downstream bug/record sink down or unwired** → the `log` entry persists
  locally with `sync_status` `pending`/`sync_failed`; never blocks execution.
  Retried via `plan-manager log sync`.
- **Attribution missing** (non-agent run) → handoff dedup falls back to
  content-based checks; no double-file guarantee weakens to best-effort.

## Cross-References

- [`DATA.md`](DATA.md) — the home store and data ownership
- [`DOMAINS.md`](DOMAINS.md) — which domain consumes each dependency
- [`ARCHITECTURE.md`](ARCHITECTURE.md) — composition boundaries
- [`../reference/configuration.md`](../reference/configuration.md) — dependency configuration
- [`../../PRD.md`](../../PRD.md) — dependencies & launch plan
