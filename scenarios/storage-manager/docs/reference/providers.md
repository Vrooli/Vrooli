# Cleanup provider contracts

## Abandoned undeclared workloads

`undeclared-workload` is a conditional, disabled-by-default provider for
operator-approved disposal proposals. It consumes the control-plane workload
classification and previews only `abandoned` findings; declared workloads and
unmanaged workloads are never candidates. Under `vrooli_only`, the abandoned
evidence must name a manifest, scenario, resource, or historical Vrooli path.

The preview records the exact native action and evidence line. Depending on
the workload kind, apply uses Docker, the user service manager, Windows Task
Scheduler, or the platform's binary-removal command. Apply requires operator
approval and an idempotency key, is braked while host pressure is saturated,
and is followed by native-registry verification. The provider does not infer
ownership from names and does not run as part of an automatic retention sweep.

The provider declares Linux, macOS, and Windows support. Tests substitute a
fake process runner and classification resolver; no live disposal is performed
by the provider test suite.

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
