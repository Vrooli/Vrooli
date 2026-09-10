---
name: "swarm-manager"
description: "Use Swarm Manager to inspect backlog and goals, review development contracts, preserve operator authority, and follow one owned next action through evidence and decision."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["tools"]
  tags: ["backlog", "goals", "development", "approval", "evidence", "operator", "learning-spine"]
  icon: "list-checks"
  status: "active"
  revision: 2
  createdAt: "2026-09-09T00:00:00Z"
  updatedAt: "2026-09-10T00:00:00Z"
  learning:
    scope: "swarm-manager-usage"
    capture: "every attempt"
  requires:
    scenarios: ["swarm-manager", "vrooli-memory", "prompt-manager"]
    commands: ["swarm-manager backlog", "swarm-manager goals", "swarm-manager development", "vrooli-memory recall", "vrooli-memory learning record", "prompt-manager skill read"]
  origin:
    kind: "authored"
---
## Tools focus: Swarm Manager

Use Swarm Manager as the operator-facing control plane for backlog work, goals,
evidence and retained development contracts. It stores durable decisions and
projects the next action; an agent or review result cannot complete work or widen
its authority. Role and rung ownership are defined in
`path:docs/agent-system/SKILL_AUTHORING.md`.

### Before acting

1. Recall `swarm-manager-usage` through `vrooli-memory recall wake --scope
   swarm-manager-usage`. Apply advice only when it matches the current work
   identity and record the choice through the shared Memory path.
2. Identify the canonical item as `kind/name`, or the exact goal name. Read its
   current state before choosing an operation.
3. For a new capability or unfamiliar route, discover it with
   `search-hub query` or `prompt-manager discover` before assuming that no
   command exists.

### Choose one operation

Read the rows in order. Each leaf is one bounded step.

| Situation | One step |
|---|---|
| Need current backlog state | Run `swarm-manager backlog get --kind <kind> --name <name>`. **[S1]** |
| Need the actionable backlog inventory | Run `swarm-manager backlog list --json`, then use the server's status and next-action fields. **[S1]** |
| Need goal standing or next work | Run `swarm-manager goals get --name <goal>` or `swarm-manager goals list --json`. **[S1]** |
| Need to understand why a next action is offered | Read the item detail and its plan, dependency, evidence and execution status before acting. The server-owned next-action projection is authoritative. **[S0]** |
| Need to create or refine work | Use `swarm-manager backlog create`, `backlog update`, or the Plan Workshop route after reading the current item. Preserve the existing work identity. **[S1]** |
| Need to inspect a development target | Run `swarm-manager development get --item <kind/name>`. Preview output is review material and grants no authority. **[S1]** |
| Need a development contract | Use the typed preview and exact retained artifact revision. Approval is a human decision and does not launch an agent. **[S1]** |
| Need to approve, revoke, or accept a development decision | Prepare the typed request file with its expected version and run `swarm-manager development approve|revoke|accept --file <request.json>`. **[S1]** |
| Need to recover a retained artifact | Run `swarm-manager development artifact --item <kind/name> --digest <digest> --path <path>`. Verify the returned digest before use. **[S1]** |
| Need to execute an approved scenario-improvement plan | Queue the ordinary plan-backed item with `execution_strategy=adaptive-improvement`. The retained plan acceptance supplies the target revision, grant, write scope and aggregate limits; the campaign chooses successive in-scope repairs without creating another backlog item. **[S0]** |
| Need successive repairs under one approval | Continue the existing campaign checkpoint; do not create a new backlog approval for each in-scope repair. Keep required, completed and remaining outcomes distinct. **[S0]** |
| Need owner progress or a pending operation | Attach to the owner execution and checkpoint identity. Wait through the owner surface once; do not start a replacement run after reconnect. **[S0]** |
| Need evidence or final disposition | Read owner-derived evidence, Test Genie results and the independent review. Submit a typed operator decision only after required evidence is present. **[S1]** |
| Need a new goal, target, effect, device, credential or paid provider | Stop at the boundary and request the specific operator decision or amendment. A sensor, review, or recommendation never grants it. **[S0]** |

