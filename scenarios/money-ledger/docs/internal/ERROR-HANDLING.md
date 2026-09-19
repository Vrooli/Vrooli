# Error handling

Money Ledger follows an honesty rule: unavailable is a value, not an empty successful result. A failed adapter returns a receipt with `status=failed`, a named reason, and its last successful timestamp. It never emits a zero-valued event and never silently advances a sync window.

Every returned financial figure is qualified by its basis and the inputs that produced it. A partial position remains partial in proto, CLI JSON, and the operator console. An empty ledger says position is undefined; it does not display zero runway.

Input and lifecycle errors are invalid-argument responses naming the violated rule. Duplicate adapter/external ids return the original posting with `duplicate=true`. Corrections are reversing entries; no endpoint or UI affordance edits or deletes a posting. Unexpected database and transport failures use internal/unavailable errors and preserve the caller's ability to retry.
