# Git Control Tower AI Provenance

## Initiative purpose

Make Git Control Tower the operator-facing surface that proves sandbox auditability is working.

The broader sandbox effort only succeeds if operators can clearly answer:

- Which agent-manager run changed these files?
- When did that run happen?
- Who initiated it?
- Which changes are still pending review or commit?
- Which committed changes in repo history came from which runs?

## Context from the workshop conversation

- The AI Changes tab is central to the product value of sandboxing.
- The point of pushing more runs through sandboxing is to associate git changes with specific runs and their context.
- Pending provenance is useful, but the long-term audit story should survive commit as well.
- UX clarity matters as much as data capture. If the UI is confusing or lossy, teams will not trust the attribution model.

## Current-state findings

- Git Control Tower already has a workspace-sandbox client and a pending provenance-by-run API path.
- The AI Changes tab already exists and groups changes by run.
- Review dimensions already include provenance/traced-vs-untraced concepts.
- Workspace-sandbox already stores committed metadata on applied changes, but Git Control Tower does not yet appear to surface committed provenance history in the same way it surfaces pending changes.

## What this initiative must protect

- Do not accidentally ship a provenance UX that implies stronger guarantees than the data really supports.
- Distinguish clearly between pending applied changes and committed historical attribution.
- Handle missing run ids, partial provenance, and mixed-provenance files gracefully.
- Keep the UI intuitive for operators who care about auditability, not just implementation details.

## Relationship to the foundation initiative

- This initiative depends on the sandbox lifecycle producing reliable applied-change provenance.
- It should move in parallel once auto-apply semantics are corrected, because default sandbox rollout depends on Git Control Tower being trustworthy.
