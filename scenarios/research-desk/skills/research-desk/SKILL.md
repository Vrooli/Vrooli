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
[`docs/reference/FILE-CONTRACT.md`](../../docs/reference/FILE-CONTRACT.md). For the two
recurring read views, use the owner program and branch on its envelope status:

| View | Call |
|---|---|
| Operator inbox | `program-runtime library run research-desk.operator-overview --input view=needs-you` |
| Research topics | `program-runtime library run research-desk.operator-overview --input view=topics` |

The needs-you program returns counts only; it never returns decision text, answers, options or paths. Use the owner CLI for a deliberate detail review and keep that body out of evidence. The topics view returns at most eight slugs and numeric mission/commission counts; check `status` and `signals.result.truncated` before treating the overview as complete. For other work:

- Activity: `research-desk now`, `commissions [--state]`, `commission <id>`, `changes <commission\|topic> <id> [--since]`.
- Packets and supervision: `preflight`, `build`, `launch`, `launch-messages`, `control`, `say`; use `--help` for exact arguments.
- Operator decisions: `answer`, `verdict`; reviews: `reviews`. Answers and verdicts remain operator decisions.

Use `OPERATOR.md` to steer; never message a manager's chat. Unknown fields stay unknown. Launch
and control use `research-management` authorization; preserve caller identity and use the exact
authorization request shown in a refusal. File fallback remains available when the desk is down.
