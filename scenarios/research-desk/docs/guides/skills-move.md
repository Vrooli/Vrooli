# Skills Move — Executable Plan

For the skills owner. It moves the research skills into this scenario (G9), adds
`research-review` (G7), `packet.py` and the templates (contract §2 of
[`../reference/FILE-CONTRACT.md`](../reference/FILE-CONTRACT.md)), route discipline (G8),
and the no-growth test (B5). Never run git; other agents share the tree.

## 0. Record the ceilings first

Before changing any text, measure in bytes: each moved `SKILL.md`, each reference file a
role is told to read, and the three latest charters. Write them to
`skills/research-coordination/templates/limits.json`:

`ceilings.skill_bytes` keys are file paths relative to `scenarios/research-desk/skills/`.
`ceilings.{scheduler_start,scheduler_unit,owner_start}_bytes` were measured on
`programme-three-20261111` before any text changed; `read_sets` names each role's
components and `read_sets.measured_before` holds the three latest packets' values.
Ceilings may go down, never up, without an operator decision. `test_packet.py` fails when a
skill file exceeds its ceiling; the Go test in `api/internal/skillscripts` owns the role
totals.

## 1. Move each skill (one at a time)

For `research-coordination`, `adaptive-research`, `local-experimentation`:

1. `mv scenarios/prompt-manager/store/skills/packs/core/<id> scenarios/research-desk/skills/<id>`
   (one filesystem, atomic). Remove `__pycache__/` inside it.
2. Fold `skill.json` into the `SKILL.md` frontmatter: `name` (= `<id>`), `description`,
   `license`, and `metadata` with `kind`, `schemaVersion`, `modes`, `tags`, `icon`,
   `status`, `revision` (+1), `requires`, `origin`. Delete `skill.json` in the new folder.
   Follow `scenarios/device-control/skills/device-control/SKILL.md`.
3. `ln -s ../../../../../research-desk/skills/<id> scenarios/prompt-manager/store/skills/packs/core/<id>`.
   The pack loader skips a pack directory without `skill.json`
   (`prompt-manager/api/internal/store/skill_store.go`, pack listing loop), so the link
   adds no second id; it keeps old paths in launched charters working.
4. Verify at once: `prompt-manager skill show <id>` shows the scenario source;
   `prompt-manager skill list --json` contains `<id>` exactly once;
   `prompt-manager skill read <id>` returns the new text; `cat` of the old path works.
   If the registry reports a duplicate or goes offline, delete the link and stop.

Then `prompt-manager skill sync` and `prompt-manager skill topology` (the research skills
are not natively projected; expect no drift). Run `prompt-manager skill-set validate
research-desk`; every directory under `skills/` registers whether or not
`.vrooli/service.json` `skills` lists it.

## 2. Add the new material

- `research-coordination/templates/{charter.md,stance.txt,kernel.txt,limits.json}`: constant
  text from `cohorts/programme-three-20261111/README.md` (procedure base),
  `programme-four-20261110` and `standing-teams-w1-20261110` (stance and kernel skeletons),
  with the Steering, Decisions and route-discipline text the contract requires.
- `research-coordination/scripts/packet.py` and `test_packet.py` (contract §2).
  `mission_check.py --json` gains each topic's workstream rows (`id`, `destination`,
  `magnitude`, `showcase`, `status`); `packet.py` imports that parser.
- `research-coordination` SKILL.md: "Prepare a commission by hand" (run `packet.py`, or fill
  the templates and check the gates) replacing packet-shape prose; the needs-you grammar
  (contract §5) replacing the `## Actionable now` paragraph in
  `references/operator-attention.md`.
- `research-review/SKILL.md`: inputs (topics; evidence by hand: commission folders naming
  the topic, owner checkpoints, ledgers, `mission_check.py --json`); write
  `reviews/<id>/REVIEW.md` (per topic: landed with path and date, claimed but unverified,
  open, grew); tidy proposals inside the reviewed topics (move to `archive/`, never delete,
  nothing under a live commission's run folder); verdicts verbatim in `VERDICTS.md`
  (contract §4); full replacement missions in `proposals/<topic>.md`, applied to
  `MISSION.md` only after the operator confirms; suggested emphasis for the next packet.
- Owner form of route discipline in `adaptive-research`, offset by removed text.
- `skills/research-desk/SKILL.md` becomes a short pointer to the CLI's read verbs.

Route discipline text (charter template and, as an owner form, `adaptive-research`):

> Agent Manager, Program Runtime, Notification Hub, Prompt Manager and Search Hub are
> optional. When a call to one fails, do not retry, diagnose or repair it in this window;
> take the direct path (skill file by path; a needs-you item in `STATE.md`; the underlying
> commands; `rg`; native sub-agents) and append to `LEDGER.md`:
> `Route failure · <UTC> · <route> · <command> · <first error line> · direct path taken`.
> Try that route again at most once per hourly review.

## 3. Decision delivery in one place (A5)

Remove from `research-coordination` and `references/operator-attention.md` the consent
request, Hub reconciliation and Agent Manager decision-raise steps, and change
`docs/agent-system/LONG_EFFORT_DESIGN.md` D17 to "write the item; the desk delivers it"
(ruling A5 assigns this edit to the skills owner). Until both land, desk-built charters
carry the override line.

## 4. Program Runtime programs (C8)

| Program | Verdict | Action |
|---|---|---|
| `research-attention-delivery` | retire | Agent Manager effort decisions; replaced by the desk courier |
| `research-supervision-read` | retire | replaced by desk activity and direct reads; its `effort_ref` route is the one the rulings avoid |
| `research-attention-index` | retire | second parser of the needs-you grammar |
| `research-workspace-review` | re-own | `owner_skill` → `research-review`; optional evidence input for reviews |

Remove every reference to the three retired programs from the moved skills now; no skill
step may require any program. The program files, `owner_skill` edits and
`program-runtime/docs/guides/supervision-loops.md` belong to Program Runtime: report them to
the lead for that owner.

## 5. Repair links

Maintained files only: `docs/internal/research-experimentation/{README,LEDGER}.md`;
research home `README.md`, `ATTENTION.md`,
`topics/{ai-video-production-and-funnels,niche-book-production,experiment-operations}/MISSION.md`,
`topics/experiment-operations/README.md`; prepared, unlaunched packets (today
`cohorts/local-models-w1-20261111/README.md`). Do not edit launched charters, ended folders,
`ATTENTION-HISTORY.md` or evidence files.

## 6. No-growth test (B5)

A Go unit test in `api/internal/skillscripts` (`ceilings_test.go`) reads `limits.json`, computes the bytes each role is
told to read (rendered charter for a fixed spec, `STATE.md` copy, the skill files the
procedure names) and fails above a ceiling. It runs in `vrooli scenario test research-desk
--phases unit`.

## 7. Removing the links

When `research-desk commissions` shows no `live`, `paused` or `limited` commission whose
charter names an old pack path, delete the three links.
