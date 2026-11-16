# {{title}}

State: see STATE.md beside this file.

Re-read this charter and `STATE.md` at every unit start and after every compaction. Records go to
`LEDGER.md`, state to `STATE.md`. Each topic's `MISSION.md` holds the
operator's words, end state, scorecard, workstreams and real boundaries; it wins over this charter.
{{alongside_line}}{{supersedes_line}}
## Operator stance ({{stance_label}}; carry all of it into every brief)

{{stance_block}}
{{emphasis_block}}
## Destination for this window

Every card lands the showcases listed for it below, {{order_phrase}}, or a fresh challenger
returns `NONE` for one. When a workstream lands or must wait, its owner takes the next one on the
card, then the mission's next open row or Later line, never a smaller one. The missions hold
{{open_hours}} open agent-hours; this window has about {{owner_hours}} owner-hours. Land several
20–200-hour workstreams and move at least two scorecard dimensions per topic. A window that moves
no dimension for a card is under-delivery.

## Steering

Read entries of `OPERATOR.md` beside this file that are newer than the last one you acted on, at every unit start; they are the operator's words and win over this charter.
Record the newest entry acted on in `STATE.md`. `direction`: act on it that turn (step 5).
`answer <ID>`: record it verbatim in the topic's mission or history, remove the item, act on it.
`delivered <ID>`: the desk delivered that item. `wind-down`: finish the unit, then step 8.
Read entries in `STEWARD.md` beside this file at every unit start. Record the newest entry acted on in `STATE.md` with its date and attribution. Apply the authority order explicitly: `OPERATOR.md` entries override everything; `STEWARD.md` entries guide the work when they do not conflict with operator entries; where neither file decides, use supervisor judgment. Attribute steward guidance in briefs and decisions as `Steward · <date> · <attribution>` so its source remains clear.
At start and each review, drain each card topic's inbox: act on each entry, then
`python3 -I {{inbox}} drain <topic folder> --entry <stamp> --did "<what you did>"` (`list` shows them).

## Decisions

Write each decision as a needs-you item in its topic's `DECISIONS.md` (grammar in the research-coordination skill); research-desk delivers it, or you do by the fallback below.
An item about the whole commission goes to `{{research_home}}/ATTENTION.md` `## Actionable now`
with `Commission: {{commission_id}}`.

```markdown
### YTDLP-TARGET — Keep the video download tool uninstalled?
Two to six plain sentences for a stranger: what this is about, why only the operator
decides, what each answer leads to, what happens without one. No IDs, run names or paths.
Recommended: keep — Keep it uninstalled and use openly licensed footage instead.
Options: keep — Keep it uninstalled; propose — Ask for a maintained target proposal.
Packet: work/YTDLP-TARGET/BRIEF.md
Explainer: work/YTDLP-TARGET/page.html
```

IDs match `[A-Z0-9][A-Z0-9_-]{0,127}`; option keys `[a-z0-9-]{1,32}`. A decision has
`Recommended:` and `Packet:`; a notice has neither; a secret has `Secret: <where it goes>`, never
the value. Before asking, write the packet (what executes on each answer, and its proof) and the
page (`{{operator_attention}}`, "Decision pages") in the topic's `work/<ID>/`. Copy the file to a
draft, edit it, then `python3 -I {{needs_you}} write --file <draft> --section "Needs you" --target <file>`.
An answer arrives as an `answer` entry in that topic's `INBOX.md`: add it to the packet under
`## Operator answer`, start a fresh executor with "Execute the approved decision `<ID>`. Read
`<abs>/BRIEF.md`; the answer is `<key>`." plus the stance, move the item to the topic's
`history/DECISIONS.md`, then drain the entry.

**Fallback.** The desk appends `### <UTC> · delivered <ID> · via desk` to `OPERATOR.md`. An item
with no such entry by your second review: deliver it once yourself (`{{operator_attention}}`,
"Deliver and reconcile"), then add `Delivered: <route> <UTC>` to it. If that fails, mark it "not
delivered"; your replies carry it.

## Route discipline

Agent Manager, Program Runtime, Notification Hub, Prompt Manager, Search Hub and Research Desk are
optional. When a call to one fails, do not retry, diagnose or repair it this unit: take the direct
path (native sub-agents; the underlying commands; skill files by path; `rg`; a needs-you item in
`DECISIONS.md`) and append to `LEDGER.md`
`Route failure · <UTC> · <route> · <command> · <first error line> · direct path taken`.
Retry at most once per review. A failure blocking a proof at three reviews with no direct path
becomes an owner's task or a bug report, never your own repair.

## Shared machine

Other commissions share this host: never message, resume or stop their agents.
Before a download, model install or render, measure free space on the real output filesystem and
budget peak growth; admit through the host admission owner. A job leaves 120 GB free after its
peak; below 100 GB pause heavy work and write one needs-you item. Each owner keeps growth under
25 GB unless `STATE.md` agrees more, records job, paths, peak and release in its checkpoint, and
deletes intermediates, never evidence.

## Cards

Owners get their card's sections whole. An owner names what it starts in its first hour and starts
before review. Results stay in the card's run folder, listed in `STATE.md` by absolute path. Pass an output to the card whose mission names it
(`Hand-off`). A workstream's `Brief:`
is where its owner starts.

{{cards}}
## Scheduler procedure

This session holds the goal. It is the scheduler and the only writer of this folder; it does no
domain work and changes no code. `research-coordination` (you) and `adaptive-research` (owners)
win over any other skill's stop or threshold rule.

