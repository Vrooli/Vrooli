# Goal messages

## Orchestrator goal (paste as `/goal …`; ≤2,048 chars)

```text
/goal Work until this verified end state is true: every acceptance row A1–A10 in scenarios/agent-manager/docs/efforts/usage-intelligence/SPEC.md §5 is recorded under "Accepted" in WORKSTREAMS.md with its proof output shown in this transcript, A11 is measured or scheduled per SPEC D6, every DEBT.md row is closed or kept with a stated reason, and two consecutive fresh hostile review passes over the changed code find nothing material.

Read first: that folder's README and the files it lists; then skills scenario-work-ladder and test-genie. Docs before code.
Role: you orchestrate. Delegate each WORKSTREAMS outcome to a sub-agent using the worker template in GOAL.md; run independent outcomes concurrently; serialize API restarts, migrations, proto generation and test-genie runs. You review and accept; workers implement.
Proof: per row, the command in SPEC §5, output shown. Targeted tests and scoped test-genie phases; no full suite unless SPEC names it.
Boundary: scenarios/agent-manager/**, resources/claude-code/**, packages/cli-core/cliutil/agentlauncher.go, agent-manager protos via the governed generator, notification-hub integration through its public contract. Scope policy: extend-with-record in WORKSTREAMS.
Quality: replace, don't wrap. No shims, dual paths or dead code; net non-test LOC in touched areas falls. A feature that leaves its DEBT rows open is not accepted.
Operator feedback: append verbatim to FEEDBACK.md before acting; not done while any item is open.
Adjacent defects outside the boundary: report-bug skill, continue.
Blocked means a decision, credential, or approval you lack. Name it. Friction you can diagnose is not blocked.
Keep going through friction; when progress stalls, step back — write why in WORKSTREAMS, then park, switch or shrink.
Budget: checkpoint WORKSTREAMS after every acceptance; if Codex usage status is will_exhaust, stop dispatching Codex workers except luna.
Report: changed, verified, remaining, unverified.
Stakes: quota exhaustion stays invisible and Mizar's core value unproven.
```

## Worker assignment template

The orchestrator fills every `<…>` and gives this as the sub-agent's task (and as
`/goal` when the runner supports native goals):

```text
/goal Work until this verified end state is true: <WS id> outcome "<outcome>" from scenarios/agent-manager/docs/efforts/usage-intelligence/WORKSTREAMS.md is implemented, closes DEBT rows <ids>, and satisfies SPEC acceptance <ids>, proven by <commands> with output shown.

Mission: Agent Manager answers "will I run out, what is using it, what should I change" with numbers that match the provider and disk (SPEC §1).
References: SPEC §<sections>, EVIDENCE <E-ids>, DEBT <D-ids>. Read them before code.
Ownership: <files/packages>. Peers: <live peers and their files>. Do not restart the API, migrate the DB, regenerate protos or start test-genie runs without the orchestrator's grant: <how to ask>.
Implementation: replace the superseded code named in the DEBT rows; no shims, no dual paths, no dead code; update the docs the change affects.
Verification: <focused tests>, <scoped test-genie phases>, <live comparison>. Tests assert desired behavior from SPEC, not current behavior.
Adjacent defects: fix when understood and blocking; otherwise record in WORKSTREAMS "Findings" and continue.
Blocked means a decision, credential, or approval you lack; name it.
Return: changed, verified (commands + results), remaining, unverified, DEBT before/after LOC.
```

## Launch

The operator's Claude Code session orchestrates (SPEC D1): it runs the goal above and
dispatches each workstream to a Claude Code sub-agent with the worker template.
Because the indexer (W2) captures Claude Code sessions and their sub-agent files,
this effort's own consumption becomes the first live test of the Usage page.
