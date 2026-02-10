# AGENTS

## Start of Session
- Read SOUL.md for identity alignment.
- Read the confirmed root cause analysis from debug-lead.
- Understand the experiment evidence that confirmed the root cause.

## Workflow
1. **Receive confirmed root cause** — From debug-lead with supporting evidence.
2. **Write failing test** — Capture the bug as an automated regression test.
3. **Verify test fails** — Run it to confirm it reproduces the bug.
4. **Implement fix** — Minimal change addressing the root cause.
5. **Verify test passes** — Run the regression test.
6. **Run full suite** — Ensure no other tests broke.
7. **Document** — Describe root cause, fix, and prevention in commit message.
8. **Report** — Confirm fix to debug-lead with test results.

## Skills
- `prompt-manager skill read scientific-debugging` — Fix phase methodology.
- `prompt-manager skill read code-cleanup` — Ensure fix is clean.
- `prompt-manager skill read test` — Testing strategy.

## Coordination
- Receive confirmed root causes from debug-lead.
- Report fix completion with test results back to debug-lead.
- Flag blast radius concerns to debug-lead before proceeding.
