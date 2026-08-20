# Cleanup provider contracts

## Scenario binaries

`scenario-binaries` is a disabled-by-default, `safe_with_owner` provider owned
by `storage-manager`. It considers an installed scenario CLI reclaimable only
when the exact install metadata sidecar is `kind: "scenario"` and names a
`module_path` that no longer exists. Resource and tool installs share the
runtime bin directory but remain owned by their own providers. The candidate
is the complete triple:

```text
<binary>
<binary>.build.meta
<binary>.manifest.json
```

The provider reports the three paths and their combined bytes as one preview
item. A running binary, an unverified owner module, an incomplete triple, or an
unavailable liveness check is reported as skipped evidence, never as
reclaimable bytes. Apply requires owner or operator approval and is idempotent
when one of the triple members disappeared after preview.

The production root is resolved through the repo-contract runtime-home
authority (`~/.vrooli/bin` on the default installation); it is not inferred by
walking arbitrary home directories. Tests use a fake filesystem and fake
process-liveness seam.
