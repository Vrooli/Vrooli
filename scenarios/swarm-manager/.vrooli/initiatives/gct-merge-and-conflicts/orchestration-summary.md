# Meta-Orchestrator Summary: GCT Merge & Conflict Resolution

## Source
Planning conversation covering Git Control Tower enhancement roadmap. This initiative adds local merge operations with three-way conflict resolution.

## Decisions Made
- Three API endpoints: merge (branch into current), status (conflict details with ours/theirs/base per file), abort
- Conflict resolution uses Monaco editor's three-way merge capabilities (needs evaluation during research)
- Per-conflict actions: accept ours, accept theirs, manual edit
- After all conflicts resolved, transitions to merge commit composition using existing CommitPanel
- Branch protection is configurable per branch pattern (e.g., `main`, `master`)
- Protection can require review panel readiness (green) before merge — local equivalent of GitHub branch protection
- Soft block with override for emergencies

## Dependency Notes
- This initiative is independent — no dependencies on other GCT enhancement initiatives
- Existing DiffViewer components should be heavily reused
- The merge-time checks item builds on the conflict resolution UI being complete
- Review readiness gating reuses the existing review panel infrastructure

## Unresolved Questions Deferred To Workshop
- Monaco editor's exact capabilities for three-way merge rendering (may need custom implementation)
- Conflict data model: how to structure ours/theirs/base content for the UI (structured hunks vs raw content)
- How merge commit messages should be composed (auto-generated vs user-edited)
- Whether to support rebase as an alternative to merge (deferred, not in scope for v1)
