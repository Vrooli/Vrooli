# Health Contract

scenario-to-cloud answers two different questions on its health endpoints,
and this document keeps them apart:

1. **HTTP transport semantics** — did the API produce a report?
2. **Deployment health semantics** — is the deployment healthy, for which
   release, on which target, and how recent is that evidence?

A consumer that reads the first as the second is the false-green defect this
contract exists to prevent (plan phase 16, requirements STC-P0-033 and
STC-P0-034).

## 1. HTTP transport semantics

| Status | Meaning |
|---|---|
| `200 OK` | The observation or report was produced. It says nothing about the deployment: the body can be `HEALTH_STATUS_UNHEALTHY` or `HEALTH_STATUS_UNKNOWN`. |
| `404` `deployment_not_found` | No deployment record with that id. |
| `403` `forbidden_scope` / `forbidden_target` | The caller may not read this deployment (see the authorization matrix). |
| `422` `unsupported_capability` | The deployment has no VPS target, so nothing can be inspected. |
| `500` `internal` | The producer itself failed. No observation was made. |

Non-2xx bodies use the typed error envelope from
[Identity and Selectors](identity-and-selectors.md):
`{"error":{"code","message","retryable","next_action","details"}}`.

The producer never encodes "deployment unhealthy" as an HTTP error, and
never encodes "report produced" as "deployment healthy". Do not fix a health
mismatch by making every health failure a transport error.

## 2. Deployment health semantics: the typed observation

Wire schema: `vrooli.scenario_to_cloud.v1.health.HealthObservation`
(`packages/proto/schemas/scenario-to-cloud/v1/health/health.proto`). JSON uses
proto field names and enum names.

### Endpoints

| Surface | Path | Body |
|---|---|---|
| REST | `GET /api/v1/deployments/{id}/health/observation` | `{"schema_version":"1","observation":{…}}` |
| Connect | `HealthService.GetHealthObservation` | `GetHealthObservationResponse` |
| Legacy REST | `GET /api/v1/deployments/{id}/health` | The composed report; its `observation` field carries the same typed message |

Every call makes a fresh inspection (live state over SSH, DNS, TLS). Nothing
is served from a cache, so `observed_at` is the producer time of the
live-state inspection, never the time the response was assembled.

### Fields

| Field | Meaning |
|---|---|
| `deployment_id` | Stable deployment identity. Never a scenario name, domain or host. |
| `target_id` | `machine:<id>` for a Bridge-enrolled machine, `host:<host>` for an SSH-only target. |
| `observed_release_digest` | `sha256:<bundle_sha256>` of the deployment record. Until the release domain (phase 10) supplies a release digest, this is the bundle digest with the `sha256:` prefix. |
| `observed_configuration_digest` | `sha256:` over the stored manifest JSON, until the release domain supplies a configuration digest. |
| `observed_at` | Producer time of the live-state inspection (the attempt time when the inspection failed; then `freshness` is `UNKNOWN`). |
| `status` | `HEALTHY`, `DEGRADED`, `UNHEALTHY`, `UNKNOWN`. `UNKNOWN` means the target could not be observed. |
| `freshness` | `CURRENT` (inspection succeeded within 120 s), `STALE` (succeeded but older), `UNKNOWN` (inspection failed). |
| `checks[]` | One entry per stable check id (below), each with `status`, `reason_code`, `detail`. |
| `partial` / `missing_dependencies` | `true` when an evidence source was absent: `live_state`, `edge_dns`, `edge_tls`. |
| `next_actions[]` | Typed remedies (`owner`, `kind`, `reference`, `label`) for failed or warned checks. |
| `producer_ref` | `scenario-to-cloud:health:v1`. |

### Checks

The four questions the legacy report folded into one level are separate
checks:

| Check id | Question | Evidence |
|---|---|---|
| `host_presence` | Did the target answer the inspection? | SSH connectivity probe |
| `transport_reach` | Is the pinned transport identity authorized? | SSH key authorization state |
| `application_readiness` | Are the scenario and its resources running? | Process inspection |
| `release_freshness` | Does the deployed bundle match local scenario state? | Version and bundle fingerprint parity |
| `edge_dns` | Do the edge records point at the target? | DNS evaluation |
| `edge_tls` | Is the certificate valid and renewable? | TLS probe and ALPN check |
| `system_resources` | CPU, memory, disk pressure | System metrics |
| `deployment_record` | Record status (deployed, stopped, failed, …) | Deployment record |

Check statuses: `PASSED`, `WARNED`, `FAILED`, `SKIPPED` (did not apply),
`UNAVAILABLE` (evidence could not be gathered).

### Status and freshness are independent

| status | freshness | Reading |
|---|---|---|
| `UNHEALTHY` | `CURRENT` | The deployment is unhealthy now. |
| `HEALTHY` | `STALE` | It was healthy; current readiness is unproven. |
| `UNKNOWN` | `UNKNOWN` | The target could not be inspected (SSH failed). Not healthy. |
| `HEALTHY` | `CURRENT` | Healthy now, for `observed_release_digest` on `target_id`. |

A failed SSH inspection is `UNKNOWN`, never `UNHEALTHY` and never `HEALTHY`:
unreachable evidence is not evidence. The legacy `health` level follows the
same rule (`unknown` when the SSH probe did not connect).

## 3. Consumer rules (fail closed)

A consumer may report a deployment healthy only when all of the following
hold. Deployment Manager implements exactly this list in
`scenarios/deployment-manager/api/deployments/cloud_health.go`, and it is the
only health interpretation in that scenario.

1. `schema_version` is `"1"`.
2. `observation.deployment_id` equals the id the consumer resolved (through
   `GET /api/v1/deployments/resolve`, never a hardcoded slug).
3. `status` is `HEALTH_STATUS_HEALTHY`. `UNKNOWN`, `UNSPECIFIED`, a missing
   value, or a value the consumer does not know all fail closed.
4. `freshness` is `FRESHNESS_CURRENT`.
5. `observed_at` is present and within the consumer's own maximum age
   (Deployment Manager default: 120 s on its clock).
6. `observed_release_digest` equals the expected release when the consumer
   has one (compare after normalising the `sha256:` prefix and case).
7. `partial` is `false`.
8. The body decoded. Malformed JSON, an empty body, or an unknown mandatory
   enum value is `malformed_report`, not "no opinion".

Unknown *fields* are forward-compatible and ignored. Unknown mandatory
*status values* are not.

Consumer verdicts carry a reason code so a refused promotion is explainable:
`transport_error`, `malformed_report`, `unsupported_schema_version`,
`deployment_mismatch`, `environment_mismatch`, `release_mismatch`,
`status_unknown`, `status_not_healthy`, `freshness_unknown`,
`stale_observation`, `observed_at_missing`, `partial_observation`, or the
cloud service's own stable code when it refused the request
(`deployment_not_found`, `deployment_selector_ambiguous`, `health_unknown`, …).

## 4. Receipts are not observations

`GET /api/v1/deployments/{id}/receipt` returns `health: "healthy"` when the
deployment record's last deploy result succeeded. That is a record-state
claim, not an observation of the target. Consumers that need current
readiness fetch the observation as well; Deployment Manager does this
immediately after receipt validation and again in its release gate with the
expected bundle digest.

## 5. Freshness cost boundary

Interactive health calls compare the declared scenario version and defer the
full local bundle fingerprint to the explicit release-freshness workflow. This
keeps the live observation path bounded; the fingerprint workflow remains the
authoritative check when release parity is required.
