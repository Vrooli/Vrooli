---
name: "channel-manager"
description: "Operate Channel Manager as the durable account-operations floor: read identity and queue state, record manual warming actions and observations, and keep every write inside the operator's D-009 automation gate."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["tools"]
  tags: ["channel", "identity", "warming", "queue", "manual-executor"]
  status: "active"
  revision: 1
  createdAt: "2026-09-12T00:00:00Z"
  updatedAt: "2026-09-12T00:00:00Z"
  requires:
    scenarios: ["channel-manager", "content-desk"]
    commands: ["channel-manager channel"]
  origin:
    kind: "authored"
---

## Tools focus: Channel Manager

Channel Manager is the account-operations floor: platform identities, warming
programs, account health, and one unified action queue per identity that also
carries releases from Content Desk. Manual execution is the permanent
first-class executor, not a scaffold. Browser automation is an operator
decision recorded per platform and action kind (D-009), and it stays disabled
until that row exists.

Every command sits under the `channel` group: `channel-manager channel <command>`.

### Decision tree

```
I need to work an account
├─ Read what needs attention
│  ├─ identities, due actions, flags → channel-manager channel overview [S1]
│  └─ one identity's redacted history → channel-manager channel timeline <identity> [S1]
├─ Identity does not exist yet
│  └─ create with non-secret metadata and an environment reference → channel-manager channel create ... [S1]
├─ Change non-secret identity metadata or operator policy
│  └─ channel-manager channel edit ... [S1]
├─ Identity is finished
│  └─ retire without deleting audit history → channel-manager channel retire <identity> [S1]
├─ Advance a warming program
│  ├─ preconditions attested → channel-manager channel start <identity> --program <id> [S1]
│  └─ start refuses on preconditions → report the missing attestation; the CLI cannot record it [S0]
├─ Execute queued work
│  ├─ queue a manual platform action → channel-manager channel queue --identity <id> --kind <kind> [S1]
│  └─ record a completed manual action → channel-manager channel complete <action> [--evidence <ref>] [S1]
├─ Move a platform to browser execution (operator-gated)
│  ├─ no D-009 acceptance row → keep the work manual; do not assign or dispatch [S0]
│  ├─ accept the exposure → channel-manager channel assign-automation ... (operator decision) [S1]
│  └─ action is due and approved → channel-manager channel dispatch-browser <action-id> [S1]
├─ Record what happened
│  ├─ reach observation → channel-manager channel observe <identity> --value <n> [S1]
│  └─ post metric for a completed release → channel-manager channel metric <release> --sample-id <id> --metric <name> --value <n> [S1]
└─ Improve this scenario's instruments or skills
   └─ no improve skill is declared yet; read channel-manager docs/internal/PROBLEMS.md and file the gap [S0]
```

Prefer the `overview` read over assembling state from several identity reads; a
fresh agent that skips it re-derives due actions and flags inconsistently.

### What the readings mean

**A flag is a measurement, never a verdict.** A signal states a reach
observation against its baseline and carries the observations that raised it.
The only automatic effect is that the identity's queue is paused. Do not post
more, post less, or change niche in response, and never assert that an account
has been penalized. Platform penalties are unobservable.

**One queue per identity.** Warming, maintenance, and publishing share the same
cadence budget because platforms count actions per account. A release outranks
maintenance engagement; a deferred maintenance action stays queued with its
deferral recorded, and a miss is written to the observation log rather than
dropped. Do not open a second queue or raise a program's ceiling to fit more in.

**Eligibility fails closed.** The Content Desk seam returns a three-valued
result. `unknown` is not eligible; report it and name the missing identity or
attestation rather than releasing.

**Quarantine is terminal.** A failed warm is rebuilt, never resumed.
`quarantined → running` is an illegal transition.

**Attestations are recorded, not provisioned.** The scenario stores environment
references and cannot verify them. An unverifiable gate is still a gate.

**Secrets never cross the CLI.** `--credential-ref` is a credential-authority
reference, never a value. If a command asks you for a secret, stop: the fix is
an authority reference, not a literal.

### In-use settings

| Symptom | Move |
|---|---|
| The API is unreachable | Pass `--auto-start` or run `vrooli scenario start channel-manager`. Never invoke the API binary directly. Journal which route fixed it. |
| A metric replay would double-count | Reuse the same `--sample-id`; a new id creates a second sample. Only the sample id is the retry guarantee. |
| A program asks for more actions than the platform allows | The platform descriptor clamps it at plan generation. Do not edit the program or descriptor to raise the ceiling; report it. |
| A flag paused an identity and you want to resume | Do not clear the flag. Report the measurement and its observations; an unresolved flag keeps the queue paused. |
| You want to move a platform to browser execution | The operator records a D-009 acceptance row per platform and action kind first. Until then, keep execution manual. |

### Troubleshooting & Edge Cases

| Situation | Response |
|---|---|
| `channel-manager` reports "API unreachable at ..." | Start the scenario via `vrooli scenario start channel-manager` or pass `--auto-start`; do not run the API binary directly. |
| `start` refuses with an unsatisfied precondition | An environment attestation is missing. Record the environment reference first; do not bypass the gate. |
| An eligibility read returns `unknown` | Treat as not eligible. Do not release the draft; report the missing identity or attestation. |
| An identity is `quarantined` | Terminal. Rebuild a new identity; `quarantined → running` is illegal. |
| `dispatch-browser` is rejected | Expected while no D-009 acceptance row exists for that platform and action kind. Complete the action manually and record the evidence. |
| A reading looks empty | An empty list is a measured zero, not a failure. A failure carries a reason; read the reason before retrying. |

### Promotion note

The `[S1]` leaves above are single commands whose `--help` already carries their
flags, so this skill names the decision and not the flag list. No
channel-manager program exists yet; the strongest promotion candidates are
`overview` and `metric`, but they stay leaves until a repeated join earns a
program.
