---
name: "research-review"
description: "Run the operator's review of selected research topics: verify what was accomplished from the files, show media and rendered artifacts for exact-artifact feedback, diagnose under-delivery, tidy the topic folders by moving (never deleting), record the operator's verdicts verbatim, carry them to live owners the same day, apply authorized improvements to research organization and mission guidance, and suggest the next commission's emphasis."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["practice"]
  tags: ["research", "review", "missions"]
  icon: "clipboard-check"
  status: "active"
  revision: 3
  createdAt: "2026-11-11T00:00:00Z"
  updatedAt: "2026-11-12T01:00:00Z"
  requires:
    commands: ["python3"]
  origin:
    kind: "authored"
---
## Practice focus: Research review

The operator reviews a few topics at a time. You gather what was actually accomplished, put it
in front of them plainly, record what they decide in their words, and realign the missions to
it. A request to improve a topic authorizes reversible organization and guidance edits within
that topic, including improvements inferred from the initial request and later feedback.
Apply them and report the concrete changes; do not require another confirmation merely
because the review revealed useful edits. A review-only request stays read-only. Clarify only
material ambiguity that prevents a responsible edit; continue independent improvements.
This workflow does not launch research, implement product capabilities, change commission
allowances, contact clients or publish material unless separately authorized. Research home `H` = `~/.vrooli/research`; the
file grammar is research-desk `docs/reference/FILE-CONTRACT.md` (§4 verdicts, §5 needs-you).

Route discipline: research-desk, Search Hub, Prompt Manager and Program Runtime are optional
lenses (`research-desk changes <topic>`, `program-runtime.research-workspace-review`). When one
fails, do not repair it: read the files directly and note the failure in one line in `REVIEW.md`.

### 1. Gather evidence (per selected topic `T`)

1. `python3 <research-coordination>/scripts/mission_check.py H --topic T --json`: mode,
   workstream rows (id, destination, showcase, status) and errors.
2. Commissions that touched `T`: `rg -l "topics/T/" H/cohorts/*/README.md`. For each, its
   `STATE.md` (final state, what waits) and the `LEDGER.md` entries naming `T` (search; never
   read a ledger whole).
3. Run folders `H/topics/T/runs/*/CHECKPOINT.md`: changed, verified, unverified, showcases.
4. Verify every claimed landing by opening its artifact: it exists, runs or plays, and matches
   the showcase. A claim you cannot open is "claimed, unverified", never "landed".
5. Live and paused commissions: `research-desk commissions --state live` and `--state paused`;
   without the desk, any commission whose allowance has not ended, whose `STATE.md` changed in
   the last 70 minutes or says paused. When unsure, count it.
6. A commission that moved no scorecard dimension for `T`, or landed less than its card, gets
   `effort-diagnosis` (its scheduler and owner sessions) before you write anything about it.

### 2. Write `H/reviews/<id>/REVIEW.md`

`<id>` = a short slug of the topics plus the local date (`video-books-20261112`; add `-2`
if taken). Per topic, in this order, each line with an absolute path and a date:

- **Landed** — workstream, showcase, the artifact you opened.
- **Claimed but unverified** — what was claimed, where, and what you could not confirm.
- **Open** — live workstreams not started or in progress, and what waits on the operator.
- **Grew** — new workstreams, list rows, folders and files since the last review (sizes);
  clutter that makes a fresh owner read more than it needs.
- **What failed organizationally** (under-delivery only) — the diagnosed cause in the setup
  (charter, card, brief, route, capacity, supervision), not the owner's account, and the change
  it makes to the next packet.
- **Organization changes** — applied improvements and any deferred moves into `H/topics/T/history/` that would leave the topic folder
  showing a concise entry point, mission, useful current guidance and current runs. Keep durable
  synthesis easy to find; do not archive useful guidance merely to achieve a fixed folder shape. Never a deletion. Nothing inside
  a run folder of a live or paused commission (step 1.5). Never move a file whose name or path
  appears, as a link or bare text, in a live or paused commission's charter, any `MISSION.md`
  or any `CHECKPOINT.md`: check each with `rg -F <file name>` over those files first.
