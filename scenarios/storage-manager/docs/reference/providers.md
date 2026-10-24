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

## Orphaned non-live instance storage

`orphaned-instance-storage` is a disabled-by-default, `safe_with_owner`
provider owned by `storage-manager`. It owns the storage a non-live scenario
instance leaves behind.

A scenario can run as more than one instance, addressed `<scenario>@<variant>`.
Every non-live instance gets its own directory, named `<scenario>_<variant>`,
under each storage class root — `~/.vrooli/data/vrooli/web-console_presentation`
and its siblings in `config`, `cache`, `logs`, and `state`. The lifecycle
creates them on first start; nothing removed them when the instance was stopped
for good. Before this provider, no owner existed for that storage and
`cleanup plan` did not see it at all.

### Attribution

A directory name is attributed by lookup, never by splitting on `_`. Scenario
slugs contain hyphens, variants may contain hyphens and digits, and the state
root additionally holds host safeguard directories such as
`agent_session_containment` that share the shape without belonging to any
scenario. A directory is a candidate only when its prefix **is** a known
scenario (longest match wins) and the remainder is a non-empty variant other
than `live`.

Three outcomes, all explicit:

| Directory | Outcome |
|---|---|
| Name is itself a scenario slug (`web-console`) | Live instance; never a candidate, never reported |
| `<known scenario>_<variant>` | Candidate, subject to the liveness and age checks below |
| Anything else, including `<scenario>_live` | Unattributable; never a candidate, counted in one summary warning so an operator can see the provider looked and declined |

Known limitation: attribution is prefix-based, so a future host safeguard state
directory named `<single-word-scenario>_<word>` would be read as an instance of
that scenario. Safeguard directories are snake_case and scenario slugs are
kebab-case, so only the handful of single-word scenarios are exposed, and no
such collision exists today. Corroborating against the process registry would
not help: the registry entry for a long-stopped instance is cleaned up while its
storage is not, which is precisely the case this provider exists to find.

### Liveness

Liveness comes from the control plane's process registry
(`~/.vrooli/processes/scenarios/<scenario>@<variant>/`), never from mtime: a
stopped instance and an idle running one are indistinguishable on disk. A
recorded process counts as alive only when its pid is also live. An unreadable
registry blocks the whole preview — "I could not look" must never read as
"nothing is running".

### Protection

The `data`, `config`, and `state` class roots are contract-protected, and the
orchestration boundary drops preview items inside a protected root. This
provider declares its namespace roots as `ProvenOrphanRoots`, the narrow
exemption for a provider that proves a *specific* entry has no owner — the same
case the contract's `bin` rationale already makes for `scenario-binaries`. The
exemption is strict containment: a declared root itself, or any ancestor of it,
is still filtered.

### Apply

Apply requires owner or operator approval and a configured quarantine root; it
refuses without either. It re-reads the registry and re-derives attribution from
the path before acting, so an instance that restarted between plan and apply is
skipped. Reclamation is a move into
`~/.vrooli/state/storage-manager/quarantine/instances`, and bytes are counted as
reclaimed only when a quarantined entry expires after 7 days. Only entries this
provider could have created are expired.

The provider declares Linux and macOS support. Tests substitute a fake
filesystem and a fake instance registry.
