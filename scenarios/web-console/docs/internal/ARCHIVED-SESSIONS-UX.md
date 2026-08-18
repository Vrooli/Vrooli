# Archived sessions — UX design reference

Durable record of the design investigation and the operator decisions behind the
web-console session archive. The plan
`web-console-session-archive-*` executes this design; this document holds the
rationale, the wireframes, and the measured facts the plan compresses.

Visual companion: [`ARCHIVED-SESSIONS-UX.html`](./ARCHIVED-SESSIONS-UX.html) —
open in a browser for the rendered wireframes in web-console's real palette.
The ASCII wireframes below carry the same information and are authoritative if
the two ever disagree.

---

## 1. The friction

The operator keeps sessions open that they do not need running, because closing
one destroys it. Three distinct needs drive this:

1. **Reference.** A past conversation contains wording worth reusing, or a
   message worth copying into a live session for context.
2. **Resume.** A conversation with "good bones" that may be worth continuing.
3. **Cleanliness.** Sessions kept only for reasons 1 and 2 clutter the sidebar
   list and its groups.

A fourth consequence follows: each retained session holds a live agent process
(claude/codex), which costs host memory. Memory is a real cost but it is not the
design driver — retrieval and cleanliness are. A memory-only fix would be a
different, smaller feature ("kill the agent, keep the tmux").

## 2. Measured current behavior

All measurements taken 2026-08-18 against the live store
`~/.vrooli/data/vrooli/web-console/web-console.db` (139 MB).

| Fact | Value |
| --- | --- |
| Live session rows | 58 (37 claude, 20 codex, 1 shell) |
| Dismissed session rows | 143 |
| Conversation events total | 71,446 |
| Events belonging to live sessions | 29,452 across 57 sessions |
| Events belonging to dismissed sessions | 41,994 across 127 sessions |
| Events with no session row | 0 |

### 2.1 Close is a hard cascading destroy

`Adapter.Delete` (`api/handlers/sessions/adapter.go:254`) calls
`Manager.Delete` (`api/session/session_manager.go:427`), which kills the PTY,
deletes the `sessions` row, and removes the session upload directory. The
adapter then deletes the conversation (`api/conversation_repository.go:480`,
cascading `conversation_events` via FK), the codex rollout checkpoints, and the
agent transcript checkpoints.

In the UI, `removePane` (`ui/src/hooks/useSessionManager.ts:291`) calls
`deleteSession` unconditionally. The X on a tab is therefore a permanent,
unconfirmed delete of the transcript.

### 2.2 A de-facto archive already exists, with no reader

`MarkDismissed` preserves the session row and does **not** delete the
conversation. 127 dismissed sessions currently hold 41,994 conversation events
that are already durable and completely unreachable from the UI. The zero
"no session row" count confirms the delete cascade is complete and that these
41,994 events came from the dismiss path, not from leakage.

**Consequence for execution:** the archive reader has real content to display on
day one, with no migration and no backfill. This is the primary Phase 2
validation opportunity — do not discard these rows.

### 2.3 Restore is already built, and gated to crash orphans

`Recover()` (`api/handlers/sessions/adapter.go:326`) already performs the entire
restore: it creates a new session with the old shell/cols/rows/policy/cwd,
copies CODEX_HOME, copies the conversation history onto the new session id,
re-applies provenance (origin/owner/label), reassigns the workspace pane so
name, color and group survive, and pastes an agent-specific resume command from
`BuildResumeCommand` (`api/internal/sessions/helpers.go:94`).

Its only barrier to serving the archive is a status gate: it refuses any row not
in `awaiting_recovery`, and only the crash-recovery sweep
(`api/session/session_lifecycle.go`) sets that status.

### 2.4 Message search exists and is scoped to one session

`searchConversation(sessionId, query)` (`ui/src/api/conversation.ts:149`) already
searches message **text** and drives the navigator panel in `MessagesPane`.
Server-side it is `SELECT … WHERE session_id = ? AND text LIKE ? ESCAPE '\'`
with a 160-character excerpt builder (`api/conversation_repository.go:243`).

Two limits matter:

