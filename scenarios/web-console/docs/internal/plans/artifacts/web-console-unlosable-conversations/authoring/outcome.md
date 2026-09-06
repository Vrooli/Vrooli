At completion, closing a Web Console pane archives it without destroying metadata, conversation events, transcript checkpoints, agent homes, aliases, titles, or recovery lineage. Every lifecycle mutation passes through one domain-owned transition service. Process exit reports an event but cannot independently decide whether durable state should be deleted.

Web Console maintains a canonical conversation catalog that maps pane identity to agent identity and durable sources. The catalog uses tombstoned lifecycle states rather than absence as state. A reconciliation service detects missing metadata, dangling projections, and unindexed agent homes. It produces a deterministic dry-run report and can idempotently synthesize recoverable records without changing source transcripts.

Operators can search Web Console conversations across live, archived, exited, recoverable, and repaired states by remembered text, time range, agent type, pane identity, agent thread identity, title, current topic, and working directory. Results disclose lifecycle and restore state. An intact transcript never disappears merely because another projection is missing.

The Web Console CLI exposes stable archive, search, inspect, integrity-audit, reconcile, recover, and permanent-delete workflows. Destructive actions require explicit confirmation and durable receipts. The UI presents the same states and actions, including useful mobile behavior and honest degraded states.

Agent Manager remains the global cross-run conversation-search authority. Web Console publishes or exposes stable source and alias information that Agent Manager can import. Search Hub continues to federate through Agent Manager. Local Web Console search remains the continuity and recovery surface for Web Console-owned sessions.

The existing orphan corpus is classified and reconciled. Message-bearing and agent-history-bearing records remain preserved. The triggering Codex thread is discoverable through normal UI and CLI queries such as a last-hour search for `plan-manager`, and it can be resumed without manually overriding `CODEX_HOME`.

The scenario has a checked temporal lifecycle model, an accurate invariants document, requirement-linked tests, durable lifecycle telemetry, operator health measures, scenario-owned usage and improvement skills, and a governed integrity-audit program. Comprehensive validation demonstrates correctness across backends, agent types, API restarts, browser reconnects, concurrent devices, retries, retention, and partial failures.

