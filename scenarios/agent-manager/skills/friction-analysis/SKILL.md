---
name: "friction-analysis"
description: "Investigate recurring agent friction across four lenses: deterministic tool/runtime evidence, operator signals, and gated inferred annotations."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["practice"]
  tags: ["agent-manager","friction","investigation","annotations","operator-signals"]
  icon: "search"
  status: "active"
  revision: 1
  createdAt: "2026-11-18T00:00:00Z"
  updatedAt: "2026-11-18T00:00:00Z"
  requires:
    scenarios: ["agent-manager"]
    commands: ["agent-manager usage friction", "agent-manager usage runtime-friction", "agent-manager usage annotations", "agent-manager measures get", "agent-manager run report", "agent-manager run events", "agent-manager run messages", "agent-manager conversation search", "agent-manager conversation context", "agent-manager annotations coverage", "agent-manager annotations show", "agent-manager annotations run", "agent-manager usage friction-report"]
  origin:
    kind: "authored"
---
## Friction analysis

Use this when an operator asks why agents keep struggling, or when a friction finding needs
a cause. Read `prompt-manager skill read agent-manager` and
`prompt-manager skill read reviewing-agent-run-efficiency` first.

Friction has four lenses. Each proves something different; none proves a cause by itself.

| Lens | What it proves | What it cannot prove | Read |
|---|---|---|---|
| Tool | A command path failed, guessed its shape, printed too much, was polled, was slow, or was bypassed | Why the agent chose it, or whether the work mattered | `usage friction`, `measures get tool-profiles` |
| Runtime | Where measured run time went: blocked checks, validation waits, rework, sandbox, agent-manager failures | Whether the time was necessary | `usage runtime-friction` |
| Operator | What the operator reported or corrected, in their words | The operator's actual mood; a label is a text signal | `usage friction` operator section |
| Inferred | Gated labels such as spinning, discovery-slow, objective-lost, scope drift, run type, affect | A verdict; an unproven label is evidence, not a rate | `usage annotations`, `annotations coverage`, `annotations show` |

### Method

1. **Start with the deterministic finding.** Read its owner, cost, share, evidence runs and
   coverage. Pick one finding or one owner; do not scan the whole list.
2. **Read the evidence runs.** `run report <id>`, then `run tools` / `run events` only where
   a discriminator needs payload evidence. Keep the run's own work in scope: a run working
   on a scenario is not bypassing it.
3. **Add annotations only for the residual.** Use `annotations show <run-id>` to see the
   labels and probabilities for the evidence runs. If the deterministic lens already
   explains the waste, stop; do not decorate it with AI labels.
4. **Check the gate and coverage before quoting a label.** `usage annotations` and
   `annotations coverage` name which labels are proven, their precision/recall, and how many
   runs were observed. An unproven label is a lead. It is never a rate and never a finding on
   its own.
5. **Corroborate before concluding.** A recurring pattern needs at least three independent
   runs, a baseline, and either a deterministic finding or a proven label. Prefer two
   independent lenses over one strong-sounding label.
6. **Recommend the smallest durable fix.** A CLI alias, a help fix, a skill change, a
   timeout, a blocking sibling, or a corpus/owner fix. File it with `usage friction-report`
   for tool findings; use `report-bug` for another owner's defect.
7. **Record what remains unknown.** Missing events, older fact versions, unclassified
   messages, and unproven labels are unknowns. State them; never map them to zero.

### Reading the affect labels

- `operator.valence`, `operator.frustration` and `operator.satisfaction` are text signals
  about the operator's message, not a measurement of the person.
- A mood series is built from valence over time, with explicit frustration/satisfaction as
  markers. Use it for correlation and review, never as a performance verdict on an agent or
  a person.
- Six-way `operator.emotion` is optional and usually unproven; small local decision models
  are weak at it. Do not present it as a rate until its gate passes.

### Anti-patterns

- Quoting an unproven label as "the agent was confused".
- Treating a single run's label as a pattern.
- Inferring a cause from a correlation between a finding and a label without a baseline.
- Using operator frustration as proof that the agent was wrong.
- Reporting an empty annotation section as "no friction".
- Tuning a label's threshold on the same window being reported.

### Output

Return: the finding (or owner), the deterministic evidence, the annotation corroboration
and gate state, the affected runs, the cost, confidence, the smallest recommended action,
and what remains unknown.
