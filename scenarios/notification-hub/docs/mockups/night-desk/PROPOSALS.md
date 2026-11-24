# Proposals — what the desk needs, and how it lands

A field-by-field plan for the four “to add” rows in the case study, plus the
contract work that makes the mockups honest. Written 24 November 2026 after
reading the Hub's packet checker, the steward's page gate, the proto, and the
shared embedding stack.

The guiding rule is the operator's bandwidth. A decision is a packet that costs
attention whether or not it is legible. A malformed decision spends the same
bandwidth as a good one and returns nothing, so the producer — not the operator —
should pay to frame it. Everything below moves cost upstream.

---

## 1. What already exists

Do not rebuild these. The mockups use them as-is.

| Capability | Where | State |
|---|---|---|
| Decisions-only count and queue | `api/internal/hub/ask_list.go` (`awaitingCountSQL`, `state='awaiting'`) | **live** |
| Multi-question asks with per-question recommendation | `AskQuestion` (`single_choice`, `multi_choice`, `variant`, `section_review`, `text`) | **live** |
| Family grouping in the list | `ui/src/features/asks/askListView.ts` groups by `decisionFamily` + `decisionCause` | **live in UI** |
| Family fields on the contract | `AskSpec.DecisionFamily/Cause/Kind`, proto 63–65, `asks` columns | **in tree, not live** |
| Lexical relatedness | `similarSubject` (content-word Jaccard ≥ 0.5), `checkRelatedOpen`, `ErrRelatedDecision` | **live** |
| Recurrence | `SubjectRecurrence` (count, dates, `previousAnswers`), threshold 2 | **live** |
| Packet linter | `api/internal/hub/packet_*.go` — ~25 codes, `advice`/`blocking` | **live** |
| Page gate | `steward/bin/decision_pages.py:page_lint` — viewport, shots, network, prefill, restatement ratio, jargon, `operator_action`, `root_cause`, `batch_summary` | **live** |
| Answer receipts and readback | `answers`, `resolved_at`, `answer_changes` | **live** |
| Attention budget | `push_budget.go` leaky bucket (6/min) | **live** |
| Embedding stack | `ai-gateway InferenceService/Embed`; `nomic-embed-text`, 768-d, L2, 87 ms/single, 168 ms/32 | **live, unused here** |
| Vector precedent in SQLite | `code-facts/internal/retrieval/sqlite_vectors.go` | **live** |
| LLM rubric scorer | `ai-gateway InferenceService/Decide`, `DECISION_QUESTION_TYPE_SCORE` | **live, unused** |

The gap is not capability. It is that none of it is projected to the operator,
and the two quality rule sets (Hub + steward) disagree.

---

## 2. Relatedness — suggest kin, never merge

### 2.1 Three tiers of truth

| Tier | Signal | Authority | Cost |
|---|---|---|---|
| **Claim** | `decision_family` + `decision_cause` from the producer | Authoritative. Groups the list. | none |
| **Identity** | `subject_key` (same source + event + key) | Authoritative. One open ask per subject. | none |
| **Suggestion** | Lexical score, then embedding score | Advisory. Never merges, never refuses on its own. | ~0 ms / ~90 ms |

The desk shows suggestions *under* the claim: “Same cause” when a family exists,
“Similar wording 0.72” when only a suggestion does. The operator never sees a
suggestion presented as a fact.

### 2.2 Tier 0 — lexical (keep, and make it one function)

Move `subjectWords`/`similarSubject` out of `packet_quality.go` into
`api/internal/related/` and expose:

```go
type Basis string // "subject", "family", "lexical", "semantic"

type Match struct {
    AskID string
    Score float64
    Basis Basis
}

func Score(a, b Text) (float64, Basis) // deterministic, no I/O
```

Keep the existing Jaccard ≥ 0.5 refusal in the write path (`ErrRelatedDecision`)
exactly as it is. Tier 0 is the floor: it works with every service down, which
is the scenario's stated posture.

