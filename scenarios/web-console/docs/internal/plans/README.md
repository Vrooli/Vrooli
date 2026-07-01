# internal/plans

Archive directory for implementation plans that are referenced from source code (`// docs/internal/plans/...` comments) so the reference stays stable while the plan is still live.

This directory is **not** a manifest section — files here are not surfaced in the docs UI. The docs validator whitelists `docs/internal/plans/*` as non-orphan.

**Authoring new plans**: do not create files here. Use Plan Manager (`plan-manager author start/continue/finalize`) to create the canonical structured plan and rendered mirror. Only copy a plan here if a code comment must point at a stable archived file.

**Removing entries**: when every checklist item in a plan is satisfied, delete the file from this directory and remove the `// DOC: docs/internal/plans/...` comment from the source that referenced it.
