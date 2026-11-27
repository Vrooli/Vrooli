---
name: "research-desk"
description: "Read and steer supervised research through the research-desk CLI, with the file path that works when the desk is stopped."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["tools"]
  tags: ["research-desk", "research", "usage"]
  status: "active"
  revision: 3
---

## Use Research Desk

The desk reads and steers `~/.vrooli/research`; its file grammar is
[`docs/reference/FILE-CONTRACT.md`](../../docs/reference/FILE-CONTRACT.md). Use the owner program
for the recurring inbox and topic summaries:

| View | Call |
|---|---|
| Operator inbox | `program-runtime library run research-desk.operator-overview --input view=needs-you` |
| Research topics | `program-runtime library run research-desk.operator-overview --input view=topics` |

Read `status` and `signals.result`: `needs-you` is a counts/time/redaction summary (no decision text, answers, options or paths); `topics` is at most eight slugs with numeric mission/commission counts. Treat `partial` or `truncated` as incomplete and `failed`, `refused` or `unavailable` as no summary. Use the owner CLI only for a deliberate detail review; keep private detail out of evidence. Leave caller/provenance unset; the verified session harness supplies identity. For other work:

- Activity: `research-desk now`, `commissions [--state]`, `commission <id>`, `changes <commission\|topic> <id> [--since]`.
- Packets and supervision: `preflight`, `build`, `launch`, `launch-messages`, `control`, `say`; use `--help` for exact arguments.
- Operator decisions: `answer`, `verdict`; reviews: `reviews`. Answers and verdicts remain operator decisions. A fixed wind-down may be delegated only with an approved `research-wind-down` grant for the exact commission; freeform directions remain operator-only.

Use `OPERATOR.md` to steer; never message a manager's chat. Unknown fields stay unknown. Launch
and control use `research-management` authorization. A delegated wind-down uses the separate,
exact-commission `research-wind-down` grant. Do not provide caller or provenance fields; the
verified session harness preserves caller identity. Use the exact authorization request shown
in a refusal. File fallback remains available when the desk is down.
