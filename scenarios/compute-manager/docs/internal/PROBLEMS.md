# Problems — Compute Manager

## What belongs here

- Known defects that are not worth fixing right now, with the reason.
- Technical debt taken deliberately, with what would repay it.
- Work deferred on purpose, with the trigger that would start it.
- Constraints discovered the hard way, so the next reader does not rediscover them.
- Blockers that live in another scenario and gate work here.

## What does NOT belong here

- Generic template or tooling trouble. That belongs in
  [../guides/troubleshooting.md](../guides/troubleshooting.md).
- Feature requests. Those are operational targets in `PRD.md`.
- Failing tests. Fix them rather than describing them.
- Anything already recorded as a decision. That belongs in
  [DECISIONS.md](DECISIONS.md).

## Entry template

Use this shape so entries are scannable. Newest at the top.

```markdown
### YYYY-MM-DD — short title

**Symptom:** What goes wrong, observable from outside the system.

**Root cause:** What actually causes it, or "unknown" if not yet diagnosed.

**Workaround:** What to do today to keep moving.

**Real fix:** What needs to happen for this entry to be deleted.

**Owner:** Who should drive the fix, or "unassigned".

**Refs:** Code paths, related issues, prior commits.
```

## Entries

### 2026-09-03 — The CLI manifest cannot be schema-valid until the first domain lands

**Symptom:** The `contracts` phase fails with
`manifest.parse_error: cli manifest "compute-manager": at least one group is
required`, so the whole suite verdict is FAIL.

**Root cause:** `template-manager detemplate` removed the example domain and set
`cli/manifest.json` `groups` to an empty array, while
`.vrooli/schemas/cli-manifest.schema.json` declares `minItems: 1` on that field.
This scenario is currently the only one in the fleet with zero groups, so the
constraint has never been exercised in this state before.

**Workaround:** None taken deliberately. Declaring the intended groups from
`docs/reference/cli-commands.md` would satisfy the schema and immediately fail
the manifest-to-handler binding test, because no handler exists. Writing a group
purely to turn the gate green is the same class of dishonesty as hand-setting a
requirement to `complete`, and this scenario already corrected one of those.
The failure is left visible.

**Real fix:** Upstream, and filed. Either relax the schema to allow zero groups
and let the binding test carry the real invariant, or have detemplate leave a
declared stub group behind. Locally, the condition clears on its own when the
first command group lands with its handler, which is part of the first vertical
slice.

**Owner:** unassigned, upstream in `template-manager` and the CLI manifest schema

**Refs:** `cli/manifest.json`, `.vrooli/schemas/cli-manifest.schema.json`, scenario-qa `knw-1788462360339936549`


### 2026-09-03 — Nothing is implemented, and the requirements registry says so

**Symptom:** All fifteen requirements are `planned`. Every automated validation
entry is `not_implemented` and carries no `ref`, because naming a test file that
does not exist would be a fabricated claim. Six validation entries do carry a
`ref`, and all six point at documentation rather than code: three manual
business validations on `COMPUTEM-P0-003`, `-004` and `-005` point at runbook
procedures, and the three `COMPUTEM-P2-*` roadmap entries point at the design
sections that define them.

**Root cause:** The scenario was generated and documented in one session, ahead
of any implementation. This is intended, not accidental.

**Workaround:** None needed. The intended test path for each automated layer is
recorded in the validation `notes` field, so the first implementation slice
knows exactly which file to create and which requirement tag to carry.

**Real fix:** Build the first vertical slice. As each test lands, add its `ref`
and let the sync earn the status. Never hand-set a status to `complete`.

**Owner:** unassigned

**Refs:** `requirements/01-must-ship/module.json`, `docs/START-HERE.md`

### 2026-09-03 — The convenience charge helper never refunds on failure

**Symptom:** `ReserveAndCharge` in the business suite commits the full estimated
cost and then returns on a downstream error without any compensating
adjustment. For an hour of compute that fails three minutes in, the customer is
charged the whole hour.

