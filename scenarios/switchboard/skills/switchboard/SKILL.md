---
name: "switchboard"
description: "Reach my agents over the messaging apps people already use: list the available messaging channels, bind an agent to a channel, send a message through a channel, and control what each agent may do for whom."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  status: "active"
  modes: ["tools"]
  tags: ["switchboard", "messaging", "chat", "sms", "telegram", "whatsapp", "channels", "agent-binding", "message-agent"]
  requires:
    scenarios: ["switchboard"]
    commands: ["switchboard channels list"]
  origin:
    kind: "authored"
---

## switchboard

Channels are ordered by setup friction.

| You want | Command |
|---|---|
| Which channels can I use | `switchboard channels list`, `switchboard channels get` |
| Connect an agent to a channel | `switchboard channels create-binding`, `switchboard channels bindings` |
| Send a message | `switchboard channels send` |

Run `switchboard <group> <command> --help` for flags.
