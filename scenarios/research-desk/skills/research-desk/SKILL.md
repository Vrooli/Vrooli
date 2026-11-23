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

Research Desk is a lens, launcher and courier over the research home (`~/.vrooli/research`);
it is never in the research loop. Every read has a file behind it, so a stopped desk changes
nothing for managers or owners. File grammar: `docs/reference/FILE-CONTRACT.md`.

| Need | Command | Without the desk |
|---|---|---|
| What is moving, what needs the operator | `research-desk now`, `commissions [--state]`, `commission <id>`, `topics` | `cohorts/<id>/STATE.md`, `ATTENTION.md` |
| What changed since a time | `research-desk changes commission\|topic <id> [--since]`, `mark-seen [--all]` | file times, `LEDGER.md`, run `CHECKPOINT.md` |
| Prepare or check a packet | `research-desk preflight <spec\|id>`, `build <spec>\|--relaunch-of <id>` | `research-coordination/scripts/packet.py preview\|build\|check` |
| Launch, pause, resume, stop, discard, recover an interrupted launch | `research-desk launch <id> [--supersede\|--alongside <ids>]`, `launch-messages <id>`, `control <id> <action>` (`retry-goal` only for a desk launch failed at `ready_seen`/`goal_sent`/`goal_verified`) | At `ready_seen`, retry resolves exactly one top-level thread from the recorded session; it reads that session's goal rows first, then sends only if the target thread is idle, shown and has no matching goal. Existing matching goals reconcile without typing; ambiguous threads, unreadable goals, or different objectives fail closed |
| Steer or wind down | `research-desk say <id> direction\|wind-down` | append an entry to `cohorts/<id>/OPERATOR.md` |
| Answer a needs-you item | `research-desk needs-you`, `answer <id> <item> <option> [--note]` | append `answer <ITEM>` to `OPERATOR.md` |
| Reviews and verdicts | `research-desk reviews`, `verdict <review> <topic> <words>` | `reviews/<id>/`, the `research-review` skill |

`now` and `commission` print `FLAG` on a window with no `STATE.md` an hour in or a `STATE.md` that contradicts its goal; `launch` refuses over a read ceiling (`templates/limits.json`).

Use each command's `--help` for its flags and JSON output. A field the desk cannot derive reads
`unknown`; do not replace it with a guess. Never send messages into a manager's chat to
steer it: use `OPERATOR.md`.

Launch and control can be delegated through the shared `research-management` policy.
Use the exact `vrooli agent authorize` request in a refusal; targets are
`research-desk:commission/<id>`. Superseding an older commission also requires control
coverage for that commission. Preserve caller identity; never strip credentials to
imitate an operator shell. Answers and review verdicts remain operator decisions.
