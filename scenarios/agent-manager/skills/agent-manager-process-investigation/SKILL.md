---
name: "agent-manager-process-investigation"
description: "Diagnose why agent runs fail by classifying root causes as Environment/Tooling or Agent Setup through active codebase exploration."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["contract"]
  tags: ["agent-manager","investigation"]
  status: "active"
  revision: 3
  createdAt: "2026-02-17T00:00:00Z"
  updatedAt: "2026-03-15T00:00:00Z"
  requires:
    scenarios: ["prompt-manager"]
    commands: ["prompt-manager skill", "prompt-manager skill read"]
  origin:
    kind: "authored"
---
## Investigation contract

Diagnose the supplied run evidence and return the workflow's declared result.
Use `prompt-manager skill read investigating-agent-runs` for the investigative
method, severity definitions, and drill-down choices.

### Variable legend

`{{.context}}` contains bounded RunReport projections, run identifiers, and
operator context. The durable run report and its CLI drill-down commands are
authoritative; it deliberately excludes transcript bodies and unified diffs.

`{{.evidence_refs}}` contains the bounded typed evidence references supplied by
the caller. Cite only these exact references in a category or recommendation's
`evidenceRefs`; never invent a locator, owner, revision, or schema version.

{{.context}}

### Outcome work table

| Proven evidence | primary category | confidence |
| --- | --- | --- |
| A failed command, unavailable service, policy denial, or runtime mismatch is sufficient to explain the run | Environment/Tooling | High |
| The environment evidence is healthy and an instruction, capability declaration, or context omission explains the failure | Agent Setup | High |
| A run identity is refused a lifecycle operation that the task explicitly instructed it to perform | Both | High |
| Independent evidence proves both conditions materially contributed | Both | Medium or High |
| A successful or failed run has repeated work, rereads, or avoidable waiting proven by the efficiency method | Efficiency/Friction | Medium or High |
| The report has a discriminator but no payload proves its cause | Both | Low |

Conservative default: when a required predicate is unproven, retain the
uncertainty in coverage, use `inconclusive` disposition, and use `unknown` for
root cause rather than guessing.

### Authority boundary

Read the bounded evidence and permitted drill-downs, classify only causes
supported by that evidence, and recommend bounded corrective work. Do not
modify files or launch/stop/resume runs.

If evidence shows that a run identity was refused an operator-only lifecycle
operation, do not retry or execute that operation. State that the task must
move it to an operator context and investigate the task/guardrail mismatch.

Return the bounded structured result declared by the workflow. Include
`condition`, `disposition`, `rootCause`, `confidence`, and
`unprovenPredicates` when the evidence supports them. Use `no_intervention`
only with zero recommendations; use `inconclusive` when a required predicate
is unavailable or stale; and use `unknown` for a root cause that the evidence
does not establish. Evidence prose explains a conclusion but never proves it;
only typed evidence references can support a finding or recommendation.
