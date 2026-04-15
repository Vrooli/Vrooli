# Run-Level Undo And Revert

## Initiative purpose

Provide a first-class way to back out run-linked changes once sandboxed runs auto-apply accepted changes by default.

## Why this exists

The workshop conversation settled on a default where accepted sandbox changes apply automatically even if the run fails. That is the right choice for auditability and workflow simplicity, but it increases the importance of a safe, provenance-based undo path afterward.

## Design assumptions captured from the conversation

- Undo should build on recorded provenance, not on destructive git history rewriting.
- Git Control Tower is the most natural first operator surface because that is where AI-originated changes and provenance are already reviewed.
- The system should preserve an audit trail for the revert itself, not just the original run.
- Partial overlap and mixed-provenance files are expected edge cases, not unusual failures.

## Relationship to earlier initiatives

- This initiative should not lead the sandbox rollout.
- It depends on auto-apply semantics existing and on committed provenance being visible or at least queryable.
- The design work should explicitly distinguish deterministic revert cases from cases that need human intervention.
