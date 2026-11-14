# Why supervised research works

Supervised research is the operator's most productive way of working. This page says why,
where it breaks, and the rules that keep a change from weakening it. It describes the method;
the desk's design is in [`SYSTEM.md`](SYSTEM.md). Evidence: the research home and the Codex
measurements of 2026-11-11 and 2026-11-12
([measurements](../internal/evidence/codex-measurements.md)).

## The method in five lines

The operator picks topics and a time allowance. One supervisor session (the scheduler) runs
for that window. It starts one owner per topic and keeps them supplied; it does none of the
research. Each owner works from its topic's `MISSION.md` for hours, with its own workers.
Everything worth keeping is a file in `~/.vrooli/research`.

## Why it works

1. **The push survives compaction.** A long session compacts many times, and each compaction
   keeps only a goal objective of 600 characters or fewer and a limited number of the
   operator's typed messages. So the start is three things: one stance message in the
   operator's own words (3,500 characters or fewer), the reply "ready", and a short goal that
   points at a charter file. Nothing else is typed. Hours later the session still holds the
   operator's push and the path to its instructions. An ordinary session loses both.
2. **The conversation is disposable.** State is in files: the charter, `STATE.md`,
   `LEDGER.md`, each owner's checkpoint. A compacted, restarted or replaced agent re-reads
   them and continues. Nothing depends on what an agent remembers.
3. **One small entry point.** Each role is told exactly what to read to start, and that
   reading has a ceiling (`limits.json`). An agent with a small, clear start acts; an agent
   with a large or vague one reads until its context is spent.
4. **The supervisor schedules and does not work.** Its context stays free for judgment:
   which owner is stalled, what the operator must decide, what to staff next. Owners keep
   one outcome for hours, so no work is cut to fit a short-lived worker.
5. **Missions are programmes.** A mission names a large destination and its showcases, and
   an owner is judged against that, so work does not shrink into small safe tasks.
6. **Time ends the work, not a checklist.** The goal wakes the supervisor again after every
   turn until the allowance ends. A finished item leads to the next one.
7. **Every service is optional.** Agent Manager, Program Runtime, Notification Hub, Prompt
   Manager and Search Hub each have a direct path (a file, a script). A failing service is
   recorded in one line and retried at most once an hour. It is never repaired inside the
   window.
8. **The operator is asked little.** Only real decisions reach the operator, each with a
   recommendation, and work that does not depend on the answer continues.

## Where it breaks, and the guard

| Failure seen | Guard |
|---|---|
| Starting took an agent conversation and about five minutes, and the setup varied | One packet builder (`packet.py`) and one launch action |
| An earlier commission on the same topics was still running | The launch check names it and the launch supersedes or runs alongside it |
| Chat steering was replayed after every compaction and pushed the stance out | Direction goes in the commission's `OPERATOR.md` |
| A decision never reached the operator (2026-11-11) | The supervisor only writes the item; the desk's courier delivers it and writes the answer back |
| A supervisor spent its window repairing a platform fault | Route discipline in the charter |
| Missions and charters grew until owners read too much | Read sets are measured and shown; a mission over 40,000 bytes is flagged |
| Nobody could say what was running | State is derived from files and session records |
| Results were not digested | The `research-review` skill and recorded verdicts |

Still open: no reading of remaining quota before a launch; owners run on the supervisor's
model; a supervisor never marks its goal complete, so "ended" is inferred.

## Rules for a change

- No step of a supervisor's or owner's procedure may need a running service.
- Never raise a reading ceiling. To add a line to a template, cut one.
- Never type into a supervisor's conversation after the goal is set.
- One writer per file.
- After a Codex or Web Console upgrade, run `research-desk launch-test` again: the
  compaction and goal behaviour above is measured, not promised.
