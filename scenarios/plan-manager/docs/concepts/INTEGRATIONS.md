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
| `search-hub` | Scenario | Reference **discovery** at authoring — the Answer projection; hits routed by locator shape into reviewable `[CODE:]/[DOC:]/[REQ:]` candidates | Soft |
| `code-facts` | Scenario | Resolve `[CODE:]`/`[REQ:]` references at **validation** where its current surface can provide evidence | Soft |
| `git-control-tower` | Scenario | Regression anchor baseline snapshot (captured at **execution start**) + diff (validation/DoD) | Soft |
| `cli-health` | Scenario | Validate authored `cli:` command references in plans without executing them | Soft |
| git / freshness engine | Platform | Per-reference drift is git-sourced today; freshness engine remains scenario-artifact scoped | Soft |
| `test-genie` / `scenario-validation` | Scenario | Validation results consumed (not owned) | Soft |
| `prompt-manager` | Scenario | Relevant-context skill/action discovery for authoring and setup guidance | Soft |
| `meta-optimization-manager` | Scenario | Velocity signal sink (trials) | Soft |
| `agent-manager` | Scenario | Run-id attribution contract; owns prose-handoff capture | Soft |
| `scenario-qa` | Scenario | Downstream sink for `log` **bug_report** entries (issue tracker) | Soft |
| `swarm-manager` | Scenario | Downstream sink for `log` **record** entries (`records create`) | Soft |

Future consumers to be **inverted** (they will depend on plan-manager, not the
reverse): `swarm-manager` `phased-plan-drain`, project hygiene plan checks, the
`vrooli plans` CLI. Sequenced after standalone proof (OT-P2-002).

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

## Scenario Dependencies

All soft; each is reached through a seam and degrades to a marked gap rather than
failing the flow:

- **search-hub** — the reference **discovery** source at authoring (read-only
  `query --json`, the Answer projection). The wizard sends the rich
  title+scope+approach query, routes the hits by locator shape, and offers the
  `[CODE:]/[DOC:]/[REQ:]` hits as **reviewable** candidates the author accepts or
  rejects. Plan Manager consumes whatever search-hub federates today and improves
  automatically as more Answer-projection providers register; it never builds those
  providers. If down/empty: no candidates — the references step falls back to
  manual locator entry or a `NO_CODE_REFS:` reason (never a fabricated reference).
- **code-facts** — resolves reference evidence at validation where its current
  surface can express the locator. If down: references remain recorded; validation
  falls back to filesystem resolution for CODE/DOC refs and marks unresolved gaps
  honestly.
- **git-control-tower** — captures the regression-anchor baseline snapshot at
  **execution start** (relocated from authoring: a plan is durable across the
  authoring→execution gap, so the "before" is only true immediately before edits
  begin) and diffs it for validation/DoD. Authoring records typed anchor **intent**
  only and never shells git-control-tower. The capture is delegated to the
  validation domain through execution's `InputFreshener` seam, runs once per start,
  and degrades honestly (recorded + surfaced, non-blocking, retried on resume); DoD
  verification reports anchor-unavailable when the diff cannot run.
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
- **test-genie / scenario-validation** — validation results consumed for plan
  health. plan-manager never re-implements project-level validation; it reads.
- **prompt-manager** — relevant-context skill/action discovery for authoring and
  setup guidance (the Guide projection, distinct from the search-hub Answer
  projection that feeds references). Authoring stores discovered setup as pending
  candidates; the author must accept useful candidates or reject noisy ones before
  finalization. If down: context candidates are marked degraded or left for the
  author to supply explicitly.
- **meta-optimization-manager** — velocity sink. If down: velocity is retained
  locally and emit is retried/skipped; no flow blocks.
- **agent-manager** — provides the run-id attribution contract
  (`VROOLI_AGENT_MANAGER_RUN_ID`) used to dedup handoff actions. **Owns the prose
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

The v1 DEFAULT downstream sinks are documented **pending stubs**, mirroring the
existing `VelocitySink`/MoM stub pattern: an entry is created and left
`sync_status = pending`, durable and retryable — there is no API-blocking
auto-forward to an absent downstream. The live bug-filer (scenario-qa) and
record-writer (`swarm-manager records create`) land behind the same seams as a
deferred follow-up (a drop-in wire). Downstream
failure is **never fatal**: the local entry persists with `sync_status`
`pending`/`sync_failed`, surfaces in list/status summaries, and is retried with
`plan-manager log sync <id>` once the downstream is reachable.

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
