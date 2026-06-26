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
| `code-facts` | Scenario | Resolve `[CODE:]`/`[REQ:]` references where its current surface can provide evidence | Soft |
| `git-control-tower` | Scenario | Regression anchor (baseline snapshot/diff) | Soft |
| `cli-health` | Scenario | Validate authored `cli:` command references in plans without executing them | Soft |
| git / freshness engine | Platform | Per-reference drift is git-sourced today; freshness engine remains scenario-artifact scoped | Soft |
| `test-genie` / `scenario-validation` | Scenario | Validation results consumed (not owned) | Soft |
| `prompt-manager` | Scenario | `plan-skill-discovery` for required-reading autofill | Soft |
| `meta-optimization-manager` | Scenario | Velocity signal sink (trials) | Soft |
| `agent-manager` | Scenario | Run-id attribution contract; owns prose-handoff capture | Soft |

Future consumers to be **inverted** (they will depend on plan-manager, not the
reverse): `swarm-manager` `phased-plan-drain`, project hygiene plan checks, the
`vrooli plans` CLI. Sequenced after standalone proof (OT-P2-002).

## Vrooli Resources

- **SQLite home store** — the only required resource. Rooted at `~/.vrooli` (see
  [`DATA.md`](DATA.md)) so plans persist independent of the server process. No
  heavy resources (no Ollama/Qdrant) — plan-manager aggregates structured data and
  composes other scenarios' typed CLI/RPC; it does not embed or search.

## Scenario Dependencies

All soft; each is reached through a seam and degrades to a marked gap rather than
failing the flow:

- **code-facts** — resolves reference evidence where its current surface can
  express the locator. If down: references remain recorded; validation falls back
  to filesystem resolution for CODE/DOC refs and marks unresolved gaps honestly.
- **git-control-tower** — captures/diffs the regression anchor. If down: anchor
  autofill is skipped and flagged; DoD verification reports anchor-unavailable.
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
- **prompt-manager** — `plan-skill-discovery` for required-reading autofill. If
  down: required-reading is left for the author to fill.
- **meta-optimization-manager** — velocity sink. If down: velocity is retained
  locally and emit is retried/skipped; no flow blocks.
- **agent-manager** — provides the run-id attribution contract
  (`VROOLI_AGENT_MANAGER_RUN_ID`) used to dedup handoff actions. **Owns the prose
  final-handoff capture** (it has the transcript); plan-manager links to it by
  reference but never reads transcripts itself.

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
- **Attribution missing** (non-agent run) → handoff dedup falls back to
  content-based checks; no double-file guarantee weakens to best-effort.

## Cross-References

- [`DATA.md`](DATA.md) — the home store and data ownership
- [`DOMAINS.md`](DOMAINS.md) — which domain consumes each dependency
- [`ARCHITECTURE.md`](ARCHITECTURE.md) — composition boundaries
- [`../reference/configuration.md`](../reference/configuration.md) — dependency configuration
- [`../../PRD.md`](../../PRD.md) — dependencies & launch plan
