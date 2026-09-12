# Activation and Reconciliation

> [CODE: api/execplan/compile.go] · [CODE: api/vps/argv.go] · [CODE: api/vps/execute.go] · [CODE: api/reconcile/reconcile.go] · [CODE: api/handlers_reconcile.go] · [CODE: ../../internal/cloudtarget/activate.go] · [CODE: ../../internal/cloudtarget/persistent_data.go]

A release reaches a target as an immutable, verified artifact set; it becomes
the active release through the target-local owner (`vrooli cloud-target`)
in a recoverable order; persistent data lives outside every release tree;
and every later lifecycle entry point (deploy, start, stop, repair, resume,
retire) is one reconciliation model: desired state versus observed state,
corrected only through the executable plan and the durable operation owner.

## 1. Execution model

Every effectful plan action is one typed target verb `{verb, argv[]}` built
by `vps.ActionCommands` from the action's inputs and the operation identity
(`--deployment --operation --step --fence`). The executor dispatches it
through `reach` on the deployment's bound transport; nothing composes a
shell string, and the shell preview (`preview.shell_preview[]`) is rendered
from the same argv after the fact. JSON-shaped verb inputs travel as
`b64:<base64url JSON>` arguments; a credential value travels only on the
ingest verb's standard input. Artifact bytes (bundle, release manifest,
native control plane, the autoheal scope document) travel only through
`reach.Deliver`.

| Action | Target verb |
|---|---|
| `host.prepare` | `cloud-target host repair --action apt.packages.ensure` |
| `edge.firewall.allow` | `cloud-target host repair --action edge.ufw.allow` (one receipt per port) |
| `data.inventory` | `cloud-target data inventory` (read) |
| `release.deliver` | `reach.Deliver` (scp on explicit SSH, or Bridge ArtifactsService with durable placement confirmation) then negotiated platform check |
| `release.verify` | `cloud-target release verify` (read) |
| `release.stage` | `cloud-target release stage` |
| `config.apply` | `vrooli setup --yes yes --environment production` + autoheal scope delivered as a file |
| `credentials.provision` | credential authority lifecycle (`credential ingest`, value on stdin) |
| `runtime.start_dependencies` | `vrooli resource start <id>`, `vrooli scenario start <id>` |
| `data.backup` | `cloud-target data backup` (recovery point recorded on the cloud side) |
| `workload.stop` | `cloud-target host repair --action process.stop.scoped` |
| `release.activate` | `cloud-target release activate --release --strategy --port --data-binding --legacy-carry --legacy-root` |
| `edge.route.apply` | `cloud-target edge route-apply --spec b64:…` |
| `verify.readiness` | `cloud-target release list` (active pointer, no interrupted intent) + public probes |
| `release.retain_predecessor` | `cloud-target release list` (previous pointer present; rollback eligibility recorded) |
| `workload.start` | `cloud-target release list` → `release activate --restart` of the active pointer |
| `edge.route.retire` | `cloud-target edge route-rollback` |
| `grants.revoke` | credential authority revoke |
| `data.retire` | no target owner yet: `retain` is a no-op, `delete` is refused |
| `artifacts.retire` | `cloud-target release list` (reports unreferenced releases; prune owner pending) |

The target refuses any fence below the highest it accepted and replays the
receipt of an `(operation, step)` it already completed, so re-running an
action is a replay of the invocation, never of a completed effect. A failed
receipt is replayed as failed; the recovery is a new operation.

## 2. Release identity on the target

The plan's `release_digest` is `sha256:<bundle sha256>` (the identity the
health observation reports). The target owner stages and activates by the
canonical release id from `release-manifest.json` (`inputs.release_id`,
64-hex, `packages/cloudrelease`). A bundle without a manifest beside it has
no release id: the plan compiles and previews the refusal
(`release_verification_failed`, reason `release_manifest_missing`) and
`release.verify` refuses before any target write. Deployments build releases
through `api/release` (`deployment.ReleaseBuilder`); the mini-bundle builder
without a manifest is no longer a deploy input.

## 3. Activation strategy and downtime

`execplan.SelectStrategy` chooses per closure and manifest and the choice is
material (it changes the plan digest):

| Strategy | When | Preview |
|---|---|---|
| `maintenance` | persistent data or legacy preserve paths declared, or the manifest pins listener ports | `workload.stop` declares `downtime.expected_seconds` (policy default 60) and the presentation states the bound |
| `side_by_side` | no persistent data and no pinned ports (the lifecycle owner allocates candidate ports) | no downtime declared |

Limitation: the target activator delegates candidate allocation to the
lifecycle owner's `vrooli scenario restart --path <release>/scenarios/<id>`;
a candidate does not yet run beside the current release, so `side_by_side`
is a declared intent without a concurrent candidate. Every production
manifest pins ports, so production activations are `maintenance`.

Listener ports are pinned as `<NAME>_PORT` environment for the lifecycle
owner (`--port ui=3000`), never as argv to the scenario.

## 4. Recoverable activation ordering

