---
name: "treasury"
description: "Check a spend attempt against a grant: what an agent is allowed to spend or charge, whether a charge is authorized under its mandate, spending caps and budgets. Create, revoke or cancel a mandate or recurring charge, propose a charge for authorization, and read the decision."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  status: "active"
  modes: ["tools"]
  tags: ["treasury", "spend", "budget", "mandate", "authorization", "charge", "caps", "grant", "payments", "wallet"]
  requires:
    scenarios: ["treasury"]
    commands: ["treasury mandate list"]
  origin:
    kind: "authored"
---

## treasury

Every attempt is evaluated against a grant a human created; nothing is spent without a mandate.

| You want | Command |
|---|---|
| What may an agent spend | `treasury mandate list` |
| Ask to charge | `treasury authorization propose`, `treasury authorization get` |
| Create or revoke a mandate (operator) | `treasury mandate create`, `treasury mandate revoke` |
| Caps | `treasury budget set-caps` |
| Stop a recurring charge | `treasury mandate cancel-standing` |
| Safety rules | `treasury safety invariants` |

Run `treasury <group> <command> --help` for flags.

Mandate creation and caps are operator actions.
