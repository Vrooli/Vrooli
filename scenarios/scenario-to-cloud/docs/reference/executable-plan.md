# Executable Plan

> [CODE: api/execplan/plan.go] · [CODE: api/execplan/compile.go] · [CODE: api/vps/plan.go] · [CODE: api/vps/execute.go] · [CODE: packages/proto/schemas/scenario-to-cloud/v1/plans/plans.proto]

One typed action graph is compiled from desired state, the declared closure
and target observations. Preview renders it; policy checks it; apply admits
it by the exact semantic digest the operator reviewed; execution iterates
its actions. There is no second, independently constructed step list.

## Schema version rules

- `schema_version` is `"1"`. A field that changes meaning bumps it; additive
  optional fields keep it.
- Clients that receive an unknown `schema_version` MUST treat the plan as
  unsupported rather than reading fields positionally.
- JSON uses proto field names (snake_case). The Connect `PlansService` and the
  REST endpoints carry the same shapes.

## Envelope

| Field | Meaning |
|---|---|
| `deployment_id`, `scenario_id`, `environment` | Deployment identity (see identity-and-selectors.md). Ad hoc plans use `adhoc:<host>:<scenario>`. |
| `target` | `{machine_id, node_id, enrollment_generation, transport}`; the locator (host, port, user) is reachability metadata and is not part of the plan. |
| `scope` | `full` (install then runtime), `install`, `runtime`, `start`. |
| `outcome` | `apply`, `no_op` or `needs_input`. Material. |
| `desired_revision` | The revision the plan moves the deployment to. |
| `release_digest`, `configuration_digest`, `closure_digest` | The artifact, configuration and closure identities. Until the release domain binds a canonical release digest, `release_digest` is `sha256:<bundle_sha256>`; a runtime-only plan without a known bundle binds `manifest:<configuration digest>`, and a preview compiled before the bundle exists binds `pending:<hash of path>`, which `release.verify` cannot satisfy at apply. |
| `policy_version` | Policy the plan was compiled under (default `cloud-launch-v1`). |
| `preconditions[]` | Material facts: `deployment_revision`, `target_enrollment`, `release_digest`, `closure_digest`, `configuration_digest`, `data_schema`, `privilege_set`. |
| `actions[]` | The action graph (below). |
| `handoff` | Present only for `needs_input`. |
| `presentation` | Human text. Excluded from the digest. |

## Semantic digest

`sha256:` + SHA-256 over the canonical JSON of the envelope with
`presentation` zeroed (struct field order, sorted map keys, `[]` for empty
lists). Properties enforced by `api/execplan/execplan_test.go`:

- Equal material inputs give one digest (map ordering, JSON round trips).
- Presentation-only edits keep the digest.
- A changed target, artifact, closure, configuration, data schema, privilege
  set or destructive effect (maintenance strategy) changes it, so a prior
  review cannot authorise the new plan.

## Action vocabulary

Every action has `id`, `owner_operation`, `effect`, `required_capability`,
typed `inputs` (never a shell string), `depends_on`, `verification`,
`recovery`, `retry` (`safe_replay` | `observe_then_replay` | `recover`),
`cancel_point` and optional `downtime`. Ids equal the owner operation
because each appears at most once per plan.

| Owner operation | Effect | Owner / capability | Inputs | Scope |
|---|---|---|---|---|
| `host.prepare` | host_write | `cloud-target host repair --action apt.packages.ensure` | packages, workdir, bundle_dir | install |
| `edge.firewall.allow` | edge_write | `cloud-target host repair --action edge.ufw.allow` | ports `80,443`, protocol | install (Caddy enabled) |
| `data.inventory` | none | `cloud-target data inventory` | workdir, scenarios, bindings, data_bindings, legacy_preserve, mutable_names | install |
| `release.deliver` | deployment_write | reach `Deliver` (bundle, release manifest, native CLI) | artifact_ref, release_id, bundle_sha256, artifact_path, destination, release_manifest, native_cli | install |
| `release.verify` | none | `cloud-target release verify` | release_digest, release_id, bundle_sha256, archive, release_manifest | install |
| `release.stage` | deployment_write | `cloud-target release stage` | release_digest, release_id, archive, release_manifest, release_dir | install |
| `config.apply` | host_write | `vrooli setup` + autoheal scope delivered as a file | workdir, environment, selection, scenario_id, resources, scenarios, autoheal, autoheal_scope_path | install |
| `credentials.provision` | credential_write | credential authority (ingest verb, value on stdin) | workdir, scenario, descriptors (references only, never values) | runtime, start (when secrets declared) |
| `runtime.start_dependencies` | runtime_start | `vrooli resource start`, `vrooli scenario start` | workdir, resources, scenarios | runtime, start |
| `data.backup` | data_write | `cloud-target data backup` | bindings, binding_specs, legacy_preserve, recovery_key_ref, schema_version, configuration_digest, release_digest, migration_posture, retention_policy, recovery_point_required | runtime, when data (declared or legacy) exists |
| `workload.stop` | runtime_stop | `cloud-target host repair --action process.stop.scoped` | workdir, scenario, ports; declares downtime under `maintenance` | runtime (maintenance), stop, retire |
| `release.activate` | deployment_write | `cloud-target release activate` | release_digest, release_id, release_dir, scenarios, strategy, ports, data_bindings, legacy_carry, legacy_root, predecessor_digest | runtime |
| `edge.route.apply` | edge_write | `cloud-target edge route-apply` (typed spec) | domain, upstream_port, tls_email, tls_enabled, config_path, release_id | runtime |
| `verify.readiness` | none | `cloud-target release list` + public probes | checks `local[,https,origin,public]`, domain, ui_port, host, release_id | runtime, start |
| `release.retain_predecessor` | none | `cloud-target release list` | predecessor_digest, release_id, rollback_eligible, rollback_reason | runtime, when a predecessor is observed |
| `workload.start` | runtime_start | `cloud-target release activate --restart` of the active pointer | workdir, scenario, scenarios, ports, ui_port, release_id, data_bindings | start |
| `edge.route.retire` | edge_write | `cloud-target edge route-rollback` | domain | retire |
| `grants.revoke` | credential_write | credential authority revoke | scenario, descriptors | retire |
| `data.retire` | none (`retain`) / data_write (`delete`) | no target owner yet | bindings, legacy_preserve, retention_policy | retire |
| `artifacts.retire` | deployment_write | `cloud-target release list` (report) | release_id, protected | retire |
| `input.resume_handoff` | none | vrooli-onboarding | owner, kind, reference, missing | needs_input only |

