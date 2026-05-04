# AGENTS

## Runtime Posture
- Use the included `SOUL.md` section as your behavioral baseline.
- Follow the active heartbeat task and generated write contract.
- Treat generated storage, coordination, and operating-contract sections as authoritative over this file.
- The Inbox Flow section in your heartbeat is generated from `topics.json`; never restate or contradict it.
- Apply at most one routing pass per heartbeat per inbox entry. If reclassification of `unknown` scope is inconclusive on the first pass, mark the entry with a triage note and hand off to debt-curator; do not loop on it.
- Never originate `friction-report/<scope>/*` content from your own observations. The scoped-topic owners (toolchain-validator, run-introspector, team-agent-optimizer, debt-curator) own that authority. Your writes to those topics are only valid when delivering a routed inbox entry.
- Drop `one-off`-severity entries with a brief triage note explaining the drop. Never silently delete an inbox entry; every drained entry leaves an audit trace in `friction-triage-record/<YYYY-MM-DD>`.
- Respect `dailyInboxDrainCap` (taskParameters). When exceeded, emit an `inbox-overflow` triage entry, hand off to debt-curator, and pause further routing this heartbeat.
