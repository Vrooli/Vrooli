# Decision-quality eval (Codex Luna)

Most Notification Hub decisions are written by Codex Luna agents. This eval checks
that a Luna agent that gets only the facts and the notification-hub skill
produces a decision the operator finds professional, informative and readable on a
phone in light and dark.

## Run it

From an operator terminal:

```bash
python3 scenarios/notification-hub/evals/decision-quality/run.py all --round after --local-owner
```

This one command does all of the following:

1. Launches one Codex Luna run per case through Agent Manager, on the
   `meta-optimization-manager/trials` profile (role `code.cheap`, which is GPT-6 Luna
   medium; `--profile` picks another, such as `notification-hub/decision-eval` once
   reconciled).
   Runs work in place with no sandbox (`--sandbox-config` mode off, localhost network),
   because the case folders are outside the repository; the trials profile alone would
   resolve to the protected sandbox, which refuses in-place runs.
2. Waits for each run once.
3. Records the model from the run record.
4. Re-runs the Hub's dry-run check on each run's final attempt.
5. Photographs the page at 390 px in light and dark, using the Hub's own
   frame code (`document.ts` and `theme.ts`, bundled with esbuild).
6. Audits contrast in the browser.
7. Writes `SUMMARY.md`.

Other commands:

- `--round after-2` (any `after-<n>`) runs a later iteration of the live skill into its own folder.
- `--round baseline` runs against the skill as it was before the decision-quality work. That snapshot is kept under the evidence folder in `skill-baseline/`.
- `--cases rebaseline,policy-batch` runs only the cases you list.
- `launch` and `collect` are the two halves of `all`.
- `audit` checks each run's own transcript (Agent Manager run events): every write landed in its case
  folder and nothing was sent. `collect` runs it too; the result is `containment.json` per case and
  the last column of `SUMMARY.md`.
- `shoot page.html --out DIR` photographs any page.

An agent session can launch runs only if it holds a run-creation grant
(`vrooli agent authorize <session> --policy run-creation --target 'agent-manager:*'`,
approved by the operator). The host caps concurrent runs (10); launch a round's
remaining cases with `launch --cases ...` when slots free.

## What a run gets

`prompt.md` holds the case's facts from `cases/<case>.md`. It also tells the agent:

- read the skill;
- prepare the decision;
- check it with the case folder's `check-decision`.

`check-decision` is `bin/check-decision`. It always adds `--dry-run`, refuses
`--and-wait`, and keeps every attempt: arguments, working folder, the files it
names, and the Hub's output. **Nothing reaches the operator.** Subject keys start
with `eval-<case>-`.

## Cases

Each case comes from real Hub asks, and sizes and dates are real.

| Case | From | Tests |
|---|---|---|
| `backup-secrets` | 45 sensitive backup items (asked 3 times) | List/batch shown as groups and a proportion, a `--form` multi-select, a standing rule that stops the repeats |
| `rebaseline` | Root rebaseline (asked twice) | The command on the first screen with where, what and undo; naming the cause |
| `policy-batch` | Eight standing-team policies | Many items in one decision |
| `codex-lower-tier` | 14 "keep the lower tier on?" asks | Recurrence and a standing rule |
| `image-tools-nav` | Phone navigation choice, with real screenshots | A UX choice shown in the real app |
| `bas-purge` | A simple yes/no | A short decision that still adds something |

## Evidence

Evidence goes under `~/.vrooli/research/notification-hub-decision-eval/<round>/<case>/`:

- `prompt.md`, `run.json`, `run-record.json`, `run-facts.json`
- `attempts/NN/`
- `dryrun.json`, `cover.json`, `page.html`
- `shots/first-{light,dark}.png` (the first screen) and `shots/full-{light,dark}.png`
- `shots/shots.json` (contrast audit)
- `result.json`

Look at every screenshot. The checks catch shape and readability, but whether
the page is beautiful is still a human judgement. Results and judgements for
each round are recorded in `RESULTS.md`.

## Results

The 2026-11-15 rounds (baseline, after, after-2; 18 Codex Luna runs) and what changed
between them: `~/.vrooli/research/notification-hub-decision-eval/RESULTS.md`.