### 2.3 Tier 1 — embeddings in the Hub's own SQLite (recommended)

Do **not** call `search-hub query` on a read path. Measured 6–9 s end to end.
Do **not** make Qdrant a required dependency: the PRD's OT-P0-009 says no
required resource. Use the in-repo pattern that avoids both.

**Vector table** (mirror `code-facts/internal/retrieval/sqlite_vectors.go`):

```sql
CREATE TABLE ask_embeddings (
  ask_id      TEXT PRIMARY KEY REFERENCES asks(id) ON DELETE CASCADE,
  model       TEXT NOT NULL,          -- 'nomic-embed-text:latest'
  dimensions  INTEGER NOT NULL,       -- 768
  recipe      TEXT NOT NULL,          -- 'nomic|q=search_query: |d=search_document: '
  text_hash   TEXT NOT NULL,          -- drift gate
  vector      BLOB NOT NULL,          -- little-endian float32, dim*4
  created_at  TEXT NOT NULL
);
CREATE INDEX idx_ask_embeddings_model ON ask_embeddings(model, dimensions);
```

**Embed text** (one row per decision, written once per revision):

```
search_document: <question>
<recommended label>
<big_picture, first 400 chars>
```

Query side embeds `search_query: <question>` only. Task prefixes are what the
shared engine uses for nomic and are worth the one-time reindex.

**Write path** — never inside the answer request. After `askSubjectOnce`
commits (`ask_subjects.go:177`), enqueue the ask id on the existing worker
channel. A failed or absent embed leaves the row missing; Tier 0 still answers.
Batch at most 64 texts per call (gateway limit is 1 MB), and treat
model/dimension mismatch as “no vector”, not an error the operator sees.

**Read path** — `related(ctx, askID, limit)`:

1. Resolve the ask and its audience with the existing `AskAudience` rules.
2. Embed the question (one call, ~90 ms) or, if unavailable, use Tier 0.
3. Cosine over at most a few thousand open rows (10k × 768 float32 = 30 MB,
   single-digit ms in-process).
4. Return at most 3 matches **the audience may already see**. Keep the
   content-free `ErrRelatedDecision` behaviour for other conversations.

**Bands** (calibrated on this host, nomic, symmetric):

| Cosine | Label | Behaviour |
|---|---|---|
| ≥ 0.80 | **Same cause** | Offer “answer together”; still requires a producer family claim to merge |
| 0.68–0.80 | **Similar wording** | Show the score, nothing more |
| < 0.68 | — | Ignore |

Do not use an absolute relevance floor to reject junk: measured gibberish scores
0.54–0.58 against `vrooli-docs`, overlapping real answers. Use the relative gap
from the top candidate plus a lexical check.

**Cost.** ~90 ms once per opened decision, off the list path. Locally free.
On a CPU-only fleet node, slower; cache the row and degrade to Tier 0.

### 2.4 Tier 2 — Qdrant (only if the corpus outgrows SQLite)

If decisions ever exceed ~50k open+recent rows, adopt
`github.com/vrooli/ai-go/search`: `NewDenseEngine(cfg, "notification-hub_decisions")`,
`NewDenseBinding`, `NewReconciler`, `NewSyncLoop`, `NewService` with the Tier 0
lexical scorer as `TextFallback`. Declare `qdrant` optional
(`required:false, startup_policy:try_start`), never required. Same collection
naming as `signal-inbox_signals`, `swarm-manager_records`.

### 2.5 What this unlocks in the desk

- The guard pair reads as **one card with two checks** because identical question
  text already scores 1.0 lexically; a declared family makes it intentional.
- “Answered before” comes from `SubjectRecurrence.previousAnswers`, which the
  Hub already sends and the UI does not render.
- Cross-source kin (“Portal screenshot access” ↔ “Search presence retention”)
  appear only as suggestions, with a basis and a score.

