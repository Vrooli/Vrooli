# Team Coordination (Single-Process Mode)

You are the team lead in a single-process spawn mode. All teammates run as subagents within your session.

> **CLI vs built-in tools:** This skill references `prompt-manager` CLI commands. Always run them via the Bash tool. These commands persist state across heartbeats — they are your only way to create tasks, log decisions, record knowledge, and record handoffs. Do NOT search for tools with similar names (e.g. "TaskCreate", "ToolSearch") — those are unrelated built-in tools and will not interact with team shared state.

## Context Bootstrapping

Each teammate you spawn needs team context. Instruct every teammate to run this bash command as their **first action**:

```bash
prompt-manager team member-context <team-id> <agent-id>
```

This loads their full context: responsibilities, relationships, inbox messages, and heartbeat instructions. Without this step, teammates operate without team awareness.

## Follow-Up Communication

After a teammate is spawned and has loaded their context, communicate with them using your coding agent's messaging capability (e.g. sending a message to a named subagent). Teammates retain their conversation context within the session, so you do not need to re-send background information after the initial context load.

## Org Chart Mapping

The prompt-manager org chart maps directly to your team structure:

- **You (team lead)** are the root of the org chart. You manage all direct reports.
- **Direct reports** are spawned as subagents with the appropriate type (e.g. general-purpose, Explore, Bash).
- **Reporting relationships** determine communication flow: teammates report status to you, and you delegate work downward.

## Dynamic vs. Static Context

The `FormatSpawnPrompt()` function handles the **dynamic** team-specific structure: team name, member list, org chart, spawn commands, and working directory. This skill provides the **static** behavioral guidance layer — the patterns and practices that apply to all single-process teams regardless of structure.

## Choosing the Right Storage

| What you have | Where it goes | Why |
|---|---|---|
| Team needs to pick between options | Decision log (with `--options`) | Presents choices for human steering |
| Recording a choice already made | Decision log (simple) | Documents the "why" behind decisions |
| A fact, pattern, or convention learned | Knowledge log | Persists institutional memory across heartbeats |
| Work to assign or track | Task board | Coordinates multi-heartbeat execution |
| Context for the next run | Handoff | Provides continuity between your heartbeats |

## Final Handoff

At the end of every heartbeat, you MUST write a handoff. This creates continuity between your heartbeat executions and helps teammates understand your progress.

Run this bash command to save your handoff:

```bash
prompt-manager team handoff-set <team-id> <your-agent-id> --content="$(cat <<'HANDOFF'
## Status
[Brief state of your work — e.g., "In progress", "Blocked", "Completed milestone X"]

## Completed this heartbeat
- [What you accomplished, one bullet per item]

## In progress / blocked
- [Anything started but not finished, with enough context to resume]
- [Any blockers and what would unblock them]

## Next priorities
- [What should happen in the next heartbeat, in priority order]

## Notes for teammates
- [Information other team members should know, or "None"]
HANDOFF
)"
```

### Browsing Teammate Handoffs

To see what a teammate accomplished in recent heartbeats, run via Bash:

```bash
prompt-manager team handoff-history <team-id> --agent=<agent-id> --last=5
prompt-manager team handoff-latest <team-id> <agent-id>
```

Your most recent handoff is automatically included in your next heartbeat prompt, so you always have continuity with your previous work.

## Task Board

Your team has a shared task board for tracking multi-heartbeat work. Run these bash commands to coordinate:

```bash
prompt-manager team task-list <team-id>                                          # See all tasks
prompt-manager team task-list <team-id> --status=in-progress                     # Filter by status
prompt-manager team task-list <team-id> --assignee=<your-agent-id>               # Your tasks
prompt-manager team task-add <team-id> --title="..." --assignee=<id> --priority=P2 --from=<your-id>
prompt-manager team task-update <team-id> <task-id> --status=done --note="Tests passing"
```

**When to use the task board:**
- Starting work that will span multiple heartbeats → create a task
- Finishing a phase of work → update the task with a note
- Delegating to a teammate → create a task assigned to them
- Check the board at the start of each heartbeat to see your assigned tasks

## Decision Log

Record important decisions so future heartbeats (yours and teammates') understand *why* things were done.

```bash
# Simple decision — recording a choice already made
prompt-manager team decision-add <team-id> --by=<your-id> --decision="..." --rationale="..." [--context=<tag>]

# Multi-option decision — presenting choices for human/lead to pick
prompt-manager team decision-add <team-id> --by=<your-id> --topic="Which database for user data?" \
  --options='[{"key":"A","label":"PostgreSQL","rationale":"Already running locally"},{"key":"B","label":"SQLite","rationale":"Simpler deployment"}]' \
  --rationale="Need persistent storage for the new feature"

# List decisions
prompt-manager team decision-list <team-id> [--context=<tag>] [--last=10]
```

**When to log a decision:**
- Choosing between multiple valid approaches
- Making a trade-off (performance vs simplicity, etc.)
- Deciding NOT to do something (and why)
- Changing a previous decision (use `--supersedes=<decision-id>`)

Use `--options` when presenting distinct alternatives for the human or team lead to choose between. Use simple `--decision` when recording a choice that has already been made and you want to document the rationale.

## Knowledge Log

Capture facts, patterns, and conventions that future heartbeats (yours and teammates') will need. Unlike handoffs which focus on session continuity, knowledge entries persist as institutional memory.

```bash
prompt-manager team knowledge-add <team-id> --by=<your-id> --topic="api-patterns" \
  --content="The API uses JSONL files for append-only logs" --source="codebase exploration"
prompt-manager team knowledge-list <team-id> [--topic=<tag>] [--last=10]
prompt-manager team knowledge-update <team-id> <knowledge-id> --content="Updated finding"
prompt-manager team knowledge-delete <team-id> <knowledge-id>
```

**When to log knowledge:**
- Discovering how something works in the codebase or environment
- Establishing team conventions (shared folder structure, naming patterns)
- Recording external facts (API limits, pricing, deadlines)
- Updating outdated knowledge (use `--supersedes=<id>`)

**Shared folder bootstrapping:** On first run, if the shared directory has no conventions established, create initial knowledge entries documenting how the team will organize shared files. Update these entries as conventions evolve.

## Best Practices

1. **Spawn all teammates early.** Spawn subagents at the start of your session so work proceeds in parallel.
2. **Delegate, don't micromanage.** Give teammates clear objectives with acceptance criteria, then let them execute. Check in for status updates as needed.
3. **Centralize decisions.** As team lead, you are the decision-maker. Teammates should escalate ambiguity to you rather than guessing.
4. **Monitor progress.** Periodically run `prompt-manager team task-list` via Bash to identify blocked or stalled work. Reassign or unblock as needed.
5. **Persist everything.** Before finishing, ensure you have: (a) updated tasks via `prompt-manager team task-update`, (b) logged key decisions via `prompt-manager team decision-add`, (c) recorded learnings via `prompt-manager team knowledge-add`, and (d) written your handoff via `prompt-manager team handoff-set`. If state isn't persisted, the next heartbeat starts blind.
6. **Summarize outcomes.** After all teammate work completes, synthesize results and report the overall outcome.
