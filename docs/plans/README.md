# Plan sources and design context

Implementation plans live in Plan Manager. Read the integrations plan with:

```text
plan-manager plans get professional-integrations-monetization-ux
plan-manager plans render professional-integrations-monetization-ux
```

The plan's authored source records are retained outside the repository in its
protected Plan Manager artifact root. They are provenance and design context,
not additional project documentation:

- `plan-artifacts/professional-integrations-monetization-ux/source-artifacts/`

Read the structured plan first; use `plan-manager plans get` and its persisted
references when the source record is needed.

Design context also includes the cross-ramp delivery spine (preserved HTML)
and heartbeat prompt restructure (preserved HTML).
These are design records, not the current execution state.

The 63 redundant field and phase fragments were compared with Plan Manager
before retirement. Their originals remain in the
[preservation archive](../internal/PROGRESS.md#documentation-capture-preservation--2026-09-05).

## Preserved visual sources

Historical HTML and editable canvas sources live beneath the protected
control-plane runtime home (`~/.vrooli` for the invoking user).
These are dated evidence or design supplements; this relocation does not
change plan status or establish release readiness.

- `plan-artifacts/docs-html-progress-20260908/docs/plans/assets/cross-ramp-delivery-spine.html`
- `plan-artifacts/docs-html-progress-20260908/docs/plans/assets/heartbeat-prompt-restructure.html`

The originating installation must retain these artifacts with its durable backups.
Exact originals and SHA-256 hashes are in `backups/docs-html-progress-20260908/manifest.json`.
