---
name: "content-desk"
description: "Use Content Desk as the durable editorial ledger for evidence-backed content workflows."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  status: "active"
  modes: []
  tags: [content, editorial, publishing]
  requires:
    scenarios: []
    commands: []
  origin:
    kind: "authored"
---

## Content Desk

Use Content Desk as the durable editorial ledger. It stores drafts, cited
claims, reviews, campaign capacity, and publish records; it does not generate
copy, hold credentials, or publish to social platforms.

### Required reading

Before creating a draft, read:

- `docs/marketing/catalogs/post-types/README.md` and the selected post-type
  canon — a post type must be active before approval.
- `docs/marketing/strategy/CHANNELS.md` — channel and disclosure constraints.
- `scenarios/content-desk/docs/reference/configuration.md` — claim evidence
  and campaign-capacity calibration.

### Agent workflow

1. Create an evidence-backed campaign and activate it.
2. Create a draft against an active campaign slot and post type.
3. Create shared claims with evidence; use a re-runnable check for
   quantitative, existence, and status claims.
4. Cite each claim against the exact span of the current draft body.
5. Request/record the appropriate review verdicts.
6. Inspect the draft's cited claims and blockers. An agent must never approve
   or publish a draft; an operator performs approval and the scheduler owns
   external publishing.

### Single read address

Read the board before deciding what to do. `content-desk.board-read` is the
single marketing read address for a person or a fresh agent: one call returns
current campaign and draft work, the deterministic next action per draft,
capability readiness gaps, the offer release readiness for the scenarios the
active campaigns target, and what changed against an optional baseline.

```bash
program-runtime library run content-desk.board-read
program-runtime library run content-desk.board-read --input 'baseline=<prior snapshot>'
```

Store the returned `campaign_snapshot` and `draft_snapshot` as the next run's
`baseline`. An absent baseline reads `changes_status=no_baseline`; it is never
an unchanged board. `content-desk.capabilities-read` stays the focused read when
only the capability catalog is needed.

Draft status determines the next action: `requested` → begin drafting;
`drafting` → complete drafting; `drafted` → start checking and review;
`checking` → complete the review run; `blocked` → resolve the blocking gate;
`reviewed` → operator approval required; `approved` → submit for release.
`published` and `abandoned` are terminal. Any other status is reported as drift,
not given an action.

The same read surfaces offer release readiness: `offer_readiness.launch_targets`
matches each active campaign's `scenario_names` to an Offer Desk release-ladder
deliverable and reports its rank, status, blocking enablers and the next
readiness action. Readiness goal fields are copied from Offer Desk; when that
owner reports no goal state the board marks it unknown (`readiness_goal_reported=false`),
never false. `offer_read_status` is `read` or `unavailable`; an unavailable
ladder degrades the whole board to `partial` and is never read as an empty or
ready target.

### CLI surface

Use `content-desk <group> help` for current flags. The normal sequence is:

```bash
content-desk campaigns create --name <name> --evidence-ref <ref> --slot <channel:format:capacity>
content-desk campaigns activate <campaign-id>
content-desk artifacts create --campaign <campaign-id> --post-type <post-type-id> --channel <channel> --format <format>
content-desk claims create --statement <statement> --kind <kind> --evidence-kind <citation|check>
content-desk claims cite --draft <draft-id> --claim <claim-id> --span-start <n> --span-end <n> --body <current-body>
content-desk claims list-draft <draft-id>
```

If a draft is blocked, report the exact failed gate and repair the evidence,
post-type activation, or review outcome. Do not bypass a gate or substitute a
claim from a different draft.
