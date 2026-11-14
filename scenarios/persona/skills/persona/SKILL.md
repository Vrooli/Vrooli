---
name: "persona"
description: "Fetch a verification code or one-time password the machine cannot receive itself (sent to the persona's email, SMS or device), act as an agent's persona identity, send from its controlled address, and hand off to a human for steps only a person can do."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  status: "active"
  modes: ["tools"]
  tags: ["persona", "identity", "agent-identity", "verification-code", "otp", "email", "sms", "handoff", "human-step", "attestation"]
  requires:
    scenarios: ["persona"]
    commands: ["persona personas create"]
  origin:
    kind: "authored"
---

## persona

Credentials and document bytes are never exposed; secrets stay with their adapters.

| You want | Command |
|---|---|
| Create or list personas | `persona personas create`, `persona personas list` |
| Act as a persona | `persona access act-as`, `persona access resolve` |
| Get a verification code | `persona channels retrieve-code` |
| Send from the persona's address | `persona channels send` |
| Bind a controlled channel | `persona channels bind`, `persona channels list` |
| A step only a human can do | `persona handoffs open`, `persona handoffs get`, `persona handoffs complete`, `persona handoffs resume` |
| Who may use it | `persona access grant`, `persona access list-grants` |

Run `persona <group> <command> --help` for flags.
