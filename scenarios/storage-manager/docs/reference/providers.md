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

## Model store disk retention

`model-store-retention` decides when downloaded AI models leave disk and which
ones. Storage Manager decides; each model store's owner deletes. The policy is
`config/model-retention-policy.json`:

| Setting | Shipped value | Meaning |
| --- | --- | --- |
| `check_interval` | `15m` | How often the controller decides again |
| `notify_below_free_percent` | `15` | Below this much free space on a store's filesystem, preview and notify |
| `evict_below_free_percent` | `10` | Below this, select eligible idle models for eviction |
| `evict_until_free_percent` | `15` | Select until projected free space reaches this |
| `idle_window` | `14d` | A model used or pulled more recently than this is held |
| `registry_tag_refetch_penalty` | `2` | Divides the score of a model whose refetch re-resolves a registry tag |
| `approval` | `operator` | `operator`: each selection is a Notification Hub decision. `automatic`: the controller applies selections itself |
| `stores` | `ollama`, `ollaya` | Store name and owner command |

Per-store budgets come from each resource manifest:
`resources/<store>/resource.json` `storage.entries.models.budget.max_bytes`.

**Facts.** Every store reports its view through
`<command> models retention --json` (schema `vrooli.model-store-retention/v1`):
each installed model's digest, total and unique bytes (blobs no other tag
references), pull time, whether it is loaded, whether its owner's policy
requires it, the reacquire command, and whether reacquisition reproduces the
same digest. Real use comes from `~/.vrooli/state/resources/<store>/model-usage.json`,
which the store's activity edge writes from actual requests (a keep-alive-0
unload is not use). Residency is not use. A model's idle clock is the latest of
its last request, its pull time, and the time usage tracking started.

**Held models.** A model is never a candidate when it is required (Ollama: a
role's primary or `retention.pinned` in `resources/ollama/model-policy.json`;
Ollaya: a default pull or a route target), loaded in memory, refused by its
owner (Ollaya: unpinned or digest drift from the pinned digest), missing a
digest, inside the idle window, sharing every blob with another tag, or when
the store's usage has never been recorded (unknown use fails closed).

**Selection.** Candidates rank eligible first, then by unique bytes divided by
the refetch penalty (1 for an exact digest refetch), then by idle time. The
decision first selects from each over-budget store until it is under budget,
then, on any filesystem below the evict band, until projected free space
reaches the evict-until level. A shortfall that eligible models cannot cover
is reported, not hidden.

**Approval and apply.** The controller in `internal/modeleviction` sends a
notice (once per situation per day) while a filesystem is in a band, a store
is over budget, or a store could not be observed. Each non-empty selection
becomes one Hub decision (`storage-manager.model-eviction.decision_requested.v1`,
subject key `model-eviction`) with a recommendation; a changed selection
updates that decision, an empty one withdraws it, and an answered selection is
not asked again. On approval, Storage Manager re-decides and evicts only the
approved models whose digest and size are unchanged and that are still
eligible, each through `<command> models evict <name> --digest <d> --size <n>`.
That owner operation takes the store's lifecycle lock and the per-model
request fence, refuses a model that is loaded (re-read at deletion time), has a
request in flight, changed digest or size, or became required, and appends the
reacquire command and expected size to
`~/.vrooli/state/resources/<store>/model-evictions.jsonl` before it deletes.
Through the cleanup plan API the provider applies only with operator approval;
pressure recovery never applies it. `GET /api/v1/cleanup/model-retention`
returns the current decision without applying anything.

## Owner-reclaim providers

A governed root that declares `owner_reclaim_command` in `storage.roots`
delegates removal to the tool that owns it. Each such root registers one
`OwnerReclaimProvider` under the root's id. It is `safe_with_owner`,
owner-budgeted, and owner-approved. Storage Manager plans and invokes; the
owner decides what is safe to remove. The owner binds its plan to the exact
store, enforces its own budget, and reports what it freed. Storage Manager
never walks or deletes the root itself.

The owner-reclaim/v1 contract adds two verbs to the declared command. Each
verb prints one JSON document whose `schema` ends in `/v1`:

| Invocation | Document fields Storage Manager reads |
|---|---|
| `<command> plan --cache <root> --json` | `cache`, `token`, `store_bytes`, `budget_bytes`, `reclaimable_bytes`, `blocked_reason` |
| `<command> reclaim --cache <root> --token <token> --max-bytes <n> [--time-budget <d>] --json` | `cache`, `freed_bytes`, `removed_entries`, `stop_reason`, `blocked_reason` |

Behaviour:

- **Blocked owner.** An owner that declines exits 0 and names its reason in
  `blocked_reason`. The provider reports that reason verbatim as its blocked
  reason.
- **Failed owner.** A command that cannot start or exits non-zero is also
  reported as blocked during estimate and preview, and as a skipped batch
  during apply. It never fails the recovery run.
- **Contract violations.** A document for another cache, a document with a
  missing or foreign schema, or a plan that proposes bytes without a token is
  an error.
- **Preview items.** Preview splits the reclaimable bytes into items of at most
  1 GiB, up to 256 items. Each item carries the plan token as its `identity`,
  so the recovery controller's bounded batches map onto bounded owner applies.
- **Apply.** Apply requires every item to name the same token. It passes the
  sum of the item bytes as `--max-bytes`, and the caller's remaining deadline,
  less a 5-second margin, as `--time-budget`.
- **Reclaimed bytes.** These are the owner's `freed_bytes`, never the estimate.

The schema confines `owner_reclaim_command` to tier `safe_with_owner`, and
forbids it together with `tool_prune_command`, so exactly one owner reclaims a
root. The Go build cache (`vrooli host go-cache`) is the first owner; see
[Shared Go build-cache safety](recovery-ledger.md#shared-go-build-cache-safety).
Tests substitute a scripted process runner; the provider package never runs a
real owner.
