"""Plan a safe registry sweep without invoking write-effect bindings."""

plan = program_runtime.bindings.sweep(dry_run=True)
print({
    "result_rows": plan.count(),
})

# Live CLI evidence on 2026-08-12: result_rows=1168; the CLI summary reported
# eligible=632, skipped=536, provenance=operator.