- The `session_id` predicate is the only thing preventing cross-session search.
- There is no FTS5 table and no index on `conversation_events.text`. A
  `LIKE '%…%'` scan is acceptable over one session's few hundred rows and is a
  full table scan over the whole archive. At 71,446 rows today and growing, the
  scope widening and the FTS5 index are one change, not two.

### 2.5 There is no retention anywhere in web-console

No prune, no vacuum, no age or size policy exists in the scenario. **Today the
delete cascade is the retention policy.** Archiving removes it. Retention must
therefore ship with the archive, not after it.

### 2.6 Semantic search has no foundation in this scenario

web-console has one AI provider (Ollama, `api/internal/ai/provider.go`) used for
generate/summarize. There are no embeddings, no vector store, and no backfill
anywhere in the scenario. Semantic retrieval is a genuinely useful third rung
and a multi-week project; it must not block the first two rungs.

## 3. Decisions

### D1 — Close means archive; delete becomes explicit

The X on a tab archives. Delete moves to the context menu behind a confirm.
Rationale: this inverts the default from destructive-and-silent to safe. The
undo toast is the cheapest possible expression of that inversion and ships
first, before any archive UI exists.

*Rejected:* a confirm dialog on close. It adds friction to the common case
without making the outcome recoverable.

### D2 — `archived_at` column, not a new `status` value

The `sessions.status` column carries
`CHECK(status IN ('live','awaiting_recovery','dismissed'))`
(`api/internal/sessions/schema.sql:21`). SQLite cannot widen a CHECK with
`ALTER TABLE`; adding `'archived'` requires a full table rebuild of a table that
58 live sessions depend on.

Add `archived_at TEXT NOT NULL DEFAULT ''` instead:

- It is a pure additive column, which api-core's schema reconciliation adds to
  the existing DB automatically once declared in `schema.sql`.
- It is semantically better. Archived is orthogonal to whether the process
  survived: a crash orphan (`awaiting_recovery`) and a deliberately archived
  session are different facts about the same row and should not compete for one
  field.

*Tradeoff:* two fields now express "not open", so every listing query must
consider both. Accepted, because the table rebuild risk against live sessions is
worse. *Revisit if:* a third lifecycle concept appears and the pair becomes
ambiguous.

### D3 — Sidebar footer row, not a fourth origin tab

The origin tab strip partitions by **provenance** (ui / programmatic / remote).
Archive is a **lifecycle** axis. Adding it as a fourth tab means that inside the
archive the origin partition silently disappears, and the sort control
("unread", "activity") stops having meaning.

The archive gets a pinned footer row in the sidebar instead. It is always
visible, including when the origin strip is hidden — which is the common case,
because the strip only renders when a non-UI session exists
(`ui/src/components/SessionSidebar.tsx:135`).

*Operator note:* the fourth-tab option was proposed and considered; the footer
was chosen for the reasons above. This is a mild preference. Do not spend
execution time relitigating it, and do not silently switch to the tab form.

### D4 — Two-tier retrieval: shallow sidebar, deep drawer

The sidebar archive view is the shallow tier: the most recent ~20 archived
sessions with a **name** filter. The deep tier is a full drawer with
**message-text** search across every archived session. One escalation control
connects them.

Rationale: most retrieval is "the thing I closed twenty minutes ago" and should
not cost a full-screen context switch. Keeping name-filtering and message-search
on visibly different surfaces removes any ambiguity about what the box you are
typing in actually searches.

### D5 — Results are messages, not sessions

The deep surface returns **messages** with their session as context. The
operator does not remember which session it was; they remember a phrase.
Returning sessions that must then be opened and searched again reproduces the
problem the feature exists to solve.

### D6 — Reuse `MessagesPane` read-only; do not build a viewer

The detail pane is `MessagesPane` in a read-only mode. This inherits markdown,
mermaid, the file viewer, and the existing copy/export affordances, and keeps
the archive consistent with the live messages view permanently.

### D7 — Resumability is a prediction, and it decays

`Recoverability()` (`api/internal/sessions/helpers.go:65`) already refuses
claude, opencode and grok rows with no `agent_session_id`, and a shell session
with `agent_type=none` has nothing to resume. Beyond that, resume replays the
agent's own on-disk history (`~/.claude/projects`, per-session CODEX_HOME); if
that is gone, resume produces a broken session.