Scopes: `install` (host, deliver, verify, stage, setup), `runtime`
(credentials, dependencies, recovery point, maintenance stop, activate, route,
readiness, retain), `full` = install then runtime, `start` (dependencies,
restart of the active pointer, readiness), `stop` (one scoped stop),
`retire` (route, runtime, grants, data, artifacts; `retention_policy`
required when data exists).

Activation strategies are selected by `SelectStrategy` (material):
`maintenance` when persistent data, legacy preserve paths or pinned ports
exist (`workload.stop` declares the policy's downtime bound, default 60 s);
`side_by_side` otherwise. See
[activation-and-reconciliation.md](activation-and-reconciliation.md).

## Outcomes

- **apply**: the action graph above.
- **no_op**: the observed active release, configuration and closure digests
  all equal the desired ones. An unobserved target (empty digest) is never a
  no-op. Apply returns `state: "no_op"` and admits no operation.
- **needs_input**: the closure's required credential descriptors or the
  manifest's required `user_prompt` secrets are not satisfied. The plan
  carries exactly one handoff `{owner: "vrooli-onboarding", kind:
  "resume_handoff", reference, missing[]}` and one effect-free
  `input.resume_handoff` action. Apply refuses with `needs_input` (HTTP 428)
  carrying the same reference as `next_action`.

## Preview, policy, apply

| Endpoint | Behaviour |
|---|---|
| `POST /api/v1/deployments/{id}/plan` (body `{scope?}`) | Compile + preview. Creates no records and no target effects. Returns `{schema_version, plan, plan_digest, preview, steps, closure_status}`. |
| `POST /api/v1/deployments/{id}/plan/apply` (body `{plan_digest, request_key, scope?, plan?}`) | Recompiles, refuses `plan_digest_mismatch` (409) on a different digest, `plan_stale` (409, with `details.reasons`) when the submitted reviewed `plan` shows a material precondition changed, `needs_input` (428), returns `no_op` without a record, otherwise admits `cloud_operations(deployment_id, request_key, plan_digest, plan)` and returns `{operation_id, plan_digest, state: "admitted"}` (202). The same key and digest replay the admitted operation; a different digest is `request_key_conflict`. |
| `POST /api/v1/vps/setup/plan`, `/vps/deploy/plan` | Compile the install / runtime scope from a manifest (ad hoc). Response keeps the legacy `plan[]` step view (id = action id, `command` = shell preview) and adds `plan_digest`, `executable_plan`, `preview`. |
| `POST /api/v1/vps/setup/apply`, `/vps/deploy/apply` | Require `plan_digest`; recompile and refuse on mismatch or `needs_input` before touching the target. Optional `deployment_id` + `request_key` admit a durable operation. Results carry `plan_digest`, `actions[]` receipts keyed by action id and `failed_step` = the failing action id. |
| Connect `PlansService.CompilePlan` / `ApplyPlan` | Same implementation, proto shapes from `plans.proto`. |

`preview.shell_preview[]` is a derived convenience rendered by
`vps.ShellRenderer`; execution never consumes it.

## Execution

`vps.ExecutePlan(ctx, ExecuteRequest{Plan, Manifest, BundlePath, Runtime{Reach, Target, Identity{OperationID, Fence}, Credentials, Backups}, Hooks})`
iterates the plan's actions in order, checks `depends_on`, derives each
action's typed target verbs with `vps.ActionCommands` and dispatches them
through `reach` on the deployment's bound transport with the operation id
and fence. Every reach call (exec, deliver, negotiate) is attributed to the
action in flight; a call outside any action fails with `undeclared effect`.
No shell string is composed: JSON inputs travel as `b64:` arguments and
credential values on the ingest verb's standard input. The durable operation
worker owns execution through `deployment/operation_runner.go`
(`Hooks.Before` skips receipted steps, `Hooks.After` commits or replays).
Fault points: `activation_before_switch`, `activation_after_switch`,
`data_before_backup`, `worker_after_effect`.

`preview.shell_preview[]` is rendered from the same argv through the SSH
adapter's quoting rule (`sshadapter.RemoteCommand`) and is never executed;
actions without a target verb (credential values, data disposition) render
as descriptions.
