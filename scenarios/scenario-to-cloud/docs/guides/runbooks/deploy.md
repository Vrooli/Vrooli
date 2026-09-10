# Runbook: deploy a scenario for the first time

Outcome: a deployment record with a stable id, one applied plan, a healthy
current observation and a served public route. Time: 5–15 minutes on a
prepared target.

## 0. Prerequisites (do not start without them)

| Need | How you know it is met |
|---|---|
| A target on a certified platform (`docs/reference/support-policy.md`: Ubuntu 24.04 LTS amd64/arm64) | `scenario-to-cloud preflight requirements` prints the policy; `preflight run` reports the target against it |
| A reach path: the target enrolled through `vrooli-bridge onboard` (`transport: bridge`), or an explicit SSH connection recorded on the manifest (`transport: ssh`) | `deployment resolve` prints `target: machine:<id> (bridge)` or `host:<addr> (ssh)`; a revoked enrollment is `reach_unavailable` and never falls back |
| DNS for the public domain pointing at the target | `edge dns-check` after the record exists; `dns_policy: required` blocks the plan otherwise |
| The scenario's declarations complete (`.vrooli/service.json`, resource declarations, credential descriptors) | `manifest doctor` and the closure preview name every unmet requirement |
| Operator-supplied secrets ready to hand over | the plan's `needs_input` handoff lists the exact addresses |

## 1. Author and validate the manifest

```bash
scenario-to-cloud manifest init --scenario <scenario-id> --host <target-host> --domain <public-domain> --out scenarios/<scenario-id>/.vrooli/cloud/manifest.prod.json
scenario-to-cloud manifest doctor scenarios/<scenario-id>/.vrooli/cloud/manifest.prod.json
scenario-to-cloud manifest validate scenarios/<scenario-id>/.vrooli/cloud/manifest.prod.json
```

`manifest fix` applies the doctor's proposals; re-run `validate` after it.
Field reference: `docs/guides/manifest-reference.md`.

## 2. Check the target

```bash
scenario-to-cloud preflight run scenarios/<scenario-id>/.vrooli/cloud/manifest.prod.json --json
```

Read every `fail` before continuing. The fixes below are owner operations
through the bounded adapter (privilege-broker actions, scoped lifecycle
stops); they act on an explicit connection because no deployment record
exists yet:

```bash
scenario-to-cloud preflight fix-firewall --scenario <scenario-id> --host <target-host>
scenario-to-cloud preflight fix-processes --scenario <scenario-id> --host <target-host>
```

A capacity failure (disk, memory, architecture) is not fixed here: choose a
different target or trim the closure. `unsupported_capability` names the
component and the reason (for example `missing_platform_artifact` on an
architecture the resource does not ship for).

## 3. Create the deployment record

```bash
scenario-to-cloud deployment create scenarios/<scenario-id>/.vrooli/cloud/manifest.prod.json --json
scenario-to-cloud deployment resolve --scenario <scenario-id> --environment production
```

The second line prints the identity line you will reuse:
`deployment: <deployment-id>  scenario: <scenario-id>  environment: production  target: … (<transport>)`.
A second record for the same `(scenario, environment, target)` is refused;
one scenario may have several environments, each its own deployment.

## 4. Review the plan

```bash
scenario-to-cloud deployment plan <deployment-id>
scenario-to-cloud deployment plan <deployment-id> --show-commands     # derived argv preview, read-only
```

What you are approving (`docs/reference/executable-plan.md`):

- **plan digest** — the identity of exactly this change; copy it;
- **release / configuration / closure digests** — what will be on the target;
- **changes** — each action with its effect, required capability,
  verification, recovery and retry class;
- **data effects** — bindings that will be inventoried and the recovery
  point that will be captured before activation;
- **downtime** — `maintenance` declares an expected bound (default 60 s);
  `side_by_side` declares none;
- **outcome** — `apply`; `no_op` means nothing to do; `needs_input` prints the
  onboarding handoff (`vrooli-onboarding://…`) and the missing addresses:
  supply them there, then plan again. Nothing has run.

## 5. Apply exactly what you reviewed

```bash
scenario-to-cloud deployment apply <deployment-id> --plan-digest <plan-digest> --request-key deploy-<scenario-id>-1 --timeout 900
```

`apply` admits the digest as a durable operation and waits once. Use the
same `--request-key` if you have to re-issue after a disconnect: the same
key and digest return the same operation instead of a second one.

Equivalent one-step forms when the review has already happened in front of
you: `deployment execute <deployment-id> --yes` (compile, print the review,
apply the digest just printed, wait), or `redeploy <manifest.json>`
(create-or-update the record, then execute).

| Exit | Meaning | Do |
|---|---|---|
| 0 | `succeeded` | go to step 6 |
| 3 | admitted, not terminal (`--no-wait`, or `waiting_input`) | `operation wait <operation-id> --timeout 900` |
| 124 | wait bound elapsed; operation unchanged | `operation resume <operation-id> --timeout 900` |
| 2 | refused: `plan_stale`, `plan_digest_mismatch`, `request_key_conflict`, `forbidden_target`, `deployment_selector_ambiguous` | re-plan (step 4) or fix the authority; nothing ran |
| 1 | `failed` / `failed_recovery` | `operation get <operation-id>` shows the failed step, receipts and `next_action`; `result.recovery_outcome` says whether service was restored. Follow `incident.md` §3 |

## 6. Verify with a fresh observation

```bash
scenario-to-cloud deployment health <deployment-id> --json
scenario-to-cloud edge status <deployment-id> --json
```

Required before you call it deployed: `status: HEALTHY`, `freshness: CURRENT`,
`observed_release_digest` equal to the plan's release digest, `partial: false`;
edge `routes[]` carrying the public host and `tls` issued. `UNKNOWN` or
`STALE` is never healthy (`docs/reference/health-contract.md`).

## 7. Record

Keep with the deployment's evidence: deployment id, plan digest, operation
id, release digest, the observation timestamp. A first deployment is the
baseline every later update and rollback compares against.

## Failure notes

- **`release_verification_failed`** at `release.verify`: the bundle has no
  release manifest beside it. Build the release (`deployment plan --force-bundle`
  rebuilds and re-signs) and re-plan.
- **`reach_unavailable`** with `transport: bridge`: the enrollment was
  revoked or the node is offline. Re-enroll through `vrooli-bridge onboard`;
  the cloud never falls back to SSH on its own.
- **`operation_conflict` naming `blocking_operation_id`**: another operation
  holds this deployment's lease. Wait on it (`operation wait <id>`); operations
  of one deployment run strictly one after another.
