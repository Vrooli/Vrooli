## Investigation contract

Diagnose the supplied run evidence and return the workflow's declared result.
Use `prompt-manager skill read investigating-agent-runs` for the investigative
method, severity definitions, and drill-down choices.

### Variable legend

`{{.context}}` contains bounded RunReport projections, run identifiers, and
operator context. The durable run report and its CLI drill-down commands are
authoritative; it deliberately excludes transcript bodies and unified diffs.

{{.context}}

### Outcome decision table

| Proven evidence | primary category | confidence |
| --- | --- | --- |
| A failed command, unavailable service, policy denial, or runtime mismatch is sufficient to explain the run | Environment/Tooling | High |
| The environment evidence is healthy and an instruction, capability declaration, or context omission explains the failure | Agent Setup | High |
| Independent evidence proves both conditions materially contributed | Both | Medium or High |
| A successful or failed run has repeated work, rereads, or avoidable waiting proven by the efficiency method | Efficiency/Friction | Medium or High |
| The report has a discriminator but no payload proves its cause | Both | Low |

Conservative default: when a predicate is unproven, retain the uncertainty in
the evidence and select `Both` with `Low` confidence rather than guessing.

### Authority boundary

Read the bounded evidence and permitted drill-downs, classify only causes
supported by that evidence, and recommend bounded corrective work. Do not
modify files or launch/stop/resume runs.
