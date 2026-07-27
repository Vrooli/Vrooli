# Replacement & Migration — Vrooli Memory

This document is the canonical statement of **what this scenario replaces, what
it deliberately does not replace, and how the transition happens**. It exists
because this scenario absorbs a capability that currently lives elsewhere, and
absorption without an explicit contract produces two half-used systems.

Durable rationale for each decision below lives in
[`../internal/DECISIONS.md`](../internal/DECISIONS.md); this document is the
operational view.

## What This Replaces

### 1. Per-harness private memory stores

**Today.** Every agent runtime keeps its own memory. Claude Code writes a memory
directory plus a hand-curated `MEMORY.md` index; Codex, Cursor, and agent-manager
runs each accumulate their own. None of them can read the others.

**After.** One journal, one search space. Each harness reads a *generated
projection* of `memory wake` written to the file it already loads at session
start, and writes through `memory note`.

**Why the projection rather than a sync.** Bidirectional sync between an
editable file and a database is a conflict problem with no clean answer. A
one-directional projection also gives something better than interception: a
harness with **no hook API at all** still receives memory, because reading its
own memory file is native behaviour. Hooks (`VROOLIME-P2-002`) are hardening on
top, never the mechanism.

### 2. The hand-curated `MEMORY.md` index

**Today.** Roughly 90% of that file is *pointers* — "READ FIRST for X", "grep the
prefix", archive files, one index line per memory. It is a hand-built retrieval
index that exists only because nothing searches memory semantically.

**After.** It is generated. The pointer layer disappears entirely, because
`memory recall` answers the question the pointers were standing in for. This is
the clearest measure of the scenario's value: the maintenance is currently paid
by a human, every session.

**Migration.** Existing `MEMORY.md` content imports as journal entries with
facets assigned by classification, then reviewed. The categories map cleanly:

| Existing `MEMORY.md` content | Facet |
|---|---|
| "NEVER git stash/restore", "Proto+Connect always" | `standing-rule` (pinned) |
| "HOST AUDIO: no hardware sink", "Linux host, never macOS" | `environment-fact` |
| "never DELETE+VACUUM giant SQLite tables" | `gotcha` |
| "plan FINALIZED, not started", "shipped uncommitted" | `thread` |
| Shipped-work summaries | `episode` / `work-record` |

Import is **not** automatic promotion to pinned. A misclassified standing rule
is this scenario's highest-consequence error (see `DOMAINS.md` → `facets`), so
pinning is operator-confirmed.

### 3. `swarm-manager records` as an agent-facing write verb

**Today.** `swarm-manager records create` with `--kind`, `--scenario`,
`--trigger`, `--approach`, `--evidence`, `--backlog-ref`, `--commit`, `--files`.

**After.** `memory note --kind work-record` carrying the same required narrative
field set. Backlog association derives from run correlation instead of a flag.

**The evidence that decided this.** Of the 200 most recent records, **1 carries
`backlog_ref` and 1 carries `commit`.** The linkage fields exist and are
essentially never populated — the predictable fate of metadata an agent must
remember to pass. Attribution derived from the request cannot be forgotten.

**Migration.** One-time read-only import of existing records as `work-record`
memories, preserving narrative and outcome. Docs and skills stop referencing
`records create`. Tracked as `VROOLIME-P1-001`.

## What This Does **Not** Replace

Being explicit here matters more than the absorption list, because an
over-broad reading of "memory replaces records" would break working systems.

| System | Status | Why it stays |
|---|---|---|
| `vrooli-events` receipts | **Untouched.** | It is the one truth about what happened in a run — every API call in every scenario, published automatically. Memory stores correlation ids and references it; copying any of it would create a second, drifting truth. |
| swarm-manager backlog, goals, milestones, proposals | **Untouched.** | Project management is a different concern from memory. Only the *records* write verb is absorbed, not the execution model around it. |
| swarm-manager records **storage and relational queries** | **Retained where relational.** | Absorption is of the agent-facing *write surface*. If swarm-manager's execution-review loop needs relational queries over records, that stays in swarm-manager; memory is not a good home for foreign keys into another scenario's domain objects. |
| `search-hub` | **Extended, not replaced.** | Memory registers as one more provider. The router still holds no corpus content and no vectors. Adding memory is a registry row, not a router change. |
| `knowledge-observatory`, `cli-health`, and other fleet indexers | **Untouched.** | Those index *project artifacts* (docs, commands, surfaces). Memory indexes *what agents learned*. Different corpora, both federated. |
| prompt-manager skills and actions | **Untouched.** | Skills are instructions; memory is experience. A skill may instruct an agent to write memory; it is not stored as memory. |

## Transition Sequence

Absorption happens only after the replacement is demonstrably better, never on
announcement. The ordering below is deliberate — each step is reversible until
the one after it lands.

1. **Journal + recall live.** Memory writes and retrieves. Nothing is retired.
   Both systems run; `records create` still works.
2. **Facets + pinning live.** Standing rules reach `wake` unconditionally. Still
   nothing retired.
3. **Compaction live.** The frontier holds under pressure at realistic corpus
   size.
4. **Federation + projection live.** Memory is reachable from `search-hub query`
   and each harness reads its projection. `MEMORY.md` becomes generated —
   the first actual retirement.
5. **Work-record kind live; records imported.** Docs and skills switch to
   `memory note --kind work-record`. `records create` is retired from
   agent-facing documentation but is not deleted.
6. **Reassess.** Only after the write path has real usage does it make sense to
   ask whether swarm-manager's records storage should be removed. That is a
   separate decision with its own evidence, not a consequence of this one.

## Known Risks Of The Replacement

Recorded honestly rather than argued away.

- **Adoption depends on the prompt block.** The mechanism for redirecting a
  harness's built-in memory behaviour is instructional. If the block is not
  installed or drifts stale in a runtime, that runtime silently keeps its
  private store, and the unification claim quietly stops being true for it.
- **Memory quality is bounded by what agents notice.** The deliberate write path
  (D-004) covers every harness, but it inherits the assumption that agents
  recognise what is worth remembering. The 1-in-200 records measurement is
  evidence about *flags*, not about *noticing*; the two are different, and the
  second is untested.
- **The compaction scoring shape is unvalidated.** `cohesion × slots freed` is
  plausible, not derived. Expect it to change on contact with real clustering
  output — it is the first thing to re-examine if summaries read badly.
- **A misclassified standing rule can vanish from context.** Pinning is the
  mitigation and operator re-facet is the correction, but a rule wrongly
  classified as an episode is compaction-eligible. This is why facet correction
  is a first-class UI action rather than an admin tool.

## Cross-References

- [`../internal/DECISIONS.md`](../internal/DECISIONS.md) — D-001, D-004, D-010, D-011 and the evidence behind each
- [`DOMAINS.md`](DOMAINS.md) — domain boundaries and explicitly-rejected domains
- [`INTEGRATIONS.md`](INTEGRATIONS.md) — dependency contracts and failure behaviour
- [`ARCHITECTURE.md`](ARCHITECTURE.md) — the layering invariants this replacement preserves