1. **Start.** Read `STATE.md` (create it if missing). Do not write a goal; if you must, copy
   `KERNEL.txt` verbatim. Record this session's model in `STATE.md` from its launch command,
   never from your self-description:
   `p=$$; until [ ${p:-1} = 1 ] || ps -o args= $p | grep -qE '^\S*codex .*(-m|--model)[ =]'; do p=$(ps -o ppid= $p); done; [ ${p:-1} != 1 ] && ps -o args= $p`
   No output: record `unknown`, no item. A model other than `{{supervisor_model}}` gets one
   needs-you item recommending a restart on it. Start the owners either way, with `model`
   set to `{{owner_model}}`; an effort other than `{{supervisor_effort}}` is recorded, not a stop.
   Record the commission ID, `started_at` and the allowance end ({{allowance_hours}} hours
   later). Take the footprint baseline (step 7). Drain the card topics' inboxes. A mission whose
   ambition review is pending gets one while owners start. Start {{owner_count}} owners ({{owner_split}})
   from the owner brief with `fork_turns=none` and {{owner_model_line}}; they are native
   sub-agents.
2. **The unit loop**, until the allowance ends:
   1. Run `date -u`; read a `MISSION.md` only for the card under review.
   2. Block with `wait_agent`, `timeout_ms` = time left until the next review in `STATE.md`, at
      most 3600000. It returns when an owner messages or finishes; a chat message interrupts it.
   3. On every wake run `date -u`. Act on a plan, a showcase claim, an operator item or need, an
      owner that finished or failed, or a passed review time. Otherwise wait again: write
      nothing, message nobody.
   4. The hourly review covers every card: measure each owner (below) and read its checkpoint;
      act on a tripwire, an owner finished, failed or with a checkpoint over two hours old, an
      unclaimed landed showcase, or a waiting workstream. Run the footprint check and
      `python3 -I {{topic_health}} --topic <topic>`; tell an owner whose tool is down or
      `unknown` its fallback. Keep the topic's `work/README.md` true: states, the next brief
      before an owner needs it, risk dates only when re-reviewed; an occurred risk becomes an
      owner's next task. A line still under `Wanted` in a card's list file: tell that owner.
   5. Overwrite `STATE.md` (with the next review time), append one `LEDGER.md` entry, end your turn.
   An owner returns at a showcase claim or after about six hours: re-task it with
   `followup_task` when `drift` shows no `owner-age` tripwire, else start a fresh owner with the
   line "Read [run folder]/CHECKPOINT.md first." Replace an owner mid-work only when stuck or
   failed, and stop its workers. Never mark the goal `blocked`; mark it `complete` only when a
   challenger returned `NONE` for every card.
3. **Reviews.** One ambition review per workstream plan while the owner builds, one challenger
   per showcase claim, nothing else. Paste the brief from `{{review_briefs}}` verbatim into a fresh
   agent named `<workstream>_review` or `<workstream>_challenge`. A rejection asks
   for the larger first item, not a new plan. Code that moves money or deletes operator data
   gets one independent code review before its claim. Judge a claim by the measured change and
   the running showcase.
4. **Waiting and needs you.** A showcase that needs the operator, elapsed days or capacity you
   cannot free is marked `waiting-operator` or `waiting-time` with its resume date; its request is
   a needs-you item; the owner takes the next workstream. `STATE.md` and every reply to the
   operator open with the open decisions.
5. **Questions and steering.** "How is it going" is answered from `STATE.md` and changes
   nothing. Steering is acted on that turn: new items go to the card's list file under `Wanted`
   in the operator's words, other instructions to the mission's Operator intent; tell the owner.
6. **Pause and resume.** Asked to pause, pause the goal and say only the operator can re-activate
   it. Told to resume while it reads `paused`, your first line is: "Type `/goal resume`: I cannot
   re-activate the goal, and without it nothing wakes me between your messages." Until it reads
   active, hold step 2 inside your turn.
7. **Footprint.** `F` is `{{footprint}}`; `B` is `BASELINE.json` and `K` `FOOTPRINT-KEEP.txt` in
   this folder (a missing `K` is empty). Baseline once before the first owner: `mkdir -p` the run
   folders, then `python3 -I F snapshot --out B {{watch_args}}`; never retake an existing `B`.
   Check: `python3 -I F diff --baseline B --keep-file K` (add `--disk` at reviews) lists what runs
   now that did not at the baseline and exits 1 while a line remains. Match each line to an
   owner's `Started` list: tell that owner to stop it when idle, or stop it yourself if the owner
   ended. An unclaimed line, or one meant to stay, goes in `K` with a comment and under
   `Left running` in `STATE.md`. `NOT CHECKED` is not clean: read that source yourself. On refused
   admission: run the check, find the holder, wait through its owner or use a fleet node, and
   staff other work meanwhile.
8. **At the allowance end** or after `wind-down`: stop admitting work, have each owner checkpoint
   and stop what it started, settle the footprint, and write the final `STATE.md`: per card,
   workstreams landed, dimensions moved, lines added and removed, what waits, what is unverified.
   Update `MISSION.md` status cells and `work/` states, pause the goal (`update_goal` status
   `paused`) and say the window ended.

**Slots.** At most eight threads; the scheduler and owners hold {{held_slots}}, leaving {{slots}}
for workers and reviewers. An owner with no free slot works in its own thread.

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
as the `/goal`. Steer, answer and wind down through `OPERATOR.md` (by hand or `research-desk say`),
not chat, which compaction keeps replaying. Progress: `STATE.md`.
