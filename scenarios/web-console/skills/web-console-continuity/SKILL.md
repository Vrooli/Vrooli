---
name: "web-console-continuity"
description: "Choose the safe Web Console continuity surface for finding, inspecting, recovering, and deleting retained conversations."
license: "MIT"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["practice"]
  tags: ["web-console", "continuity", "archive", "recovery"]
  status: "active"
---

# Web Console continuity usage

Use local continuity when the goal is to recover a Web Console conversation.
Start with `web-console continuity integrity --json` when metadata may be
missing. Use `web-console continuity search <text> --state any --json` for
deterministic local retrieval; results remain available without Agent Manager,
Search Hub, embeddings, or an LLM. Inspect the returned lifecycle state before
choosing reopen or recovery.

Use `web-console continuity reconcile --json` to review a dry-run. Apply only
after a verified backup and reviewed generation, with an explicit
`--operation-id`. Inspect the receipt after apply. Archive and recovery are
preserving actions; permanent delete is a separate explicit operator decision
and must retain its receipt.

Do not infer deletion from a closed pane, browser disconnect, process exit, or
an absent session row. Do not scan raw agent homes manually when the typed
catalog and local search surfaces are available. Agent Manager owns global
cross-run retrieval and Search Hub owns federation; use those only when the
search scope crosses Web Console.
