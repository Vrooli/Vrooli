# internal/plans

Archive directory for implementation plans that are referenced from source code (`// docs/internal/plans/...` comments) so the reference stays stable while the plan is still live.

This directory is **not** a manifest section — files here are not surfaced in the docs UI. The docs validator whitelists `docs/internal/plans/*` as non-orphan.

**Authoring new plans**: do not create files here. Use `vrooli plans add --stdin` to register new implementation plans in the canonical location. Only copy a plan here if a code comment must point at it.

**Removing entries**: when every checklist item in a plan is satisfied, delete the file from this directory and remove the `// DOC: docs/internal/plans/...` comment from the source that referenced it.