The archive therefore has three visible states, shown on the row **before** the
click, following the existing pattern in `RecoverableSessionsBanner`
(disabled button plus a reason string):

| State | Meaning |
| --- | --- |
| Reopenable | Agent id recorded and agent history present on disk |
| Read-only | Transcript intact; agent history pruned or never recorded |
| Nothing to restore | Shell session, no messages, no agent |

An archive naturally decays from reopenable to readable as agent homes are
pruned. Those directories are far larger than the SQLite rows and are what
retention actually needs to reclaim. Showing the decay is more honest than
implying everything reopens forever.

### D8 — Reopen must not multiply the transcript

`Recover()` **copies** conversation rows to the new session id
(`CopySession`, `api/conversation_repository.go:504`). Archive → reopen →
archive → reopen therefore leaves three copies of one conversation in the
archive. The archive must present one entry per conversation lineage. See open
question Q1 for the two acceptable resolutions.

### D9 — The archive subsumes the recovery banner

A crash orphan is an involuntarily archived session. Folding
`RecoverableSessionsBanner` into the archive surface gives one mental model and
one code path, and removes a surface.

## 4. Wireframes

### 4.1 Sidebar — chosen form (D3)

```
┌──────────────────────────────┐
│ Sessions                  +  │
├──────────────────────────────┤
│ [ UI 4 ][ Prog 2 ][ Remote 1]│  origin tabs (unchanged)
├──────────────────────────────┤
│ ⇅ Sort · Activity          ▾ │  (unchanged)
├──────────────────────────────┤
│ ● Vrooli work                │
│   claude · agi        2m   × │
│   codex · docs       now   × │  ← × now archives
│ ● Scratch                    │
│   terminal            1h   × │
│                              │
├──────────────────────────────┤
│ ▤ Archive               137  │  ← NEW pinned footer row
└──────────────────────────────┘
```

Rejected form, for the record — the fourth origin tab:

```
│ [ UI 4 ][ Prog 2 ][ Rem 1 ][ Archive 137 ] │   ← NOT this (see D3)
```

### 4.2 Sidebar — archive mode (shallow tier)

Clicking the footer row swaps the sidebar body. It does not open a modal.

```
┌──────────────────────────────┐
│ ← Archive                137 │
├──────────────────────────────┤
│ ⌕ Filter archived…           │  name filter only, client-side
├──────────────────────────────┤
│ Recent                       │
│ claude · trusted-receipt-…   │
│   2h ago · 148 msgs · reopen │
│ codex · storage-manager…     │
│   yesterday · 39 msgs·reopen │
│ terminal · scratch           │
│   3d ago · no msgs · read-on │
│                              │
│ ┌──────────────────────────┐ │
│ │ Search all 137 sessions →│ │  escalation to deep tier
│ └──────────────────────────┘ │
└──────────────────────────────┘
```

### 4.3 Deep archive drawer — search first (D4, D5)

`DrawerShell size="full"` already exists and is used by
`RecoverableSessionsBanner`.

```
┌────────────────────────────────────────────────────────────────────┐
│ Archive                                              Esc to close  │
├────────────────────────────────────────────────────────────────────┤
│ ⌕ receipt signing                      142 messages · 9 sessions   │
├────────────────────────────────────────────────────────────────────┤
│ [Text] [Semantic]  │  [All agents ▾] [Any time ▾] [My messages]    │
├──────────────────────┬─────────────────────────────────────────────┤
│ ● claude ·           │ claude · trusted-receipt-signing            │
│   trusted-receipt-…  │ archived 3 days ago · 148 messages          │
│   …the ▮receipt      │ ~/Vrooli · branch agi                       │
│   signing▮ key must  │ ● Reopenable   [Reopen] [Export] [Delete]   │
│   never leave…       ├─────────────────────────────────────────────┤
│   3 days ago · you   │ you                                  14:02  │
│                      │ The ▮receipt signing▮ key must never leave  │
│ ● codex · docs sweep │ the enclave, which means the verifier…      │
│   …document ▮receipt │   [Copy] [→ Send to composer] [Jump]        │
│   signing▮ under…    ├─────────────────────────────────────────────┤
│   1 week ago · codex │ claude                               14:03  │
│                      │ That constrains you to a detached-…         │
│ ● claude · agi       │                                             │
│   …trusted experiment│                                             │
│   ▮receipt signing▮  │                                             │
│   2 weeks ago · you  │                                             │
└──────────────────────┴─────────────────────────────────────────────┘
   message hits across        full transcript, scrolled to the hit,
   every archived session     read-only (MessagesPane reuse)
```