```
stage (releases/<id>, .complete)
  → data.backup (recovery point at the current schema; precondition
    recovery_point_required=<release>@<schema> on schema-changing updates)
  → workload.stop (maintenance only; scoped lifecycle stop)
  → release.activate:
      write activation-intent.json
      bind persistent data (section 6)
      runtime switch through the lifecycle owner (health-gated restart)
      commit active-release.json {active_release, previous_release, strategy, operation, fence}
      clear the intent
  → edge.route.apply (route switch only after a committed pointer)
  → verify.readiness (pointer == plan release id, no interrupted intent, public path)
  → release.retain_predecessor
```

A crash between the runtime switch and the pointer commit leaves the prior
pointer and `activation-intent.json`; `release list` reports it as
`interrupted_activation`. Reconciliation (section 7) reports `blocked` with
correction `resolve_interrupted_activation` and the runtime plan re-invokes
activation under a new operation; the successor ends with the candidate
active, the predecessor retained and exactly one pointer commit
(`api/vps/execute_test.go: TestActivationOrderingAndCrashRecovery`).

## 5. Predecessor retention and rollback eligibility

The active pointer retains one predecessor. `release rollback --to` accepts
only that digest (`rollback_not_eligible` otherwise). Eligibility is
explicit in the plan and the preview: `release.retain_predecessor` carries
`rollback_eligible` and `rollback_reason` derived from the closure's
recovery declaration (`code_rollback`, `schema_strategy`), mirroring
`backup.EvaluateRollback`:

| Reason | Eligible |
|---|---|
| `no_predecessor` | no |
| `code_rollback_not_declared` | no |
| `no_schema` (no persistent data, or `schema_strategy: none`) | yes |
| `schema_strategy_undeclared` | no |
| `explicit_restore_required` | no (restore a recovery point, then roll back) |
| `predecessor_retained` (`expand_contract`) | yes while the expand phase keeps the data readable |

## 6. Persistent data and legacy conversion

Persistent data lives beneath `<runtime-home>/cloud/deployments/<id>/persistent-data/`:
`persistent-data/<binding id>/` for declared bindings, `persistent-data/legacy/<scenario>/<path>/`
for heuristic carries. Activation binds each path inside the candidate
release tree with a symlink. The first activation that finds data at the
same relative path in the predecessor release or beneath the legacy root
(`--legacy-root <workdir>`, the in-place layout of converted deployments)
adopts it by rename. Nothing is copied and nothing is deleted; content the
release ships at a bound path is moved aside inside the release
(`<path>.shipped`). A heuristic-matched directory nobody mapped is reported
as `legacy_unmapped` in the receipt and left where it is.

Conversion of an existing deployment:

1. `POST /deployments/{id}/data-bindings/inventory` (read-only) lists the
   target's mutable directories against declared and recorded bindings.
2. `POST /deployments/{id}/data-bindings/adopt {mappings:[{binding_id, scenario, path}]}`
   records the mapping on the deployment (`persistent_data`). Only declared
   binding ids are accepted; no data moves.
3. The next plan binds recorded mappings first (`release.activate` input
   `data_bindings`), carries only the unmapped preserve paths
   (`legacy_carry`), and the target adopts them at activation. The
   directory-name heuristic therefore applies only to unmapped paths, and
   `data.backup` covers those legacy paths as bindings until they are
   declared or retired.

## 7. Reconciliation model

`api/reconcile.Detect(desired, observed, supervision, now)`:

- **Desired**: the deployment record — `desired_state` (`running` |
  `stopped` | `retired`), `desired_revision` (the fence), release,
  configuration and closure digests, `recorded_at`.
- **Observed**: the typed health observation (`observed_release_digest`,
  `observed_configuration_digest`, `status`, `freshness`, `observed_at`)
  plus the target owner's `release list` (`activation_interrupted`).
- **Outcome**: `unchanged` | `changed` | `blocked` | `unknown`.
- **Correction** (proposal only): `apply_release` (runtime scope),
  `restart_workload` (start scope), `stop_workload` (stop scope),
  `resolve_interrupted_activation` (runtime scope), `observe_first`.

Rules: `desired_state: stopped` is not drift and never yields a start; the
only correction for a stopped deployment is a stop when the workload is
observed running. `retired` is unchanged. A stale or unknown observation is
`unknown` with `observe_first`. An interrupted activation is `blocked`. A
release or configuration mismatch is `changed`. Unhealthy at the desired
release is `changed` with a restart. Otherwise `unchanged`.

`POST /deployments/{id}/reconcile` returns the report and the compiled
correction plan; resubmitting its `plan_digest` admits that plan through
the operation owner (`plan_digest_mismatch` otherwise). Observation never
mutates the target.

### Desired-state contract for observers (vrooli-autoheal)

