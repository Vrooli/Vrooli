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
