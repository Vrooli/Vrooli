# Incident runbook

Use this when an alert of `docs/reference/service-objectives.md` arrives or
an operator reports a deployed scenario as unavailable. Every step uses a
supported command from `docs/reference/cli-commands.md` or the matching API
endpoint; nothing here asks you to SSH in and run shell by hand. Record what
you did at the end (step 5); the record is part of the incident, not
optional.

Exit codes throughout: 0 ok, 1 failed, 2 refused, 3 pending, 124 observer
timeout (the operation is unchanged; re-attach with `operation resume`).

Identify the deployment once and reuse the id:

```bash
scenario-to-cloud deployment resolve --scenario "$SCENARIO_ID" --environment production --json
```

## 1. Diagnose

Read three observations before touching anything. Each is a typed record;
`unknown` is never healthy.

```bash
scenario-to-cloud deployment health "$DEPLOYMENT_ID" --json
scenario-to-cloud edge status "$DEPLOYMENT_ID" --json
scenario-to-cloud operation list "$DEPLOYMENT_ID" --json
```

Interpretation:

| You see | It means | Go to |
|---|---|---|
| `status: HEALTHY`, `freshness: CURRENT`, alert was `health_observation_stale` | the producer had not observed recently; the alert should resolve on this observation | 4 |
| `freshness: STALE` or `UNKNOWN`, `host_presence` failed | the target is unreachable over the bound transport (or the enrollment is revoked: `reach_unavailable`) | 2 then 3 |
| `status: UNHEALTHY`, `application_readiness` failed, host reachable | the workload is down, the host is up | 3 (start scope) |
| `release_freshness` failed | the target runs a release other than the recorded one | 3 (rollback eligibility) |
| `edge_tls` warned, `renewal_state: renewal_failed` | certificate renewal is failing; service may still be up | 3 (edge) |
| a non-terminal operation (`running`, `reconciling`, `recovering`, `waiting_input`) | an operation owns the deployment; do not start another | 2 |

For a non-terminal operation read its standing:

```bash
scenario-to-cloud operation get "$OPERATION_ID"
```

`reconciling` with an `unknown_effect` means the owner could not learn a
step's outcome from the target; the `next_action` names the fix (usually
"install the native vrooli CLI on the target, then reconcile"). Never
re-run the step by hand.

## 2. Contain

Stop the bleeding without creating a second writer.

- **An operation is running and is making it worse:** record a cancel
  intent; it is honoured at the next declared cancel point and never mid-
  effect.

  ```bash
  scenario-to-cloud operation cancel "$OPERATION_ID"
  scenario-to-cloud operation wait "$OPERATION_ID" --timeout 120
  ```

  A cancelled operation ends `cancelled` with the receipts of the steps that
  completed; the deployment status projects `failed` with the cancel message.

- **The owner restarted or lost the target reply:** run reconciliation; it
  reacquires non-terminal records under a higher fence, reads target
  receipts before replaying anything, and parks what it cannot prove.

  ```bash
  curl -sS -X POST "$STC_API/api/v1/operations/reconcile" -H "Authorization: Bearer $STC_TOKEN"
  ```

- **The workload must stop serving (bad release, data risk):**

  ```bash
  scenario-to-cloud deployment stop "$DEPLOYMENT_ID"
  ```

  This is synchronous and is not an operation; note it in the record.

- **Do not** delete bundles, recovery points or receipts to make room; the
  retention planners refuse protected artifacts and an incident is exactly
  when they are protected.

## 3. Recover

Pick the smallest supported change. Preview first; the preview digest is
what you apply.

- **Workload down, host and release fine:** start scope.

  ```bash
  scenario-to-cloud deployment plan "$DEPLOYMENT_ID" --scope start --json
  scenario-to-cloud deployment start "$DEPLOYMENT_ID" --yes --timeout 600
  ```

- **Bad release, predecessor known good:** governed rollback. Eligibility is
  decided by the server (predecessor recorded, schema-compatible,
  governance binding intact); an ineligible rollback is refused with the
  reason, not attempted.

  ```bash
  scenario-to-cloud deployment rollback "$DEPLOYMENT_ID" --dry-run --json
  scenario-to-cloud deployment rollback "$DEPLOYMENT_ID" --confirm --preview-ref "$PREVIEW_REF" --request-key "incident-$INCIDENT_ID-rollback"
  ```

  `data_compatibility` in the refusal means the predecessor cannot read the
  current schema; use a recovery point instead (`docs/guides/recovery-runbook.md`
  §4).

- **Host lost or data damaged:** restore onto a replacement target from the
  newest verified recovery point.

  ```bash
  scenario-to-cloud deployment recovery-points list "$DEPLOYMENT_ID" --json
  scenario-to-cloud deployment recovery-points verify "$DEPLOYMENT_ID" --recovery-point "$RECOVERY_POINT_ID" --open
  scenario-to-cloud deployment recovery-points restore "$DEPLOYMENT_ID" --recovery-point "$RECOVERY_POINT_ID" --target-ref "$TARGET_REF" --dry-run
  scenario-to-cloud deployment recovery-points restore "$DEPLOYMENT_ID" --recovery-point "$RECOVERY_POINT_ID" --target-ref "$TARGET_REF"
  ```

  The restore receipt records the measured RTO and the recovery-point age
  against the RPO (`docs/guides/recovery-runbook.md` §3).

- **Certificate renewal failing:** inspect the renewal state, then renew
  through the edge owner; never disable verification.

  ```bash
  scenario-to-cloud edge tls "$DEPLOYMENT_ID" --json
  scenario-to-cloud edge tls-renew "$DEPLOYMENT_ID"
  ```

- **Credential rotation pending or a credential compromised:** follow
  `docs/reference/credential-lifecycle.md` (rotate → activate → revoke); the
  alert clears when no binding has a pending rotation.

Every recovery above is an operation (except stop and tls-renew): wait once,
and if the wait times out re-attach instead of re-issuing.

```bash
scenario-to-cloud operation resume "$OPERATION_ID" --timeout 600
```

## 4. Verify

Recovery is proven by a fresh observation, not by the operation's success.

```bash
scenario-to-cloud deployment health "$DEPLOYMENT_ID" --json
scenario-to-cloud edge status "$DEPLOYMENT_ID" --json
```

Required: `status: HEALTHY`, `freshness: CURRENT`, `observed_release_digest`
equal to the release you intended, `partial: false`. The alert owner then
publishes `status: resolved` for the incident within the recovery
notification budget (120 s); if the alert stays open after a healthy current
observation, the emitter is the fault, not the service (record it as a
separate defect).

If the observation is healthy but stale, wait one observation interval and
read again; do not declare recovery on a stale observation.

## 5. Record

Write the incident record while the identifiers are in front of you:

- deployment id, target id, release digest before and after;
- alert `incident_id` and `detected_at`; time the condition was met if known;
- operation ids of every contain/recover step with their terminal state;
- recovery-point id used (if any), measured RTO from the restore receipt;
- what remains open (certificate, rotation, reconciliation next_action).

Keep the record with the certification evidence for the release
(`certification/evidence/`, or the plan's `evidence/P23-*.md` during
qualification). The detection budget (60 s) and the recovery notification
budget (120 s) are measured from these timestamps; a missed budget is
recorded as missed.

## What this runbook does not do

- It never runs shell on the target; target effects go through
  `vrooli cloud-target` verbs invoked by the operation owner.
- It never edits the production deployment record by hand.
- It never lowers a budget or withdraws a failed receipt to close an
  incident.
