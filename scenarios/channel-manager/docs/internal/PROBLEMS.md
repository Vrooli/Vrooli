# Problems — Channel Manager

Persistent register of known issues, tech debt, and deferred work
specific to **this** scenario. Future agents read this file to avoid
re-discovering the same constraint.

Entries below were recorded during the 2026-07-28 design pass, before any product
code existed. They are open questions and known gaps rather than defects — recorded
so the implementing agent does not mistake an unverified assumption for a settled
one.

## What belongs here

- **Known bugs** that are real but not yet worth fixing
- **Tech debt** — workarounds that need a real fix later
- **Deferred work** — features descoped from a phase, with the reason
- **Architecture drift** — code/docs/tests that no longer line up with
  the intended capability map or boundary model
- **Constraints discovered the hard way** that aren't visible from
  the code (e.g., "this resource needs warm-up before the first call;
  see commit X")

## What does NOT belong here

- **Generic template issues** — those go in
  [`../guides/troubleshooting.md`](../guides/troubleshooting.md)
- **Open feature requests** — track those in PRD operational targets
- **Code comments** — if the constraint is local to one file, a
  comment there is more discoverable
- **Test failures** — fix them, don't document them

## Entry template

Use this shape so entries are scannable. Append newest at the bottom.

```markdown
### YYYY-MM-DD — short title

**Symptom:** What goes wrong, observable from outside the system.

**Root cause:** What actually causes it (or "unknown" if not yet diagnosed).

**Workaround:** What to do today to keep moving.

**Real fix:** What needs to happen for this entry to be deleted.

**Owner:** Who should drive the fix (or "unassigned").

**Refs:** Code paths, related issues, prior commits.
```

## Entries

### 2026-07-28 — every warming threshold is unvalidated

**Symptom:** The shipped warming programs and platform cadence ceilings look like
specifications but are hypotheses. A reader could reasonably act on them as though
they were platform documentation.

**Root cause:** They were distilled from operator practice shared publicly, not from
any platform's published behaviour, and none has been measured on an account this
fleet operates. See D-002.

**Workaround:** Every descriptor carries a `provenance` block with
`confidence: speculative` and a note saying so, surfaced wherever the program is
displayed (`CHANMGR-P0-018`).

**Real fix:** `CHANMGR-P1-006` — accumulate real outcomes in `observations[]` until
the observations are better evidence than the source note, then raise `confidence`.
The shipped programs name five completed runs as their own revisit trigger.

**Owner:** unassigned.

**Refs:** `data/warming-programs/tiktok/*.json`, `data/README.md`,
`docs/internal/DECISIONS.md` D-002.

### 2026-07-28 — the BAS dispatch contract is unread

**Symptom:** `CHANMGR-P1-001` (browser executor) is specified against an integration
whose call shape nobody has confirmed.

**Root cause:** The design pass verified that `browser-automation-studio` exposes
`workflows`, `executions`, `schedules`, and `session-profiles` CLI domains — enough
to establish that per-identity session isolation is possible — but did not read how
a workflow is dispatched or how a session profile binds to a run.

**Workaround:** The browser executor is sequenced last among P1, and the manual
executor covers every action in the meantime.

**Real fix:** Read the BAS dispatch and session-profile contracts before planning
`CHANMGR-P1-001`, and model dispatch as a sub-flow of the queued-action lifecycle
once the shape is known.

**Owner:** unassigned.

**Refs:** `docs/concepts/INTEGRATIONS.md`, `docs/concepts/FLOWS.md` § Deferred.

### 2026-07-28 — platform format limits are deliberately absent

**Symptom:** `data/platforms/tiktok.json` declares that video is supported but
carries no duration or file-size limit.

**Root cause:** Guessing a limit is worse than omitting one — an omitted field fails
loudly at validation, whereas a wrong constant silently rejects valid media or
accepts invalid media.

**Workaround:** None needed until the first post action is queued.

**Real fix:** Populate from current platform documentation before the first TikTok
identity is created. The descriptor's own `revisit_trigger` says the same.

**Owner:** unassigned.

**Refs:** `data/platforms/tiktok.json`.

### 2026-07-28 — environment attestations are unverifiable

**Symptom:** An identity can attest that its proxy is region-locked and its
fingerprint unique when neither is true, and the scenario cannot tell.

**Root cause:** Device, proxy, and region are provisioned upstream by third-party
services this scenario deliberately does not integrate with (D-006). A leak degrades
distribution in a way that is indistinguishable from an ordinary failed warm.

**Workaround:** Preconditions still block program start, so the attestation is at
least deliberate rather than implicit. A failed trust-check on an aggressive program
is weak evidence that the environment was not as clean as attested.

**Real fix:** Only available if an environment provider exposes a programmatic
check. Until then this is an accepted limitation, not debt.

**Owner:** unassigned.

**Refs:** `docs/internal/DECISIONS.md` D-006, PRD § Operational risks.

### 2026-07-28 — only one platform is described

**Symptom:** `data/platforms/` contains TikTok alone, while `CHANNELS.md` names
eight channels and the PRD claims coverage of them.

**Root cause:** TikTok was described first because it is the platform the AI-UGC
research covers and the one where warming matters most. The others need their own
research pass and would otherwise be filled with guesses.

**Workaround:** Descriptor validation fails for an identity naming an undescribed
platform, so this surfaces as a clear error rather than as silent misbehaviour.

**Second-order consequence:** this is not only a coverage gap — it makes
`CHANMGR-P0-003` unprovable. "Adding a platform requires no code change" cannot be
demonstrated against a single descriptor, because the abstraction has never been
asked to bend. A second *structurally different* platform is therefore part of
meeting the requirement rather than a later expansion, and the pair must differ in
shape rather than only in name: X is text-led with no video action kinds and a
different disclosure posture, which is what makes it the useful second.

**Real fix:** X described alongside TikTok before `CHANMGR-P0-003` is called done,
each with its own provenance; then one descriptor per remaining active channel.
Reddit is the likely third — it already has live accounts per `CHANNELS.md`.

**Owner:** unassigned.

**Refs:** `data/platforms/`, `docs/marketing/strategy/CHANNELS.md`.

## Architecture Drift

Use this section for deferred findings from `screaming-architecture-audit`.
Do not create a standalone architecture-audit report unless the work is
a migration handoff with a planned retirement path back into
`ARCHITECTURE.md`, `SEAMS.md`, or this file.

| Area | Drift | Maturity Impact | Real Fix |
|---|---|---|---|
| Whole scenario | No drift is possible yet: `api/internal` holds only the template scaffold, so there is no code to diverge from the documented boundary model. | None. The maturity table in `ARCHITECTURE.md` reports every surface as Scaffold, which is accurate. | Run `screaming-architecture-audit` after the first product domain lands, and record findings here rather than in a standalone report. |

## Cross-references

- [`PROGRESS.md`](PROGRESS.md) — lifecycle log (forward-looking)
- [`SEAMS.md`](SEAMS.md) — boundary registry (load-bearing for tests)
- [`TESTING.md`](TESTING.md) — test patterns
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — generic-template issues