**Root cause:** It is not a reservation despite the name. It creates no
reservation row, holds no identifier, takes no idempotency key and therefore has
no release path to call. There is nothing to compensate against.

**Workaround:** Do not call it. Use `ReserveCredits`, then
`FinalizeReservation` or `ReleaseReservation`, which is the path the streaming
inference caller already uses correctly.

**Real fix:** Upstream. Either give the helper a reservation and a release path
or remove it. This scenario is not blocked as long as it avoids the helper, so
this is recorded as a hazard rather than a blocker.

**Owner:** unassigned, upstream in `landing-page-business-suite`

**Refs:** `scenarios/landing-page-business-suite/api/internal/commerce/reservation_service.go`

### 2026-09-03 — Requirement sync promotes unattested manual validations and deletes `delivery_scope`

**Symptom:** After a comprehensive suite run, the three `COMPUTEM-P2-*` roadmap
requirements come back as `status: complete` with their manual validations
marked `implemented`, and the `delivery_scope: roadmap` field is gone. Validate
then reports `business_unproven_claim` for each row it just promoted, so sync
and validate disagree about the same three records.

**Root cause:** A manual validation has no artifact the sync writer can check,
so it promotes on the strength of the run alone. The second half is worse: the
promotion strips `delivery_scope`, which is the field that marks a requirement
deliberately deferred, so the marker that should prevent promotion is destroyed
by the act of promoting.

**Workaround:** Reset the three requirements to `planned` with
`delivery_scope: roadmap` and their validations to `planned` after any suite
run, and re-run `vrooli scenario requirements validate compute-manager --json`
to confirm the claim findings are gone. This was done twice on 2026-09-03. The
sync also re-checks the three `OT-P2-*` boxes in `PRD.md`, so those need the
same reset. There is no author-side way to prevent any of it.

One consequence has no workaround at all. A `business_evidence_stale` warning
clears only by running the suite, and running the suite is what corrupts the
statuses, so the contract can be honest or it can be fresh but not both until
the upstream fix lands. Honest was chosen. The standing warning is expected and
should not be treated as drift.

**Real fix:** Upstream, and filed. A manual validation should promote only from
an attestation, and `delivery_scope` should survive a sync unchanged.

**Owner:** unassigned, upstream in the requirements-sync writer

**Refs:** `requirements/03-future/module.json`, scenario-qa `knw-1788462423812925000`, run `20260903-190022-c2883150`

### 2026-09-03 — Bridge publishes no onboarding public key, which blocks unattended enrollment

**Symptom:** `OT-P0-005` cannot be completed. There is no way for this scenario
to obtain the key it must embed in an instance's first-boot configuration.

**Root cause:** `vrooli-bridge` can read its own onboarding public key
internally, but exposes it on no endpoint. The capability exists; the wire
contract does not.

**Workaround:** None acceptable. The tempting substitute is to pass an owner
password through this scenario to bridge's first touch, which would put a
credential on a wire that currently carries none. Do not do this.

**Real fix:** Bridge adds one endpoint returning its onboarding public key. This
is the single new wire contract the whole compute effort needs, and it is small.

**Owner:** unassigned, upstream in `vrooli-bridge`

**Refs:** `docs/concepts/INTEGRATIONS.md`, `DECISIONS.md` 2026-09-03 no-SSH row

### 2026-09-03 — The upstream reservation window is shorter than an hour of compute

**Symptom:** A credit reservation expires after ten minutes. An instance lives
for hours. A naive reserve-then-settle would lose the hold long before teardown.

**Root cause:** The reservation expiry is hard-coded upstream in the business
suite, sized for a streaming inference call rather than a long-lived resource.

**Workaround:** Re-reserve on a heartbeat before the window closes, and record
each renewal as a row rather than mutating the previous one so the history
survives.

**Real fix:** Parameterise the window upstream so a long-lived reservation can
declare its own duration.

