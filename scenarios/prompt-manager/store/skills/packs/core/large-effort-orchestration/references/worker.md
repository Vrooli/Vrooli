# Epoch worker card

You deliver one epoch of an orchestrated goal. Read this card and your epoch file
(`<goal home>/epochs/E<n>.md`); read other protocol docs once at epoch start, never
on resume. Your prompt names the epoch file, the orchestrator run ID and your
resume set.

**Resume set (about 20 KB).** The brief (header fields, modules and budgets,
deletion list, exit gate, non-goals), the last ~20 slice-log lines, open
directives, and the scenario docs the brief names.

**Loop.**
1. Make one change and run its focused checks. A change whose checks pass is a
   work unit. Size units as meaningful steps (a module, an owner or a whole
   pattern across a package), not single edits; the overrun trigger counts them.
2. Append one line to `## Slice log`:
   `<ISO time> | <what changed> | exit metric=<value> | net runtime lines=<±n> | net test lines=<±n> | ack=<directive IDs or ->`
   Net lines are the unit's delta from before/after inventory snapshots of the
   epoch scope. Never use git. Moved or extracted code is not removal.
3. Read `## Directives`. Acknowledge each new ID in your next line and act on it.
4. Run `agent-manager effort epoch-check <epoch file> --runs <your run ID> --wake-key <orchestrator run ID>`.

**Growth budget.** `refactor` epochs close at cumulative net runtime lines ≤ 0.
`feature` epochs stay within 1.5× `Expected size`. Tests follow
`path:docs/testing/UNIT-TEST-AUTHORING.md#mature-a-suite-instead-of-growing-it`.

**Step-back.** `epoch-check` exits 3 with `STEP_BACK <reason>` when a trigger fires:

| Trigger | Signal | Default |
|---|---|---|
| Flatline | exit metric unchanged across the last K lines | K = 10 |
| Overrun | log lines above the estimate | 2× |
| Growth | refactor: cumulative net runtime lines; feature: above expected size | > +300; > 1.5× |
| Spend | weighted non-cache tokens of the epoch's workers | set after 3 epochs |
| Wall clock | now minus `Started` | 24 h; prints `REVIEW`, never forces a stop |

Then stop adding more of the same. Write about 10 lines under the slice log: why
the epoch is stuck, what change elsewhere would make it tractable, and whether to
**park**, **switch** to an enabling slice, or **shrink** the scope. Continue only
inside a shrunk scope; the orchestrator records the choice in `QUEUE.md`.

**Broken infrastructure** (sandbox, shadow, Test Genie, Program Runtime, bridge)
never blocks you. Append a `WORKAROUNDS.md` entry (what failed, the fallback, when
to retry), continue unsandboxed in the shared tree or restart in a quiet window,
and stop retrying that tool.

**Blocked** means a decision, credential or approval you lack. Name it in a slice-log
line. Friction you can diagnose is not blocked.

**Never:** write `ACCEPTED` or directives, edit the brief's gates, read
`FEEDBACK.md`, install a tool outside the project (request it in a slice-log line),
or stop at a compaction, a green test or a status request.

**Exit.** When every exit-gate command passes, log a final line with their results,
mark your native goal complete, and end with changed, verified, remaining and
unverified. The orchestrator decides acceptance.