---

## 3. Readiness — one quality model, scored before it costs you

### 3.1 The problem

Two rule sets exist and disagree:

- Hub: ~25 `PacketWarning` codes, three blocking (`missing_recommendation_reason`,
  `explainer_restates`, `page_unreadable`), evaluated at send.
- Steward: `page_lint` + `_f206_page_problems`, ~15 more checks (screenshots,
  jargon, first-screen command, root cause, batch groups, repeated cards),
  evaluated before a page is accepted.

They share vocabulary by accident. Neither is stored, neither is aggregated by
producer, and the operator never sees either. Research asks get **advice**, not
refusal, so 11 of 18 open research pages would fail the Hub's own check.

### 3.2 One taxonomy, one package

Create `packages/api-core/decisionquality` (pure Go, no I/O):

```go
type Owner string // "producer" | "page" | "hub"

type Finding struct {
    Code     string   // stable, e.g. "no_recommendation_reason"
    Severity Severity // advice | blocking
    Field    string   // question, options, page, packet, ...
    Owner    Owner
    Message  string   // what to do, in the producer's language
}

type Report struct {
    Grade    Grade    // ready | needs_work | refused
    Score    int      // 0-100, for trends only; never shown alone
    Findings []Finding
    Basis    []string // which checks ran, so an absent check is visible
}

func Assess(d Decision) Report
```

Merge the two code lists, rename for the producer, and keep a mapping so
existing CLI output does not break. The Hub and the steward both call `Assess`;
the steward's `page_lint` becomes a thin adapter, not a second truth.

**The smell list** (initial set; every one is already measured somewhere):

| Group | Codes |
|---|---|
| Ask | `no_question`, `question_too_long`, `internal_id_in_question`, `path_in_question` |
| Choice | `no_options`, `option_without_effect`, `no_recommendation`, `no_recommendation_reason` |
| Consequence | `no_if_no_answer`, `no_big_picture` |
| Page | `page_missing`, `page_restates`, `page_unreadable`, `page_no_picture`, `page_picture_uncropped`, `page_no_viewport`, `page_network`, `page_form_control` |
| Packet | `packet_missing`, `unattached_path`, `command_hidden` |
| Repeat | `recurring_subject`, `repeat_without_cause_fix` |
| Batch | `batch_without_groups`, `batch_as_cards` |
| Language | `jargon`, `author_note` |
| Relation | `related_open_decision` |

### 3.3 Persist it, per revision

```sql
CREATE TABLE ask_quality (
  ask_id     TEXT NOT NULL REFERENCES asks(id) ON DELETE CASCADE,
  revision   INTEGER NOT NULL,
  grade      TEXT NOT NULL,
  score      INTEGER NOT NULL,
  report     TEXT NOT NULL,   -- JSON: findings + basis
  checked_at TEXT NOT NULL,
  PRIMARY KEY (ask_id, revision)
);
```

Written when the ask is sealed and on every revision. The list projection gains
`quality_grade` so the desk can badge “needs work” without a second read.

### 3.4 Make the score pay

- **Agent-sent asks:** `refused` blocks the send (this already happens for three
  codes; extend it to the merged set).
- **Event/service asks (research, portal, steward):** keep advice at send so a
  producer is never silenced, but:
  - the decision **does not count toward operator load** until it is `ready`.
    It sits in a producer queue, not the operator's count. That is the bandwidth
    accounting: a malformed packet does not get a seat.
  - `decision-load` and the badge use ready decisions only.
  - the producer's source page shows “3 decisions returned for framing”.
- **Producer scorecard:** per `source`, rolling 30 days — readiness rate, top
  three codes, median revisions to ready. The steward turns a sustained
  producer score below threshold into a `decision-quality` risk with an
  owner-brief packet, not another operator decision.
- **Feedback:** the steward's `decision` delivery already writes to producers;
  add a `quality` block to that payload so the next decision is better, not just
  rejected.

