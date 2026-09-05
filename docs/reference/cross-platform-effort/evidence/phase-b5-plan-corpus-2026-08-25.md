# Phase B5 — Plan corpus reconciliation evidence

Date: 2026-08-25

## Result

Plan Manager now treats legacy markdown as importable source input without
destroying the source, preserves prose and import provenance, and recognizes a
rendered mirror as canonical on subsequent reconciliations. Intake ordering is
deterministic, and legacy provenance that lacks a source path is backfilled at
the import boundary.

The archival policy is documented in
`scenarios/plan-manager/docs/plan-corpus-archival-policy.md`: completed plans
remain available for 365 days after terminal state, active or unresolved plans
are not archived, and reconcile never deletes a source file merely because it
was imported.

## Live evidence

After rebuilding and restarting the Plan Manager through its scenario
lifecycle, an authoritative Connect `ReconcilePlans` apply was run with all
source classes enabled (intake, runtime-home, docs, and repository sources).
The second apply returned:

- 35 `RECONCILE_ACTION_IMPORTED` results;
- 1,880 `RECONCILE_ACTION_ALREADY_CANONICAL` results;
- 97 `RECONCILE_ACTION_CONFLICT` results.

A subsequent dry run returned:

- 1,950 `RECONCILE_ACTION_ALREADY_CANONICAL` results;
- 97 `RECONCILE_ACTION_CONFLICT` results;
- zero `RECONCILE_ACTION_IMPORT_PLANNED` results;
- zero parse failures.

The 97 conflicts are existing duplicate-slug legacy and archived source files.
They are reported as guided remediation and remain untouched; automatic
reconcile does not guess whether those files should be retired or supersede a
canonical plan.

The root hygiene command reported no blocking failures. Its plan-only report
contains the same two explicit warnings: the automatic reconcile opportunity
and the guided invalid-source conflict set. This is an intentional bounded
outcome, not a claim that the entire historical corpus is conflict-free.

## Tests

Passed:

```text
(cd scenarios/plan-manager/api && go test ./internal/plans ./internal/planmodel)
```

The focused resolver tests cover source provenance, rendered-mirror
idempotence, malformed legacy markup import, and deterministic intake ordering.
