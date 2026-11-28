# research-desk proto

Wire contracts for `scenarios/research-desk`. One service per domain
(`scenarios/research-desk/docs/concepts/DOMAINS.md`):

- `v1/activity/activity.proto` — `ActivityService`: what is moving, commissions, topics and
  changes since a look. Reads only.
- `v1/launch/launch.proto` — `LaunchService`: build a commission packet, launch it, control
  its supervisor session, and the launch self-test.
- `v1/operator/operator.proto` — `OperatorService`: needs-you items and answers, directions,
  steering, review verdicts, the courier switch and self-test, and marking a look seen.
- `v1/shared/` — `Evidence` (every derived fact carries its source or why it is unknown),
  `Entry` (one `OPERATOR.md` entry), `health.proto` (the `/health` response) and
  `errors.proto`.

Which procedures act for the operator, and the files each one writes, are in
`scenarios/research-desk/docs/concepts/SYSTEM.md` and
`scenarios/research-desk/docs/reference/FILE-CONTRACT.md`.

Regenerate after any change: `cd packages/proto && make generate SCENARIO=research-desk`,
then `cd scenarios/research-desk && make endpoints`.
