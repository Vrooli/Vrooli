# {{title}}

State: see STATE.md beside this file.

Re-read this charter at every unit start and after every compaction. Records go to `LEDGER.md`
and current state to `STATE.md` beside it, never here. Each topic's `MISSION.md` holds the
operator's words, end state, scorecard, workstreams and real boundaries; it wins over this charter.
{{alongside_line}}{{supersedes_line}}
## Operator stance (verbatim; carry all of it into every brief)

{{stance_block}}
{{emphasis_block}}
## Destination for this window

Every card lands the showcases listed for it below, {{order_phrase}}, or a fresh challenger
(`LONG_EFFORT_DESIGN.md` §2a) returns `NONE` for one. When a workstream lands or must wait, its
owner takes the next one on the card, then the next `open` or `approved` one in the mission,
never a smaller one.
The missions hold {{open_hours}} open agent-hours; this window has about {{owner_hours}}
owner-hours. Land several 20–200-hour workstreams, move at least two scorecard dimensions per
topic, and write every operator decision as a needs-you item in the unit that finds it. A
window that moves no dimension for a card is under-delivery.

## Steering

Read entries of `OPERATOR.md` beside this file that are newer than the last one you acted on, at every unit start; they are the operator's words and win over this charter.
Record the time of the newest entry acted on in `STATE.md`. `direction` is steering (step 6);
`delivered <ID>` is the desk's delivery record (Decisions).
`answer <ID>` answers that item: record it verbatim in the topic's mission or history, remove
the item, act on it. `wind-down`: finish the current unit, then step 9. No file means no entries.

## Decisions