**Owner:** unassigned, upstream in `landing-page-business-suite`

**Refs:** `docs/concepts/DATA.md` reservations table

### 2026-09-03 — Upstream refunds silently do nothing for app-scoped charges

**Symptom:** A usage adjustment intended to refund an app-scoped charge succeeds
and changes nothing.

**Root cause:** The upstream adjustment query filters to rows with no
application key, while the charge path writes rows that have one.

**Workaround:** Avoid relying on refunds. Settle the real measured quantity at
teardown rather than over-reserving and correcting downwards.

**Real fix:** Fix the filter upstream. Until then, any refund path built here
would appear to work and would not.

**Owner:** unassigned, upstream in `landing-page-business-suite`

**Refs:** `docs/business/MONETIZATION.md`

### 2026-09-03 — compute_minutes is an active offer stream with no declared meter

**Symptom:** The offer catalog reports an undeclared stream and a deliverable
meter gap for compute minutes.

**Root cause:** A stream node for compute minutes already exists in the
catalog at status `idea`, recorded as intent before any scenario declared the
matching meter key.

**Workaround:** None needed while nothing is sold.

**Real fix:** Two halves, and only one of them is a file in this repository.
Declaring `compute_minutes` in this scenario's monetization manifest with its
enforcement paths, regenerating the meter inventory and seeding the per-tier
limit rows clears the undeclared-stream half. The deliverable-meter-gap half is
attributed to `vrooli-bridge`, because the gap check only follows `unlocks`
edges and the only such edge into this stream comes from bridge. Closing it
needs a compute-manager deliverable node and a rewired edge, which is operator
work in the catalog.

**Owner:** unassigned

**Refs:** `docs/business/MONETIZATION.md`, `PRD.md` `OT-P2-001`

### 2026-09-03 — Out of credit is indistinguishable from a server error in the reference client

**Symptom:** The canonical metered client discards the response body on any
non-success status, so a caller cannot tell a genuine out-of-credit refusal from
a provider fault. It also counts refusals toward its circuit breaker, so three
users out of credit can open the breaker for everyone.

**Root cause:** The reference client does decode the body, then discards the
decoded error text and returns only the status code, and it counts every
non-success status as a breaker failure.

**Workaround:** Do not copy that client. A correct example already exists in
the fleet: the switchboard metering client surfaces the response body. Branch on
the out-of-credit status before any breaker logic, and surface a refusal that
names the ceiling.

**Real fix:** Upstream client fix, or a shared client that preserves the
distinction.

**Owner:** unassigned

**Refs:** `docs/concepts/INTEGRATIONS.md` failure modes

## Architecture Drift

Places where the built shape is expected to pull away from the documented one,
recorded so the drift is noticed rather than absorbed.

| Risk | Why it would drift | What catches it |
|---|---|---|
| A `stop` or `pause` appears | It is the obvious product affordance, and a customer will ask for it | The structural test required by `OT-P0-007`, which asserts no such method exists |
| A second Machine object appears | "Machine" is the natural English word for what this scenario creates | Review against `DOMAINS.md`; the object is an Instance |
| SSH creeps back in | Adoption of an existing host looks like it needs a shell | Bridge onboarding is the only path; there is no SSH dependency to reach for |
| The reconciler starts fixing things | Reporting feels incomplete when the fix is obvious | Findings are rows with an operator-driven status; automated resolution is a contract change |
| Documented dependencies and `.vrooli/service.json` disagree | Intent gets recorded in prose and never reaches the manifest | Compare `INTEGRATIONS.md` against the manifest during review |

## Cross-references

- [DECISIONS.md](DECISIONS.md) — why the constraints above exist
- [../concepts/INTEGRATIONS.md](../concepts/INTEGRATIONS.md) — upstream dependencies and their failure modes
- [../guides/troubleshooting.md](../guides/troubleshooting.md) — template and tooling trouble
- [PROGRESS.md](PROGRESS.md) — what has actually been done