The adaptive-improvement strategy is scenario-neutral. Audio Tools is one
possible target and keeps its product, corpus, provider and accounting authority;
Swarm supplies orchestration, identity, checkpoints and operator disposition.
The internal `contract-development` transition remains a compatibility adapter
for older records, not a second operator-facing backlog kind or approval path.
Use `prompt-manager skill read scenario-improvement-campaign` for campaign
execution and the target scenario's `*-improve` skill for domain judgment.

### Development continuity

The retained approval binds the exact target revision, proposal and artifact
contents. The campaign checkpoint binds the approval digest, attempt key, owner
execution, required/completed/remaining outcomes, last checkpoint and no-progress
cycles. On restart, read those fields and resume only when the owner and grant
are still valid. A changed target or grant requires the supported amendment and
fresh operator acceptance. Cancellation first records the local intent, then
requests owner cancellation using its stable operation identity; an owner failure
leaves the canceled reservation and pending acknowledgement visible.

### Evidence and decisions

Treat these states separately:

- `available` is an owner response, not acceptance.
- `unavailable`, stale, unknown, missing and failed evidence remain unmet.
- A workflow terminal result is a child outcome to inspect, not an operator
  decision.
- A completed campaign still needs the configured human acceptance after the
  required evidence submission.

Do not infer product quality from a readable dashboard, a successful wrapper,
the number of engines, or a passing local fixture. Keep build, provider, corpus,
device, workflow, approval and grant identities together with the evidence they
qualify.

### In-use settings

| Symptom | Safe move | Verification |
|---|---|---|
| Item changed since review | Re-read the item and preview fingerprint | Approval version and target digest match |
| Preview is stale | Discard the mutation and obtain a fresh preview | Expected-version conflict is visible; no state changed |
| Owner is still running | Use the owner wait/get surface once | Same execution and attempt identities remain |
| Evidence source is unavailable | Preserve the reason and stop acceptance | Row remains unknown/unmet |
| Work needs a broader scope | Request an amendment | New scope and operator decision are retained |
| Cancellation of an ordinary plan execution is requested | Use `swarm-manager execution cancel --id <execution-id>` | Durable cancellation intent, original owner identity and settlement remain visible |
| Revocation of a retained development engagement is requested | Use `swarm-manager development revoke --file <request.json>` with its expected version | Revocation intent, acknowledgement and stable operation ID remain visible |

Never edit the event store or approval files by hand, run a scenario binary
directly, bypass the transition runner, or put credentials and raw private
transcripts in prompts or learning records. Lifecycle changes use `make start`,
`make test`, `make logs` and `make stop` or the `vrooli scenario` control plane.

### Learning and debugging

For direct operations, use the manual learning path in
`prompt-manager skill read vrooli-memory`; retain one attempt identity and the
observed result, including refusal or unavailable status. Programs own automatic
capture when a declared program is introduced later; do not duplicate it here.

Debug in this order:

1. `vrooli scenario status swarm-manager`.
2. Read the canonical item, goal or development engagement once.
3. Inspect the returned owner, workflow, approval, grant and evidence identities.
4. Read the specific owner status or Test Genie receipt named by the response.
5. Re-discover the command or binding only if the contract is missing or changed.

### Safety

- Human approval is required for target acceptance, development authority,
  amendments, final disposition and effects outside the retained grant.
- Keep ordinary repairs inside one approved engagement when the target and grant
  permit them. Do not file per-repair approvals or create duplicate work.
- Never weaken a requirement, delete a known-issue ledger, suppress a finding,
  or replace unknown evidence with a passing value. Apply D1, D2 and D3 from
  `prompt-manager skill read improvement-do-and-dont`.

### Troubleshooting & Edge Cases

| Result | Next action |
|---|---|
| `revision_conflict` | Re-read the current item or engagement and request a fresh typed mutation |
| `not_approved` or `launch_blocked` | Preserve the blocker; approval and owner resolvers are separate gates |
| `owner_unavailable` | Keep the checkpoint and wait identity; do not substitute a local success |
| `evidence_unavailable` | Keep the row unmet and route instrumentation to its owner |
| `budget_exhausted` | Stop new effects and return the bounded checkpoint |
| `canceled` | Do not reserve new work until the cancellation state is reconciled |
| `scenario_unreachable` | Check managed lifecycle status and retain the unknown observation |
| no command matches the intent | Run discovery again with the exact intent; report an unresolved capability only after the discovery path is exhausted |
