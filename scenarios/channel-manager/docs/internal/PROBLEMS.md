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

### 2026-07-28 — Vault content round-trip requires an operator-scoped token

**Symptom:** The managed Vault endpoint is reachable on port 8200, but
`resource-vault content set` refuses a scratch write because `VAULT_TOKEN` is not
present in this development session.

**Root cause:** Managed Vault intentionally requires an explicitly supplied scoped
token for content operations. Channel Manager correctly stores only a Vault path,
so this does not affect the credential-free manual workflow or its tests.

**Workaround:** An authorized operator supplies a scoped `VAULT_TOKEN`, performs the
documented scratch round-trip in `SEAMS.md`, then deletes the scratch path.

**Real fix:** None in this scenario; provisioning scoped Vault access is owned by
the Vault resource/operator.

**Owner:** Vault resource operator.

**Refs:** `docs/internal/SEAMS.md` § BAS and Vault execution handoff,
`resources/vault/docs/QUICKSTART.md`.

## Architecture Drift

Use this section for deferred findings from `screaming-architecture-audit`.
Do not create a standalone architecture-audit report unless the work is
a migration handoff with a planned retirement path back into
`ARCHITECTURE.md`, `SEAMS.md`, or this file.

| Area | Drift | Maturity Impact | Real Fix |
|---|---|---|---|
| Whole scenario | The former Notes template domain was removed and the channel-manager domain now owns the durable state, routes, CLI, and formal action lifecycle. | The old scaffold statement was stale. | Keep new behavior inside `internal/channelmanager` and update the formal flow when changing action statuses. |

## Work ladder

- Rung: W4
- Evidence: comprehensive Test Genie run `20260728-231326-13fe99a2` passed all 20 phases, including the synthetic BAS manual-operator path. The formal action lifecycle check passes with run `960df471-88c7-4ff7-be04-7d8e0447d92f`.
- Blocker: No platform-account work is authorized. The remaining real-account validation is intentionally deferred; the manual UI and credential-free synthetic path are the supported validation boundary.
- Measured: 2026-07-28

## Cross-references

- [`PROGRESS.md`](PROGRESS.md) — lifecycle log (forward-looking)
- [`SEAMS.md`](SEAMS.md) — boundary registry (load-bearing for tests)
- [`TESTING.md`](TESTING.md) — test patterns
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — generic-template issues
