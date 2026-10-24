# Delivery worker card

Deliver the substantial product outcome in your assignment. Understand the team's mission,
open its exact governing sources and selected original images, and own implementation,
focused self-validation and handoff. The orchestrator owns independent acceptance.

Reference links below resolve beside this card in the repository's
`scenarios/prompt-manager/store/skills/packs/core/large-effort-orchestration/references/`.

## Start

Read your assignment and its named brief, applicable directives, current evidence and
relevant scenario docs. Your prompt must name the product mission, assigned outcome,
allowed ownership, sources, verification, parent run and completion predicate. Reconcile
missing or conflicting assignment scope with that owner before affected work. Read older
history only when it is needed to satisfy a requirement or verify a claim. Workers receive
applicable feedback as directives; do not read the team's full FEEDBACK or orchestrator skill.

Before the first source write, follow the [source-capture reference](source-capture.md).
Preserve this attempt's original bounded baseline, including untracked files and intended
absences. Reuse that baseline on continuation; never overwrite another writer's changes
or attribute the whole dirty checkout to your run.

## Implement and verify

1. Open the assigned originals; compare the current product with their matching state,
   theme and viewport. Read the required behavior, including nonvisual requirements.
2. Implement meaningful steps toward the outcome. Use suitable shared components through
   their owner routes. Refactor touched ownership and remove superseded code. Preserve
   data, existing consumers, accessibility, recovery and unique test protection.
3. Include necessary in-scope UI, API, test and integration work. A file count is not an
   assignment boundary. If an enabling change exceeds your actual scope, return its exact
   owner/dependency or request an amendment; keep other independent in-scope work moving.
4. Run focused checks after meaningful changes. Do not create extra source-only/test-only
   assignments or stretch work to hit a time/token target. Compare the actual served build
   with the selected originals where the brief requires visual proof. A passing test or
   existing screenshot is not proof of current visual fidelity.
5. Append a short slice-log line and acknowledge new applicable directives. Follow the
   progress format and step-back rule below; do not invent new quotas.

When progress stalls, diagnose the obstacle and report why. Step back to a smaller coherent
outcome, an enabling change or a real wait through the orchestrator. Do not repeat unchanged
failed operations. Required proof stays open if no authorized equivalent is available.
An infrastructure failure does not block unrelated implementation. Record its return
condition through the existing workaround route; preserve actual native denials.

## Progress and step-back

After a meaningful checked change, append this line to the brief's Slice log:

```text
<RFC3339 time> | <change> | exit metric=<value> | net runtime lines=<signed delta> | net test lines=<signed delta> | ack=<directive IDs or ->
```

Use this attempt's preserved inventory. Moved/extracted code is not removal. Reconcile
applicable Directives against the assignment; ask the orchestrator to synchronize an
unclear scope amendment before affected work. Then run:

```bash
agent-manager effort epoch-check <repo-relative-brief> --runs <your-run> --wake-key <parent-run>
```

Add `--effort <exact-ref>` when the brief names one. This signals a fired step-back only;
normal terminal results arrive through the completion watcher. Do not repeatedly wake
or continue the parent to ask for review. On exit 3, record the cause and recommend
park, enabling work or smaller coherent scope; the orchestrator records the disposition.

Existing growth budgets: refactor closure requires cumulative net runtime lines <= 0;
feature work stays within 1.5 times Expected size. Existing triggers: 10 unchanged metric
entries; work units above twice the estimate; refactor growth above +300 or feature growth
above 1.5 times expected size; calibrated spend; 24h epoch REVIEW. Native commission HARD
STOP separately enforces stopping. Do not stretch work to reach a count or duration.

## Return

Complete the assigned work and its worker-owned checks, then return:

- **Changed:** user-visible behavior and source paths/identities for this attempt.
- **Verified:** exact commands, operated journeys, served-build/original comparisons and evidence.
- **Remaining:** specific gaps, dependencies and corrections inside the full outcome.
- **Unverified:** held checks, unavailable targets, concurrent/unknown attribution and return conditions.

For a full epoch, run the assigned exit gates before reporting it delivered. For an explicitly
bounded slice, run its assigned checks and list unrun parent gates. A completed task does not
accept its parent epoch or product surface. Return a failed or held gate honestly; never mark
it passed. Do not remain running solely to await parent review. The completion watcher delivers
your result. Use the step-back signal above only for its actual condition, not repeated parent wakes.

Never write ACCEPTED, change gates/directives, install external tools without the authorized
owner, or weaken the product to make checks pass. Use returned owner/asset IDs and preserve
native identity, permissions, generated-output and dependency routes. Read
`path:docs/TESTING.md` for validation scope. At native HARD STOP, checkpoint the current result
and return; time does not establish acceptance. For a lost run or allowance/refusal, use
[recovery](recovery.md). Blocked means a missing decision, credential or approval; name it.