### 3.5 Optional LLM rubric (advisory only)

`ai-gateway InferenceService/Decide` already supports
`DECISION_QUESTION_TYPE_SCORE` with levels and a choice. Use it as a **second
opinion** on the two things regex cannot judge:

- “Can this be answered without opening anything?” (1–5)
- “Is the question free of acronyms and build vocabulary the operator would
  have to decode?” (1–5)

Constraints: batch, 2 s timeout, local-only profile, never blocking, never in
the send path, stored as `advice` findings. The deterministic checks remain the
gate; the model only adds a sentence to the producer's feedback.

---

## 4. Aftermath — close the handshake, or say it stalled

Right now an answer is recorded and the desk goes quiet. The steward has
`answer-waiting` (five answers, 53 h old, no live window) but the operator never
sees it where the answer was given.

### 4.1 A follow-through report, not a guess

Add an optional producer event, `operator_decision.followthrough.v1`, with:

```json
{
  "ask_id": "...", "state": "queued|running|done|stalled|failed",
  "owner": "scenario:agent-manager", "run_ref": "...",
  "started_at": "...", "finished_at": "...", "bound_seconds": 3600,
  "result_url": "...", "summary": "one plain sentence"
}
```

The Hub stores the latest report per ask and projects:

```proto
message Followthrough {
  string state = 1;      // empty means no owner has reported
  string owner = 2;
  string run_ref = 3;
  string started_at = 4;
  string finished_at = 5;
  int32  bound_seconds = 6;
  string result_url = 7;
  string summary = 8;
  string reported_at = 9;
}
```

`Ask.followthrough` is filled by `GetAsk` only.

### 4.2 What the desk says

| State | Desk |
|---|---|
| none | “No owner has reported a result.” (never inferred) |
| queued | “Owner queued · starts within the bound” |
| running | “Running since 14:02 · bound 1 h” |
| done | “Closed · result” |
| stalled | “Answered 53 h ago, no live window is reading it · nudge” |
| failed | “Stopped · reason” |

The steward's stalled list and the Hub's projection must be the same rows; the
steward remains the place that launches the executor, the Hub the place that
shows it.

### 4.3 Do not invent progress

No percentage, no ETA. A bound is a fact; a progress bar is a guess.

---

## 5. Impact — say what waits on the choice

Add three optional facts to the ask contract (producer-supplied, never derived):

| Field | Meaning | Shown as |
|---|---|---|
| `unblocks` | One phrase: what can start once answered | “Unblocks: the December release cut” |
| `cost_of_waiting` | One phrase: what degrades per day/week | “Each day: two runs stay queued” |
| `reversible` | existing bool | “Reversible” chip |

These give “why now” teeth without a synthetic score. If absent, the desk says
nothing rather than guessing.

---

## 6. Grouping and batch — the honest mechanics

- **One ask, several questions** already works (`questions[]`). The focus screen
  is that feature, not a new one. “Accept all recommendations” already exists in
  the answer form.
- **Several asks, one cause** is a family. The UI already groups by
  `decisionFamily`; the missing piece is that producers do not populate it and
  nobody answers a family in one action.
- Add `AnswerAsks` (transactional, all-or-nothing) or an explicit client loop
  that reports per-ask success. The desk must never imply that one answer
  covered three asks when it did not. Show progress: “2 of 3 recorded”.
- A family is a **claim**, a similarity is a **suggestion**. Never auto-merge on
  a score, even at 0.99. The producer or the owner can promote a cluster to a
  family; the Hub records who did.

---

## 7. Existing features the redesign must keep

From the UI inventory. The mockups show the ones that fit; none should be lost
in implementation.

- Updates is **read-only**; viewing never changes state. Delivery is its second
  tab. History keeps every lifecycle state.
