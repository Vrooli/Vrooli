# Receipt capture declarations

A scenario declares the typed operations it needs observed in
`.vrooli/vrooli-events/*.json`. The canonical schema is
`scenarios/vrooli-events/schemas/receipt-capture-declaration.schema.json`.

Each file uses `schemaVersion: 1` and supplies one or more policies. A policy
names the target scenario and Connect operation, the declared response type,
and descriptor paths that may be retained. Projection paths are explicit:
receipt capture never infers fields from a response body.

Policies may additionally declare `workReferenceProjections`. Each entry maps
canonical response paths such as `plan.kind` and `plan.id` to a relationship;
revision, verification, visibility, and evidence digest paths are optional. The
mapping is generic and does not name Plan Manager or another work system. If a
declared field is absent, the receipt carries a `PROJECTION_MISMATCH` state with
a reason instead of inventing an identifier.

Reconcile is idempotent by `policyId`. Reapplying a declaration updates the
existing receipt projection rule and broadcasts a fresh policy snapshot; it
does not create a second matching rule. Missing declarations remain benign:
they produce no receipts and do not change target request behavior.