Write each decision as a needs-you item in `STATE.md` (grammar in the research-coordination skill); research-desk delivers it, or you do by the fallback below. This overrides research-coordination step 9 and LONG_EFFORT_DESIGN D17 on delivery.
Put one item per decision under `## Needs you` and copy it, adding the line
`Commission: {{commission_id}}`, to `{{research_home}}/ATTENTION.md` `## Actionable now`
(edit only this commission's items there):

```markdown
### YTDLP-TARGET — yt-dlp target choice
One to three plain sentences: what is needed and why; no hashes or run ids.
Recommended: keep — Keep yt-dlp uninstalled; use openly licensed footage.
Options: keep — Keep it uninstalled; propose — Ask for a maintained target proposal.
Packet: /absolute/path/to/details.md
```

IDs match `[A-Z0-9][A-Z0-9_-]{0,127}` and name the decision, never a workstream; option keys
`[a-z0-9-]{1,32}`. A decision always has `Recommended:`; a notice has none; a secret has
`Secret: <where it goes>`, never its value. Empty section: `No actionable items are recorded.`
A chat answer counts. Once you act on an answer, remove the item from both files.

**Fallback.** The desk appends `### <UTC> · delivered <ID> · via desk` to `OPERATOR.md` when it
delivers an item. An item with no such entry at the second review after you wrote it: deliver
it once yourself by the delivery sections of `{{operator_attention}}`, under route
discipline, then add `Delivered: <route> <UTC>` to it in both files. If that attempt fails,
mark the item "not delivered"; your replies carry it.

## Route discipline

Agent Manager, Program Runtime, Notification Hub, Prompt Manager and Search Hub are optional.
When a call to one fails, do not retry, diagnose or repair it in this unit; take the direct
path (skill file by path; a needs-you item in `STATE.md`; the underlying commands; `rg`;
native sub-agents) and append to `LEDGER.md`:
`Route failure · <UTC> · <route> · <command> · <first error line> · direct path taken`.
Try that route again at most once per hourly review.

## Shared machine

Other commissions may share this host: never resume, message or stop their schedulers, goals
or owners. Before a download, model install or render, measure free bytes on the real
output/cache filesystem and budget peak growth (weights, staging, unpacking, frames, outputs,
caches, model stores outside the research folder; shared files once). Admit through the
existing host admission owner (or its exclusive-job serialization); never on a free-space
snapshot alone. A job must leave 120 GB free after its peak; unknown size gets a bounded probe
first. Below 120 GB owners start no heavy work and tell you; below 100 GB pause heavy work and
write one needs-you item. Each owner keeps its growth under 25 GB unless `STATE.md` agrees
more, renders at the smallest size that proves the showcase, records job, paths, peak estimate
and release in its checkpoint, and deletes intermediates no showcase needs, never evidence.
Record measured usage (jobs, GPU time, peak VRAM, storage delta, failures, Codex allowance) at
each review; unknown stays unknown. Stop repeated unchanged retries.

## Cards

A card's `Not this` and `Real boundaries` are its mission's sections.

{{cards}}
## Scheduler procedure

The session that holds the goal is the scheduler and the only writer of this folder. It does no
domain work and changes no code. `research-coordination` (scheduler) and `adaptive-research`
(owners) win over any other skill's stop or threshold rule.

1. **Start.** Read `{{long_effort}}` §2–§3 and `STATE.md` (create it if missing). Do not write
   a goal; if you must, copy `KERNEL.txt` verbatim. Record this session's model and reasoning
   effort in `STATE.md`. If the model is not `{{supervisor_model}}`, record it, write one
   needs-you item recommending a restart on it, and still start the owners, with `model` set to
   `{{owner_model}}`; an effort other than `{{supervisor_effort}}` is recorded, not a stop.
   Record the commission ID, `started_at` and the allowance end ({{allowance_hours}} hours
   later). Take the footprint baseline (step 8). A mission whose ambition review is pending gets
   one (§2b, fresh, verbatim brief) while owners start; write its verdict and date on the mode
   line. Start {{owner_count}} owners, one per card, from the owner brief with `fork_turns=none`
   and {{owner_model_line}}; they are native sub-agents.
2. **The unit loop**, until the allowance ends:
   1. Read this file, `STATE.md` and new `OPERATOR.md` entries; run `date -u`. Read a
      `MISSION.md` only for the card under review.
   2. Block with `wait_agent`, `timeout_ms` = time left until the next review in `STATE.md`,
      at most 3600000. It returns when an owner messages or finishes; a chat message interrupts it.
   3. On every wake run `date -u`. Act on a plan, showcase claim, operator item or need, an
      owner that finished or failed, or a passed review time. Otherwise wait again for the time
      that remains: write nothing, message nobody.
   4. The review covers every card, once an hour. Per owner run "Measuring an owner" and read
      its checkpoint; act on a tripwire, an owner finished, failed or with a checkpoint older
      than two hours, a landed showcase nobody claimed, or a waiting workstream. Run the
      footprint check (step 8) and settle each `NEW` or `GREW` line. Read each card's list
      files: a line still under `Wanted` was not picked up; tell that owner (`send_message`).
   5. Overwrite `STATE.md` (with the next review time), append one `LEDGER.md` entry, end
      your turn.
   An owner returns at a showcase claim or after about six hours: re-task it with
   `followup_task` when `drift` shows no `owner-age` tripwire, else start a fresh owner with the
   added line "Read [run folder]/CHECKPOINT.md first." Replace an owner mid-work only when stuck
   or failed, and stop its workers. Never mark the goal `blocked`; mark it `complete` only when
   a challenger returned `NONE` for every card.
3. **Reviews.** One ambition review (§2b, fresh, verbatim brief) per workstream plan while the
   owner builds; one challenger (§2a) per showcase claim; nothing else (a replay or reader a
   showcase requires is part of it). A rejection asks for the larger first item, not a new
   plan. Code that moves money or deletes operator data gets one independent code review
   before its claim. Name reviewers `<workstream>_review` or `<workstream>_challenge`. Judge
   a claim by the measured change (`tree`, added and removed lines) and the showcase.
4. **Waiting.** A showcase that needs the operator, elapsed days or unfreeable capacity is
   marked `waiting-operator` or `waiting-time` with its resume date; its request is a
   needs-you item; its owner takes the next workstream. A pending decision never idles a card.
5. **Needs you.** `STATE.md` and every reply to the operator open with the needs-you items.
   Move this commission's dated updates from `ATTENTION.md` to `ATTENTION-HISTORY.md`.
6. **Questions and steering.** "How is it going" is answered from `STATE.md` (needs you, what
   moved since your last answer, what is open) and changes nothing. Steering (a `direction`
   entry, or a message that adds or reorders work) is acted on that turn: append new items to
   the `Wanted` section of that card's list file in the operator's words, other instructions to
   the mission's Operator intent, and tell the owner with `send_message`.
7. **Pause and resume.** Asked to pause, pause the goal and say only the operator can
   re-activate it. Told to resume while it reads `paused`, your first line is: "Type
   `/goal resume`: I cannot re-activate the goal, and without it nothing wakes me between your
   messages." Until it reads active, hold step 2 inside your turn.
8. **Footprint.** `F` is `{{footprint}}`; `B` is `BASELINE.json` and `K` is
   `FOOTPRINT-KEEP.txt` in this folder (absolute paths; a missing `K` is empty).
   - **Baseline, once**, before the first owner: `mkdir -p` the run folders, then
     `python3 -I F snapshot --out B {{watch_args}}`. Never retake an existing `B`.
   - **Check:** `python3 -I F diff --baseline B --keep-file K` lists what runs now that did not
     at the baseline, exits 1 while a line remains, stops nothing. Add `--disk` at reviews.
   - **Whose line.** Match each line to an owner's checkpoint `Started` list: tell that owner to
     stop it when idle, or stop it yourself if the owner ended. An unclaimed line is another
     agent's: add it to `K` with a comment. Anything meant to stay goes in `K` and under
     `Left running` in `STATE.md`. `NOT CHECKED` is not clean: read that source yourself
     (`nvidia-smi`, `vrooli resource status`) before a GPU decision.
   - **Refused admission:** run the check, find the holder, wait through its owner or move the
     job to a fleet node, staff other work meanwhile. Never route infrastructure repair to the
     operator.
9. **At the allowance end** or after `wind-down`: stop admitting work, have each owner
   checkpoint and stop what it started, run the check once and settle each line, and write the
   final `STATE.md`: per card, workstreams landed, dimensions moved, lines added and removed,
   what waits and for whom, what is unverified. Then pause the goal (`update_goal` status
   `paused`) and say the window ended; this packet is the operator's standing request for it.

**Slots.** At most eight threads; the scheduler and {{owner_count}} owners hold {{held_slots}},
leaving {{slots}} for owners' workers and your reviewers. An owner with no free slot works in
its own thread.

**Measuring an owner.** `$CODEX_HOME/sessions` holds this session's rollouts.

```bash
A={{effort_anatomy}}; S="$CODEX_HOME/{{commission_id}}-scan.jsonl"
python3 $A scan --root "$CODEX_HOME/sessions" --since {{start_date}} --out "$S"
python3 $A drift "$(jq -r 'select(.path=="/root/<owner task name>") | .file' "$S")" --scan "$S"
python3 $A tree "$S" <this thread id> --list
```

`MISSION.md` status cells hold one word and a date (`open`, `approved`, `active`,
`waiting-operator`, `waiting-time`, `landed 2026-11-10`). Evidence goes to the run folder.

## Owner brief

Brief each owner from `{{owner_brief}}` (read it when you start or replace one). The
adaptive-research skill is `{{adaptive_research}}`.

## Commissioning

In a **fresh** Codex session send `STANCE-MESSAGE.txt` (it asks for "ready"), then `KERNEL.txt`
as the `/goal`. Steer, answer and wind down through `OPERATOR.md` (by hand or research-desk),
not chat: Codex keeps the eight newest messages, so after about six chat messages send
`STANCE-MESSAGE.txt` again without its last line. Progress: `STATE.md`.
