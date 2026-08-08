# Problems — Vrooli Memory

Persistent register of known issues, tech debt, and deferred work
specific to **this** scenario. Future agents read this file to avoid
re-discovering the same constraint.

This register carries only scenario-specific, measured issues and deferred
work; it is intentionally non-empty while those issues remain unresolved.

## What belongs here

- **Known bugs** that are real but not yet worth fixing
- **Tech debt** — workarounds that need a real fix later
- **Deferred work** — features descoped from a phase, with the reason
- **Architecture drift** — code/docs/tests that no longer line up with
  the intended capability map or boundary model
- **Constraints discovered the hard way** that aren't visible from
  the code (e.g., "this resource needs warm-up before the first call;
  see commit X")

## What does NOT belong here

- **Generic template issues** — those go in
  [`../guides/troubleshooting.md`](../guides/troubleshooting.md)
- **Open feature requests** — track those in PRD operational targets
- **Code comments** — if the constraint is local to one file, a
  comment there is more discoverable
- **Test failures** — fix them, don't document them

## Entry template

Use this shape so entries are scannable. Append newest at the bottom.

```markdown
### YYYY-MM-DD — short title

**Symptom:** What goes wrong, observable from outside the system.

**Root cause:** What actually causes it (or "unknown" if not yet diagnosed).

**Workaround:** What to do today to keep moving.

**Real fix:** What needs to happen for this entry to be deleted.

**Owner:** Who should drive the fix (or "unassigned").

**Refs:** Code paths, related issues, prior commits.
```

## Entries

### Open gaps carried from design (2026-07-27)

| Gap | Impact | Tracked As |
|---|---|---|
| Classifier calibration was below the unattended-use bar in the legacy 17/50 review (34%). | That historical sample is superseded by the repeatable live fixture: 432/432 held-out work records correct (100.00%) and 558/558 fall-through triples unanimous (100.00%) through the normal gateway role, with no provider changes. | Keep the repeatable fixture current; do not treat the legacy human sample as the current calibration gate. See `CALIBRATION-REVIEW.md`. |
| Facet embedding-space count is a guess (3: topic, rule/implication, entities). | Too few loses clustering recall; too many wastes inference. | `VMEM-P1-005` — needs real clustering output to settle. |
| `run_id` is nullable for heartbeat-spawned agents. | Memory→run backlink is absent for those writes. Documented upstream in `docs/agent-system/RUNTIME_ATTRIBUTION.md` with the token-claim overlay listed as future strengthening. | `VMEM-P1-002` — write path must tolerate absent correlation. |
| Adoption depends on the harness prompt block being installed and kept current. | A runtime with a stale or missing block silently keeps its private store, so the unification claim stops being true for it without any error. | `VMEM-P1-007`. |
| Deliberate-write path assumes agents notice what is worth remembering. | Untested. The 1-in-200 records measurement is evidence about *flags*, not about *noticing*. | `VMEM-P2-001` is the fallback if this proves false. |
| Summarization drift (fact mutation vs. intended fact dropping) is uninstrumented. | Quality-only: `forest` is rebuildable, so drift never costs data. | `VMEM-P2-003`, deferred by decision D-012. |
| Live local summarization previously returned `unavailable: unexpected EOF` through ai-gateway after 28–43 seconds. | The provider timeout was separated from classification and the live compaction pass now completed 31 summaries atomically; the old failure is retained only as historical context in the progress log. | Keep the five-minute summary timeout and atomic forest write; do not bypass ai-gateway or write summaries directly. |
| Native behavior beyond the scenario's projection ceiling is not uniform or documented for every runtime. | **Resolved for the P0 guarantee:** all six configured targets use a measured 32,768-byte scenario guard, emit pins first, and refuse pinned overflow before writing; native oversized files are not needed for the safety proof. | `VMEM-P0-010`; see the projection ceiling table in `INTEGRATIONS.md` and `TestProjectionRefusesPinnedOverflowForEveryConfiguredRuntime`. |
| **The two single-blob harnesses (Codex, opencode) expose no usable pre-write hook in the installed runtime.** `pretooluse-bash-deny.sh` proves the mechanism exists in `claude-code` and `grok`; the per-runtime matrix records the negative result for Codex/opencode and the other no-hook harnesses. | Store diff is the universal floor, but D-015 records that single-blob stores cannot be diffed reliably into discrete memories. Capture therefore remains precise for hook-capable runtimes and best-effort via governed store-diff recovery elsewhere; the limitation is explicit rather than guessed. | `VMEM-P1-008` — retain the verified capability matrix and add a native hook when a resource exposes one. |

### 2026-08-05 — Two of the three facet embedding spaces have no reader

**Symptom:** Every entry stores three facet-text embeddings (`topic`, `rule`, `entities`) — 8,181 vectors across 2,728 entries — but retrieval and clustering behave as if there were one.

**Root cause:** All three consumers select a single vector with `ORDER BY ft.id LIMIT 1`: `internal/recall/sqlite_source.go`, `internal/forest/sqlite_source.go`, `internal/forest/sqlite.go`. Nothing scores across spaces. This is distinct from the design-table gap above, which asks *how many* spaces are right; this entry records that the extra spaces are written and never read.

**Workaround:** None needed for correctness — retrieval works on the first space. The cost is two thirds of embedding inference and storage with no consumer.

**Real fix:** A multi-space scorer (best-of or weighted blend per space), or stop deriving the unused texts. `VMEM-P1-005` was reopened on 2026-08-05 for this reason.

**Owner:** unassigned

**Refs:** `internal/recall/sqlite_source.go`, `internal/forest/sqlite_source.go`, `VMEM-P1-005`.

### 2026-08-07 — The corpus ran from `/tmp` against a different backup target (RESOLVED)

**Symptom:** The API ran with `SQLITE_PATH=/tmp/vmem-refacet-copy.db` while backup target `33df5137` pointed at `scenarios/vrooli-memory/data/vrooli-memory.db`. Two days of writes existed in one volatile file that nothing backed up.

**Root cause:** A phase-4 re-facet working copy was committed into `.vrooli/service.json` as a permanent environment override and never unwound. Removing it revealed a third path — the storage resolver default at `~/.vrooli/data/vrooli/vrooli-memory/` — which the service had never been using.

**What made it dangerous:** the two files were forks, not stale-and-fresh. 351 entries existed only in the runtime copy and 3 only in the repository copy, with all 2,725 shared bodies byte-identical. Copying either file over the other would have deleted permanent journal rows.

**Real fix (done):** merged by union to 3,081 entries with the high-water mark advanced, installed at the resolver path, override removed, backup target re-pointed, backup and restore drill verified equal row counts and equal per-row body hashes, redundant copies removed, and the plan given a 12h schedule with a weekly recovery drill. See D-039.

### 2026-08-07 — The compaction forest was empty and nothing rebuilt it (RESOLVED)

**Symptom:** `summaries` and `tree_edges` were both zero while the eligible frontier held 2,628 nodes against a target of 16. Wake still returned 39 items, so nothing looked wrong.

**Root cause:** two correct changes with a gap between them. D-031 made `Rebuild` a provider-independent cache recovery that drops summaries and does not regenerate them, and D-032 built the maintenance loop as import plus projection only. After the phase-4 re-facet rebuild, nothing ever called compaction again.

**Real fix (done):** `forest.Service.RunBounded` plus a scheduled compaction step between import and projection, and a `canopy` health check that reports the eligible frontier against its target. See D-036 and D-037.

### 2026-08-07 — Four runtimes reported a false import failure every tick (RESOLVED)

**Symptom:** codex, gemini, grok and opencode each failed with `non-empty harness store yielded zero importable items` on every maintenance run.

**Root cause:** their entire store is this service's projection — all four files byte-identical at 17,037 bytes. The one-directional projection fix correctly strips the managed block, leaving nothing, and the importer reported that empty result as an error.

**Real fix (done):** `extractPath` drops sources that are empty after marker removal and reports that it did so; that case returns a completed import with zero items. See D-038.

### 2026-08-05 — The journal database carries about 60% dead pages

**Symptom:** `data/vrooli-memory.db` is 152 MB on disk while 23,233 of its 38,891 pages sit on the freelist.

**Root cause:** The embedding migration rewrote 8,181 vectors from JSON text to blobs with no subsequent `VACUUM`, so freed pages were never returned to the filesystem.

**Workaround:** None needed — free pages are reused before the file grows again.

**Real fix:** `VACUUM` once, then add it to a maintenance path so future format migrations reclaim automatically.

**Owner:** unassigned

**Refs:** `internal/vector`, `data/vrooli-memory.db`.

## Architecture Drift

Use this section for deferred findings from `screaming-architecture-audit`.
Do not create a standalone architecture-audit report unless the work is
a migration handoff with a planned retirement path back into
`ARCHITECTURE.md`, `SEAMS.md`, or this file.

| Area | Drift | Maturity Impact | Real Fix |
|---|---|---|---|
| No confirmed architecture drift | The shipped domain boundaries match the implemented journal, facets, forest, recall, federation, and harness surfaces. | None. | Re-audit whenever a cross-domain persistence dependency is introduced. |

## Cross-references

- [`PROGRESS.md`](PROGRESS.md) — lifecycle log (forward-looking)
- [`SEAMS.md`](SEAMS.md) — boundary registry (load-bearing for tests)
- [`TESTING.md`](TESTING.md) — test patterns
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — generic-template issues

## Work ladder

- Rung: W0 passed
- Evidence: The deterministic named-mention search `swarm-manager goals list --json | jq -r --arg s 'vrooli-memory' '.goals[].goal | select((.name + " " + .title + " " + .description) | test($s)) | .name'` returns the active governing goal `complete-and-validate-vrooli-memory-to-a-p0-p1-bar-with-a`. Its description requires the plan, generic policy/engine boundary, every P0/P1 requirement with named passing evidence, federation/harness validation, and a healthy ledger-ready scenario; the plan's P0/P1 operational targets describe the same outcome.
- Blocker: None at W0. Re-measurement continues at W1/W2; no maturity claim is made until the requirement evidence and plan acceptance are complete.
- Measured: 2026-08-04
