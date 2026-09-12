# Historical plan sources

Plan Manager owns implementation plans, execution state, and supporting artifact
placement. New authoring sources belong in its returned `plan_artifacts`
directory. Source-code comments should reference durable behavior docs rather
than requiring a copied implementation plan in the repository.

## Historical source files

The previous plans, baseline notes, and `web-console-unlosable-conversations`
authoring fragments are preserved byte-for-byte under the protected runtime-home
`plan_artifacts` entry, retaining their original filenames and heading anchors:

```text
docs-cleanup-20260907-followup/scenarios/web-console/docs/internal/plans/
```

Read live continuity plan state with:

```text
plan-manager plans get web-console-unlosable-conversations
```

Relocation does not complete, supersede, or abandon any plan. The persistent
session recovery hardening source and roles-and-handoffs baseline remain
historical context in that archive. Existing Plan Manager references now name
the preserved artifacts. The [preservation record](../../../../../docs/internal/PROGRESS.md#documentation-cleanup-follow-up--2026-09-07)
locates original hashes and recovery copies.

Before retiring any historical source, verify a retained copy and update its
callers under the [shared placement rules](../../../../../docs/internal/SEAMS.md#documentation-and-execution-artifacts).
