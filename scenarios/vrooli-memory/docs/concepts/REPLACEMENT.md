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

**Today.** Every agent runtime keeps its own memory, in its own shape. Claude
Code writes 392 individual memory files plus a hand-curated `MEMORY.md` index;
Gemini CLI appends to a section of `~/.gemini/GEMINI.md`; Codex has a native
`~/.codex/memories/` directory and OpenCode has no documented native memory
store; agent-manager runs accumulate their own.
None of them can read the others. Per-harness detail is in
[`INTEGRATIONS.md`](INTEGRATIONS.md) → Harness Capability Matrix.

**After.** One journal, one search space, split by direction:

- **Reads are uniform where a native memory store exists.** Each supported
  memory store gets a *generated projection* of `memory wake`, written only to
  that store. Curated instruction files are not projection surfaces.
- **Writes are captured or explicitly routed.** The agent keeps using its own
  memory tool where the runtime exposes one; the harness domain captures that
  write via a pre-write hook and recovers it through store diff. Runtimes
  without a native memory store use the explicit `vrooli-memory journal note`
  command. The scenario never modifies an instruction file to teach it.

**Why the projection rather than a sync.** Bidirectional sync between an
editable file and a database is a conflict problem with no clean answer. A
one-directional projection also gives something better than interception: a
harness with **no hook API at all** still receives memory, because reading its
own memory file is native behaviour.

**Why native-first plus an explicit fallback.** Native capture remains the
best experience because the agent keeps using the memory feature it already
understands. Runtimes without a native store cannot be made seamless by
silently editing their instruction files, so their documented fallback is the
explicit `vrooli-memory journal note` command. This is an honest degraded path
rather than pretending store diff is real-time. Recorded as D-041 and D-042.

**Hooks are load-bearing, not hardening.** Two findings moved them:
`pretooluse-bash-deny.sh` already ships in the `claude-code` and `grok`
resources, so the mechanism is proven rather than speculative; and single-blob
harnesses cannot be diffed reliably into discrete memories, so for those a hook
is the only precise channel. Hook install rides the existing per-resource
permissions machinery under `resources/<agent>/cli/internal/permissions/` —
**this scenario never edits agent binaries**, it extends the resource that
already configures them.

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
`records create`. Tracked as `VMEM-P1-001`.

### 4. The duplicate federated `swarm-manager.records` provider

The absorbed work records remain in swarm-manager's relational storage for
workflows that need them, but that storage is no longer a second answer in
federated memory search. The `swarm-manager.records` Search Hub provider was
removed from `scenarios/swarm-manager/.vrooli/search.json` and deregistered from
the live provider registry on 2026-08-06. Federation is now owned by
source-ledger: `source-ledger.agent-memory` and the other per-scope providers
are the single federated answer for ledger records. This is provider retirement,
not data deletion.

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
   nothing retired. Sequenced before import because imported entries need a
   facet assigned on the way in.
3. **Import live; harness stores backfilled.** Existing per-harness memory is
   imported idempotently (`VMEM-P0-011`). This step is **read-only against
   the harnesses** and reversible — nothing in a runtime changes yet, so a bad
   import is corrected by re-facet or delete rather than by rollback.
   **This must precede any projection.** A projection overwrites the harness
   memory file, and `MEMORY.md` is itself an import source (§ 2) — projecting
   first would destroy the corpus before it was read. It also precedes
   compaction, because a compaction loop over an empty journal cannot be
   validated (D-022).
4. **Compaction live.** The frontier holds under pressure at realistic corpus
   size — realistic because step 3 supplied it.
5. **Federation + projection live.** Memory is reachable from `search-hub query`
   and each harness reads its projection. `MEMORY.md` becomes generated —
   the first actual retirement, and safe only because its content was imported
   at step 3.
6. **Capture live per harness.** Hook where the runtime exposes one, store diff
   elsewhere (`VMEM-P1-008`). Sequenced *after* import deliberately: import
   proves the adapters and the content-addressed key against real data before
   anything starts writing into a runtime's hook surface.
7. **Work-record kind live; records imported.** Docs and skills stop referencing
   `records create`, which is retired from agent-facing documentation but is not
   deleted.
8. **Reassess.** Only after the write path has real usage does it make sense to
   ask whether swarm-manager's records storage should be removed. That is a
   separate decision with its own evidence, not a consequence of this one.

## Known Risks Of The Replacement

Recorded honestly rather than argued away.

- **Capture coverage varies by storage shape, not by intent.** Under D-015
  adoption no longer depends on an agent obeying an instruction — but it now
  depends on the capture channel working per runtime. File-per-fact and
  append-under-section harnesses diff cleanly. Codex's native memory directory
  has no verified pre-write hook, so store diff is the available floor. OpenCode
  has no documented native memory store and therefore remains explicit-write;
  neither runtime's curated `AGENTS.md` is used for memory.
- **Native runtime limits are not uniform.** Codex documents a 32 KiB injection
  cap, while the other installed runtimes expose no stable numeric limit. The
  scenario therefore applies a measured 32,768-byte guard to every projection,
  emits pins first, and refuses a pinned overflow before writing. Native
  over-cap files were not written as part of validation.
- **opencode has no native memory feature.** It reads curated `AGENTS.md`;
  this scenario never projects into or imports that file. OpenCode therefore
  uses the explicit journal command until a native memory store or plugin
  integration is verified.
- **Cursor is not yet in scope.** It appears in market-facing docs but has no
  `resources/cursor`, and stores rules in a SQLite BLOB rather than a file. It
  is `VMEM-P2-002`, blocked on a prerequisite outside this scenario.
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
- **The pinned set accretes and nothing automatic removes from it.** Pins are
  exempt from compaction by design, so they are the one structure with no
  relief valve. Agents cannot pin, so this is slow accretion rather than a
  flood — but the binding constraint is attention, not tokens: a wake view of
  fifteen standing rules is read and one of three hundred is skimmed. The
  redundancy is already present in the file being replaced, which carries six
  git entries of which three are one rule stated three times. Curation
  (`VMEM-P1-010`, D-018) is the response; its budget and review interval
  are unvalidated.

## Cross-References

- [`../internal/DECISIONS.md`](../internal/DECISIONS.md) — D-001, D-004, D-010, D-011 and the evidence behind each
- [`DOMAINS.md`](DOMAINS.md) — domain boundaries and explicitly-rejected domains
- [`INTEGRATIONS.md`](INTEGRATIONS.md) — dependency contracts and failure behaviour
- [`ARCHITECTURE.md`](ARCHITECTURE.md) — the layering invariants this replacement preserves
