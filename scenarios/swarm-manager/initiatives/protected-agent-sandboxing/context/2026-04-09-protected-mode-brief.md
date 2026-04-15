# Protected Agent Sandboxing

## Initiative purpose

Add a stronger protected-mode execution path without changing how coding agents are prompted or expecting them to become sandbox-aware.

## Design principles captured from the workshop conversation

- The orchestration layer should change how the process is launched, not how the agent thinks.
- The agent should ideally behave the same regardless of whether the run is sandboxed.
- Tracking and auditability come first. Protected mode is an enhancement on top of the same run-attribution model.
- Scenario CLIs and localhost-backed APIs are part of normal agent workflows and must continue to work in protected mode.
- Direct shell operations from the agent can be restricted more aggressively than scenario-mediated operations.

## Current-state findings

- Workspace-sandbox already exposes contained execution APIs for one-shot, long-running, and interactive commands.
- Agent-manager does not currently use those APIs to launch the coding agent process itself.
- Existing tool/path policy controls in agent-manager are partially wired and should not be mistaken for a real protection boundary.
- Network semantics and git restrictions need a more explicit contract than the current mixed documentation and runner-specific controls provide.

## Protected-mode target behavior

- Agent-manager launches the coding agent process through workspace-sandbox execution APIs.
- Direct git usage in the agent process tree is read-only by default.
- Side-effecting git actions are routed through trusted scenarios instead of direct shell commands.
- Runtime denials are human-readable and explicitly warn the agent not to work around the restriction.
- Path/resource/network policy is enforced at the sandbox/runtime layer where possible.

## Non-goals

- Do not block the auditability rollout on protected-mode perfection.
- Do not rely on prompt wording as the primary enforcement mechanism.
- Do not regress scenario CLI usability in pursuit of a cleaner security story.
