---
name: "vrooli-events"
description: "Publish, query and stream events between scenarios: ingest an event for other scenarios to react to, query the durable event history, subscribe to a live event stream, and aggregate which scenarios and operations are being invoked."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  status: "active"
  modes: ["tools"]
  tags: ["vrooli-events", "events", "event-bus", "publish", "subscribe", "stream", "sse", "pubsub", "inter-scenario"]
  requires:
    scenarios: ["vrooli-events"]
    commands: ["vrooli-events events ingest"]
  origin:
    kind: "authored"
---

## vrooli-events

A durable event store with live subscriptions.

| You want | Command |
|---|---|
| Publish an event | `vrooli-events events ingest` |
| Look at past events | `vrooli-events events query` |
| Stream matching events live | `vrooli-events events subscribe` |
| Which scenarios and operations are used | `vrooli-events events aggregate` |
| Store health | `vrooli-events store stats` |

Run `vrooli-events <group> <command> --help` for flags.