- Needs-you badge = decisions + pending native consents; `totalAwaiting`
  excludes consents. The desk must keep the consent group visible and separate.
- Closing-soon tier from the Hub's `closing_window_seconds`.
- Facets sheet with counts (urgency, sender), URL-backed filters and sort.
- Return-position and focus restoration; prev/next follows the originating list.
- Packet: attachments, revision history, context requests, drafted answers,
  uncertain-send freeze, status “mark read”, root password/read fallbacks.
- Explainer sandbox, kit templates, region-feedback compare, observed packet.
- Settings: reach, devices, quiet hours, escalation, sources, appearance,
  language (including RTL), session and root enrollment.
- PWA install, app badge, service-worker update status, deep links, test-mode
  lease banner.
- **Guided paged answer flow is implemented but unreachable** (`paged=false`).
  Turning it on is a one-line win that matches the phone design.

---

## 8. Implementation order

Each slice is independently useful and testable.

| # | Slice | Acceptance |
|---|---|---|
| 1 | **Family contract live.** Rebuild the Hub so `decision_family/cause/kind` persist; producers set them; list projection carries them. | Two guard asks with one family render as one card; tests cover grouping and the fallback when no family is set. |
| 2 | **Readiness projection.** `packages/api-core/decisionquality`, merged codes, `ask_quality`, `quality_grade` on the list. | The 11 refused research pages are classified before send; a producer sees its top three codes; no send is blocked for event asks. |
| 3 | **Count by readiness.** Badge, `decision-load`, and `totalAwaiting` count ready decisions; unready go to a producer queue. | The count matches across Hub, brief, `TODAY.md`, `DECISIONS.md`; a returned decision is visible to its producer. |
| 4 | **Lexical related projection.** One `related` field with basis and score; `previousAnswers` rendered. | Opening a decision shows same-cause and similar-wording matches, all audience-filtered; no cross-conversation leak. |
| 5 | **Semantic tier.** SQLite vectors + optional ai-gateway embed, off the request path. | With ai-gateway down, the desk still shows lexical matches and says “semantic unavailable”; with it up, the guard pair and portal trio are found. |
| 6 | **Aftermath.** `Followthrough` projection + producer event; steward stalled rows link into it. | A stalled answer shows where it was answered; a done run shows its result; no state is inferred. |
| 7 | **Impact facts + batch answer.** `unblocks`, `cost_of_waiting`; `AnswerAsks` or a verified loop. | The desk shows the cost of waiting only when supplied; answering a family reports per-ask progress. |
| 8 | **Guided paged flow on.** Flip `paged` for phone. | Phone answer is one decision per screen with progress, matching the mockup. |

Sequencing note: 1–3 fix the economics, 4–5 add discovery, 6–8 complete the loop.

---

## 9. Risks and non-goals

- **Do not call `search-hub query` on a read path.** 6–9 s. It is a discovery
  surface for agents and humans, not a similarity API.
- **Do not require Qdrant or Ollama.** Optional dependencies only; the desk must
  work with every service down, as the scenario's PRD promises.
- **Do not auto-merge on similarity.** Cross-tenant leakage and wrong
  authorisation are worse than a duplicate.
- **Do not show a bare model score.** Bands and basis labels only.
- **Do not let quality scoring block a producer silently.** Return it with the
  reason and a route to fix it.
- **Do not fake progress, time estimates, or impact.** The mockups deliberately
  omit them.

---

## 10. Quick wins while the above lands

1. Flip `paged` on for the phone answer flow (already implemented).
2. Render `recurrence.previousAnswers` (“last answer: hold, 12 Nov”).
3. Link the `packet` field in the detail (it is sent and unused).
4. Add “Delivery” diagnostics from `RoutingService.ChannelsStatus` to the
   delivery page (“this channel is not permitted on this device”).
5. Ship Inter with the app so `--font-sans` stops depending on the host's
   fallback. The proposal folder carries the OFL font for exactly this reason.
