# Receipt capture declarations

A scenario declares the typed operations it needs observed in
`.vrooli/vrooli-events/*.json`. The canonical schema is
`scenarios/vrooli-events/schemas/receipt-capture-declaration.schema.json`.

Each file uses `schemaVersion: 1` and supplies one or more policies. A policy
names the target scenario and Connect operation, the declared response type,
and descriptor paths that may be retained. Projection paths are explicit:
receipt capture never infers fields from a response body.

Reconcile is idempotent by `policyId`. Reapplying a declaration updates the
existing receipt projection rule and broadcasts a fresh policy snapshot; it
does not create a second matching rule. Missing declarations remain benign:
they produce no receipts and do not change target request behavior.