`→ Send to composer` inserts the selected message into the active live
session's composer. It is the literal job the operator described and is expected
to be used far more often than Reopen.

### 4.4 The three archive states (D7)

```
┌────────────────────────────────────────────────────────┐
│ claude · trusted-receipt-signing                       │
│ archived 3 days ago · agent transcript on disk         │
│ ● Reopenable          [Reopen]  [Export]               │
└────────────────────────────────────────────────────────┘
┌────────────────────────────────────────────────────────┐
│ claude · early experiment                              │
│ archived 5 months ago · agent transcript pruned        │
│ ● Read-only — agent history no longer on disk          │
│                       [Reopen ✗] [Export]              │
└────────────────────────────────────────────────────────┘
┌────────────────────────────────────────────────────────┐
│ terminal · scratch                                     │
│ archived 3 weeks ago · no agent, no messages           │
│ ● Nothing to restore  [Reopen ✗] [Delete]              │
└────────────────────────────────────────────────────────┘
```

### 4.5 Close → undo (D1)

```
┌──────────────────────────────────────────┐
│ ▤  Archived claude · agi          Undo   │
└──────────────────────────────────────────┘
```

Toast on close, approximately 8 seconds.

## 5. The retrieval ladder

| Rung | Matches | Needs | Ship |
| --- | --- | --- | --- |
| Name filter | Session and group names in the shallow list | Client-side string match over rows already loaded | Yes |
| Text search | Literal substrings in every message of every archived session | Drop the `session_id` predicate; add an FTS5 index | Yes |
| Semantic | Intent with no shared words | Embedding model, vector store, backfill, re-embed on write | Design the control, do not build |

The `Text | Semantic` toggle is built in the UI now so semantic retrieval later
slots into an existing surface instead of forcing a redesign. The Semantic
control renders disabled with an explanatory title until the capability exists.

Honest expectation: literal search over the operator's own messages covers most
of the described need, because the remembered detail is usually their own
phrasing.

## 6. Open questions

**Q1 — Does reopen copy the transcript or re-point it?** (blocks D8)
Two acceptable resolutions: re-point the `conversation_events` rows to the new
session id instead of copying; or keep the copy and add a lineage column so the
archive collapses a reopen chain into one entry. Re-pointing is simpler and
loses the ability to see the conversation as it stood at archive time.
Copy-plus-lineage preserves that and costs storage per reopen.

**Q2 — What is the retention policy?** Proposed: retain message-bearing
transcripts indefinitely; expire message-less shell sessions after 7 days;
prune agent home directories on an age policy, which moves a row from
Reopenable to Read-only rather than deleting it; surface total archive size in
settings.

**Q3 — Archive uniformly, or only message-bearing sessions?** Proposed: archive
uniformly so the close gesture never surprises, and default the archive view to
a "has messages" filter.

## 7. Prior art in this repo

| Plan | Status | Relevance |
| --- | --- | --- |
| `web-console-crash-recovery-hardening-pane-state-migration` | complete | Built `Recover()`, the pane reassignment, and the recoverable drawer this design extends |
| `web-console-session-provenance-headless-launch-live-sidebar` | complete | Built the origin tab strip and `SessionOrigin`; D3 preserves its partition semantics |
| `export-selected-web-console-messages-for-coding-agent` | complete | Built message export/handoff; `→ Send to composer` should reuse it, not duplicate it |
| `web-console-overlay-consolidation-drawershell-as-the` | complete | Established `DrawerShell` as the standard overlay; the deep archive uses it |
