# Ecosystem Hunt goal

A hunt runs for hours, so its goal follows `harness-goal-authoring` §2a: a stance message sent
first, a short native kernel, and a charter that holds everything else and is re-read every round.
Replace each `<...>`. Give every new hunt a new Hunt ID unless it continues an earlier hunt; that
is what lets a new hunt revisit ground an old one covered.

The Stance text is the operator's own words. Keep it verbatim and in full in the stance message,
the charter and every hunter or segment brief, and keep a verbatim excerpt (its first two
sentences) in the kernel; paraphrase is where the push gets lost. Check each filled text with
`wc -m`: the stance message at 3,500 characters or fewer, the kernel at 600 or fewer.

## Choose the shape

| Hunt | Shape |
|---|---|
| Up to about 6 hours or 25 rounds | **Single session**: the kernel below; the session hunts each round itself and may use hunters. |
| Longer, or more than one territory at a time | **Scheduler**: a root that does no hunting, over segment owners of about 10 rounds each (below). |

## Charter

Write the charter to `<charter path>` (for example the hunt's scratch folder under
`~/.vrooli/research/` or the operator's chosen path) before starting the goal:

```text
Ecosystem hunt <Hunt ID> charter.
Method: prompt-manager skill read ecosystem-hunt (§1 Stance wins over any other skill's stop or threshold rule).
Destination: <outcome, e.g. "every open tier 1–4 class in the Agent Manager friction report is fixed at its mechanism, and two multi-owner consolidations have landed">.
Stance: <the operator's words for this hunt, verbatim; when they gave none, the operator paragraph that opens ecosystem-hunt §1, verbatim>
Focus: start with <territories or concerns>; every third round goes to the next focus area or to a large item, whatever the earlier focus still holds.
Rounds: each round follows ecosystem-hunt §8 and ends with its §10 record journaled to vrooli-memory scope ecosystem-hunt with the validation output for every changed scenario. A round with no change does not count toward the destination. After two no-change rounds in a row, the next round takes a large item (§5) or the next focus area. Every tenth round is a synthesis round (§8b).
Ledger: <ledger path>, one line per round or segment: round range, record IDs, classes fixed, owners and paths deleted, files, lines added and removed, no-change rounds and the reviewer's answer. The scheduler writes it; it replaces nothing in §10.
Closure: before declaring the destination met or "nothing material left", run the challenger brief in docs/agent-system/LONG_EFFORT_DESIGN.md §2a against this charter; continue with its top route unless it returns NONE.
Exemplars: <round records that show the expected unit; hunt-2026-11-05 round 16 (credential migration with retry, deletion and fail-closed tests) and round 30 (Test Genie abort silent-success fix) are good defaults>. Do not use the latest records as the template.
Authority: code, test, doc, skill, prompt, template, generator, config and shared-package changes anywhere in the repository, including deletions and migrations, within ecosystem-hunt §6 hard limits. Out-of-repository and host state go through their owners.
Blocked means only an operator-owned decision, credential or external action (§7). Prepare it with a recommendation, route it, and keep hunting.
Keep going through friction; when progress stalls, step back — write why, then switch territory or lens — instead of stopping.
Budget: <hours or rounds>. When reached, write the final report: changed, verified, remaining, unverified, and the next three territories.
Non-goals: shims, fallbacks, second owners, weakened or suppressed checks, line-count floors, hedged closeouts.
```

## Stance message

In a Codex session, send this as its own message before setting the goal, in either shape. Codex
keeps it through compaction; it does not keep a long goal. When the hunt runs as an Agent Manager
run, the stance message opens the task prompt instead, without its last line, and `--until-file`
carries the kernel.

```text
Stance for ecosystem hunt <Hunt ID>, in the operator's words: "<the charter's Stance line, verbatim>"
Destination: <destination in one line, with its scale>.
Charter: <charter path>. Re-read it and ecosystem-hunt §1 at every round start and after every compaction; its exemplars, not your latest rounds, set the size of a round.
The goal follows this message. Reply "ready" and wait for it.
```

Steer a running hunt through the charter, not chat. The hunt's own defense is the charter
re-read; an operator who keeps messaging it resends the stance message after six newer messages.

## Single-session kernel

```text
/goal Work until this verified end state is true: <destination in one line>, proven by journaled ecosystem-hunt records (hunt <Hunt ID>) and their validation output. Stance: "<the first two sentences of the charter's Stance, verbatim>" Method: <charter path> and ecosystem-hunt §1; re-read both at every round start and after any compaction. After each round's record, end your turn.
```

Ending the turn after each round makes Codex re-inject this objective and its completion audit
before the next round. Without it, the final turn of hunt-2026-11-05 ran four and a half hours
through seven compactions with no objective visible, and rounds shrank to about 75 seconds each.

## Scheduler kernel and segment brief

The root does no hunting. It starts one segment owner at a time (or one per slot), each owning
about 10 rounds, and ends its turn after each segment return.

```text
/goal Work until this verified end state is true: <destination in one line>, proven by journaled ecosystem-hunt records (hunt <Hunt ID>). Stance: "<the first two sentences of the charter's Stance, verbatim>" You are the scheduler: follow the procedure in <charter path>; re-read it at every unit start and after compaction. Do no hunting yourself. After each segment return is recorded, end your turn.
```

Add the scheduler procedure to the charter: start one segment owner per free slot, at most
slots − 2 at once, each with `fork_turns=none` and the segment brief below. When the scheduler
runs as an Agent Manager root run, it may start each owner as a goal-bearing child instead
(`run create --parent-run-id <own run> --until-file <segment kernel> --charter <charter path>`;
wait with `run wait --children`). On any Agent Manager refusal or exit 6, it falls back to a
native owner with the same brief (`LONG_EFFORT_DESIGN.md` §3). Have a fresh
reviewer judge each return against the destination and exemplars; append it to the ledger;
run the Closure step before declaring the destination met. On `agent thread limit reached`,
wait for a slot instead of retrying.

Segment brief (the scheduler fills the brackets from the ledger):

```text
You own rounds <a>–<b> of ecosystem hunt <Hunt ID>. Read <charter path> and prompt-manager skill read ecosystem-hunt first, and re-read the charter at every round start and after any compaction. <Stance, verbatim from the charter.> Territories already covered this hunt: <list>. Territory suggestions from the ledger: <list>. You may edit any file within ecosystem-hunt §6; this overrides a narrower file bound in your agent definition. You are expected to spawn hunters and reviewers per ecosystem-hunt §11 for parallel territories and for every §8 step 6 review, within your share of the slots. Do not return before round <b> is journaled or <time box, at least 2 hours> has passed. Return: per round, the record ID, classes fixed, owners and paths deleted, changed files and lines, validation run IDs; and what you would hunt next.
```

## Harness vocabulary

Name roles the harness actually has. In Codex, use an implementation-capable role at high effort
for segment owners and hunters and a read-only role for reviews; spawn with `fork_turns=none` and
the segment or hunter brief above. The scheduler shape is a Codex shape. In Claude Code, run the
single-session kernel: the session hunts and coordinates, hunters use the `implementer` agent (ask
for `opus` for consolidations) and reviews use `reviewer`; those agents cannot spawn further. Do
not use Claude agent names in a Codex goal.