`GET /deployments/{id}/desired-state` returns `desired_state`,
`desired_revision`, `reboot_policy` and `observation_may_restart`. An
observer (health, drift, autoheal) that finds a cloud workload down MUST
read this first: it may propose or perform a restart only when
`observation_may_restart` is true (desired `running` and the closure's
supervision declares `auto_restart`). A `stopped` or `retired` intent is
never repaired into a running workload; a `running` intent without declared
auto-restart waits for an operator start (`POST /deployments/{id}/start`).
The autoheal scope document delivered at `config.apply`
(`<workdir>/.vrooli/cloud/autoheal-scope.json`) names the deployment's
scenario and dependencies and carries `desired_state: runtime-owner`, which
means "ask the cloud owner, do not infer". vrooli-autoheal currently
restarts no cloud workload of its own accord (its scenario restart actions
are gated by the runtime recovery gate on the host that runs it), so no
seam was added there; this document is the contract it must honour if that
changes.

### Reboot policy

`reconcile.Reboot(desired_state, supervision)`: `leave_stopped` for a
stopped or retired intent; `supervisor_restarts` when the closure declares
`auto_restart` (with `startup_policy` and the declaration source);
`operator_start_required` otherwise. Nothing restarts a workload outside a
declaration.

## 8. Retirement

`POST /deployments/{id}/retire/plan {retention_policy}` lists retained and
deleted owned objects in owner order and compiles the `retire` scope plan:

| Order | Object | Disposition |
|---|---|---|
| 1 | route (`edge.route.retire`) | deleted first so no traffic reaches a retiring workload |
| 2 | runtime (`workload.stop`) | scoped stop; shared resources other deployments demand keep running |
| 3 | grants (`grants.revoke`) | revoked through the credential authority |
| 4 | data (`data.retire`) | `retain` keeps every binding in place; `delete` is irreversible and requires the explicit policy — and, until a target data-retire owner exists, is refused (`unsupported_capability`) rather than silently retained |
| 5 | artifacts (`artifacts.retire`) | active, previous and recovery-referenced releases and every recovery point are retained; unreferenced staged releases are reported for the prune owner |

Without `retention_policy` a deployment that holds data is `blocked`; apply
refuses a blocked plan. `POST /retire/apply {retention_policy, plan_digest,
request_key}` admits the reviewed plan and records `desired_state: retired`.

## 9. Protected cleanup

Cleanup protection is reference-based, never age- or lease-based alone
(`reconcile.Protect`): the active pointer, the rollback predecessor, every
release a recovery point references, and every recovery point that a
retained release or a non-terminal operation holds. Bundle retention on the
target (`deployment.enforceVPSBundleRetentionBestEffort`) merges the
deployment bundle, lease-protected releases and recovery-point releases into
the protected set; recovery-point retention (`backup.Plan`) never deletes a
protected point. A lease that expired while its owner crashed does not
expose a referenced artifact.

Bundles delivered to the target live under `<workdir>/.vrooli/cloud/bundles`
and releases under `releases/<id>`; repeated updates accumulate them. When
install-scope preflight fails on low disk the pipeline runs one bundle GC
pass with the default keep count (`bundle.DefaultVPSBundleKeepLatest`) and
re-runs preflight once. The operator surface is `GET /deployments/{id}/bundles/vps`
and `POST /deployments/{id}/bundles/vps/gc`, or
`scenario-to-cloud bundle vps-list|vps-gc <selector>`; both honour the
protected set above.

## 10. Shared dependencies across deployments

Two deployments on one host demand the same resource through the lifecycle
owner's dependency demand leases. `workload.stop` is a scoped stop of one
scenario: it releases only that deployment's demand, so a resource the other
deployment still needs keeps running; `resource stop` is refused while
demand remains. Retiring the subject removes only its own route.

## 11. Management actions and preflight fixes

`POST /deployments/{id}/actions/vps`: `stop_vrooli` runs scoped stops and
resource stops through the owner and records `desired_state: stopped`;
`cleanup` level 3 runs the broker's Docker prune actions; `reboot` and the
in-place delete levels are refused with the owner that must exist first
(`privilegebroker host.reboot`, the retirement plan). `POST /preflight/fix/firewall`
and `/preflight/fix/stop-processes` run `edge.ufw.allow`,
`process.stop.scoped` and the top-level `vrooli stop` through the bounded
SSH adapter over an explicit connection; there is no process-pattern kill
and no port kill anywhere on the cloud side.

## 12. Limitations (2026-09-10)

- Bridge artifact delivery requires the Bridge ArtifactsService and its
  device-sync-hub/node placement configuration; missing configuration or an
  unconfirmed placement is a typed transport failure and never triggers SSH
  fallback. Live disposable-target qualification remains pending.
- `side_by_side` runs no concurrent candidate (section 3).
- `data.retire delete` and `artifacts.retire` deletion have no target owner
  verb; both are honest refusals or reports.
- Health reports the bundle digest while the target owner reports the
  canonical release id; readiness compares the target pointer with the plan's
  release id and `no_op` detection still keys on the bundle digest.
- `/preflight/fix/ports` and the legacy `/preflight/disk/*` routes are retired.
  Disk capacity remains a read-only `df`/`du` observation in preflight; target
  repairs are owned by typed privilege-broker actions and bundle-GC operations.
