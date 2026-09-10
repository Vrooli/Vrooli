# Runbook: incident (first ten minutes)

Outcome: the deployment is either recovered with a fresh healthy observation
or contained with a named next action, and the record exists. The full
interpretation tables live in `docs/guides/incident-runbook.md`; this card is
the order of operations so nobody improvises a second writer during an
outage.

## 1. Diagnose (read only, three observations)

```bash
scenario-to-cloud deployment resolve --scenario <scenario-id> --environment production --json
scenario-to-cloud deployment health <deployment-id> --json
scenario-to-cloud edge status <deployment-id> --json
scenario-to-cloud operation list <deployment-id> --json
```

| You see | Go to |
|---|---|
| a non-terminal operation (`running`, `reconciling`, `recovering`, `waiting_input`) | §2: the operation owns the deployment; never start another |
| `freshness: STALE`/`UNKNOWN`, `host_presence` failed | §2 reconcile, then §3 only after the host answers |
| `UNHEALTHY`, `application_readiness` failed, host reachable | §3a |
| `release_freshness` failed (target runs another release) | §3b |
| `edge_tls` warned, `renewal_state: renewal_failed` | §3c (service may still be up) |
| `HEALTHY`/`CURRENT` and the alert was `health_observation_stale` | §4: the alert should resolve on this observation |

For a non-terminal operation:

```bash
scenario-to-cloud operation get <operation-id>
```

`reconciling` with an `unknown_effect` means the owner could not learn a
step's outcome; `next_action` names the fix. Never re-run the step by hand.

## 2. Contain

- **Running operation making it worse** — cancel intent, honoured at the next
  cancel point (never mid-effect):

  ```bash
  scenario-to-cloud operation cancel <operation-id>
  scenario-to-cloud operation wait <operation-id> --timeout 120
  ```

- **Owner restarted or lost the target reply** — reconciliation reacquires
  non-terminal records under a higher fence and reads target receipts before
  replaying anything:

  ```bash
  curl -sS -X POST "$STC_API/api/v1/operations/reconcile" -H "Authorization: Bearer $STC_TOKEN"
  ```

- **Must stop serving** (bad release, data at risk):

  ```bash
  scenario-to-cloud deployment stop <deployment-id>
  ```

Do not delete bundles, recovery points or receipts to make room; the
retention planners refuse protected artifacts, and an incident is when they
are protected.

## 3. Recover (smallest supported change; preview first)

**a. Workload down, host and release fine**

```bash
scenario-to-cloud deployment plan <deployment-id> --scope start --json
scenario-to-cloud deployment start <deployment-id> --yes --timeout 600
```

**b. Bad release, predecessor retained** (`update.md` §6a)

```bash
scenario-to-cloud deployment rollback <deployment-id> --dry-run --json
scenario-to-cloud deployment rollback <deployment-id> --confirm --preview-ref <preview-ref> --request-key incident-<id>-rollback
```

`data_compatibility` in the refusal → `restore.md`, then roll back.

**c. Certificate renewal failing**

```bash
scenario-to-cloud edge tls <deployment-id> --json
scenario-to-cloud edge tls-renew <deployment-id>
```

Never disable verification.

**d. Host lost or data damaged** → `restore.md` (replacement host path).

**e. Credential compromised or rotation parked** → `rotate.md`.

Every recovery except stop and tls-renew is an operation: wait once, and if
the wait times out reattach instead of re-issuing.

```bash
scenario-to-cloud operation resume <operation-id> --timeout 600
```

## 4. Verify

```bash
scenario-to-cloud deployment health <deployment-id> --json
scenario-to-cloud edge status <deployment-id> --json
```

Required: `HEALTHY`, `CURRENT`, `observed_release_digest` equal to the
intended release, `partial: false`. A healthy but stale observation is not
recovery: wait one observation interval and read again. The alert owner then
publishes `resolved` for the incident id within the recovery notification
budget (120 s); if the alert stays open after a healthy current observation
the emitter is the defect, not the service.

## 5. Record

Deployment id, target id, release digests before and after, alert
`incident_id` and `detected_at`, every operation id with its terminal state,
recovery-point id and measured RTO if a restore ran, what remains open
(certificate, rotation, reconciliation `next_action`). Keep it with the
release's certification evidence. Detection (60 s) and recovery notification
(120 s) budgets are measured from these timestamps; a missed budget is
recorded as missed (`docs/reference/service-objectives.md`).