- **Decisions needed** — only unresolved decisions outside existing authority, with recommendations.
  Do not turn routine edits or already supplied feedback into approval requests.

When requested, or when the work includes media or rendered artifacts, also write
`H/reviews/<id>/REVIEW.html`, one phone-friendly HTML file with inline styling and scripts. Embed review-sized media directly
when practical; link large artifacts by relative paths and explicitly identify external file
dependencies rather than calling the result self-contained. Show each exact artifact (`<video controls playsinline preload="none">`, `<audio>`, `<img>`, or a
link for a document) by a path relative to the file, with its absolute path, sha256, status and
what to look for; then a feedback section per artifact (aspect, timestamp or passage, optional
0–10 score, words) that exports one JSON file the operator saves beside it. Nothing is sent or
saved automatically. Pattern: `H/topics/ai-video-production-and-funnels/REVIEW.html`.

End with **Next commission**: suggested topics, emphasis and `read_first` for `packet.py`,
quoting the operator's words where they set it. Keep the file short enough to read on a
phone: link evidence, do not paste it.

### 3. Record verdicts verbatim

Append each verdict to `H/reviews/<id>/VERDICTS.md` (create it with `# Verdicts — <id>`):

```markdown
### 2026-11-12T10:14:03Z · verdict ai-video-production-and-funnels · via hand
The operator's words, verbatim, any number of lines.
```

`via desk` when research-desk wrote it, `via hand` when you record it from the conversation.
Append only; a correction is a new entry. Never paraphrase inside an entry.

**Same day to live owners.** For each verdict on a topic that a live or paused commission
covers, add to `REVIEW.md` under **Direction for live commissions** the commission id and the
exact entry body: the verdict verbatim, then one line on what changes for the owner. The
operator applies it (`research-desk say <id> direction`, or by appending a §4 `direction`
entry to `H/cohorts/<id>/OPERATOR.md`); `OPERATOR.md` is the operator's file, not yours. If
that charter lacks the Steering line (older charters), say first that it must be enabled: the
desk appends it on the operator's action, or the operator appends the Steering sentence of
`research-coordination/templates/charter.md` to the charter by hand.

### 4. Apply authorized improvements and iterate

- Apply reversible cleanup within the requested scope without a second approval. Preserve
  originals and evidence; record each `from → to` in **Moved** so it can be reversed. The
  live-work and reference protections above still apply. Leave protected files in place.
- Improve the topic's existing entry point, current state, method, results and mission as
  needed. Separate historical evidence from current guidance, remove contradictory current
  claims, and give a fresh owner a short starting path. Prefer editing existing sources over
  adding overlapping documents. Preserve unresolved findings and failed experiments.
- Turn feedback into operational research guidance: desired outcomes, ambitious workstreams,
  falsifiable experiments, acceptance criteria, exact-artifact evidence and review gates.
  Include reusable capability development and maturation where relevant; avoid reducing the
  task to its easiest local demo or adding infrastructure without a demonstrated purpose.
- Record operator feedback verbatim in verdicts and distinguish it from your interpretation.
  Link the verdict to a concise change record with rationale and any deferred decisions.
  Proposals under `proposals/T.md` are for unresolved changes, not a mandatory approval stage.
- Before editing `MISSION.md`, re-read it (a supervisor may have changed it), change only
  relevant sections and rows, preserve status cells unless new evidence warrants an explicit
  status correction, and run `mission_check.py`. Never silently shrink ambition, discard
  original intent or overwrite another owner's current work.
- Follow-up feedback continues the same authorized improvement cycle: update guidance and
  affected reports, keep earlier verdicts intact, and explain what changed. Regenerate stale
  HTML after substantive changes; check links, artifact labels and phone-width layout. Report
  validation limits honestly. Preparing live-owner direction does not authorize writing the
  operator-owned steering file; retain the routing boundary in step 3.

**Exit:** `REVIEW.md` (and `REVIEW.html` for media), `VERDICTS.md`, applied changes, any unresolved proposals and moves are
on disk; each verdict on a live topic has its direction entry ready; the operator has the
next-commission suggestion; nothing was deleted, and no file a live or paused commission names
was moved.
