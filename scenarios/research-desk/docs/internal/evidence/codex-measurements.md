# Codex measurements behind the launch and control design

Measured on Codex 0.167 through Web Console sessions, 2026-11-11 and 2026-11-12. The raw
session notes are in [`launch-proof.txt`](launch-proof.txt). Re-measure after a Codex upgrade
with `research-desk launch-test`.

- **Keys in one burst are a paste.** Codex takes keys that arrive together as a paste, so an
  Enter right behind typed text inserts a newline. The launcher waits one second before each
  Enter, reads the input row back afterwards, presses Enter once more if the text is still
  there, and fails the step if it stays (first launch test).
- **`/goal pause` and `/goal resume` take no kept slot.** Codex records each as a goal-status
  item. Compaction drops those items and goal continuations, and keeps the stance message.
- **They act during a running turn.** Typed with Enter while a turn runs, each acts within a
  second and is not queued as a chat message. The running turn finishes. After a pause, no
  goal turn follows. After a resume, a goal-continuation turn starts when the running turn
  ends (session 3).
- **Re-sending the stance takes a kept slot.** Retention looks budget-based, not "the newest
  eight": one real session (2cf9ccfd-8dbf-408e-bd7d-9cb046589f6c) kept all 24 typed operator
  messages across seven compactions. The charter's stance limit is unchanged.
- **Both commands apply to a goal in any state.** Resuming a completed goal starts it again.
  So the desk pauses only an active goal and resumes only a paused one.
- **A goal that never completes loops.** In the design probe, a goal whose work was already
  done started about 90 tasks in two minutes. The launch test therefore uses a kernel that
  completes at once.
