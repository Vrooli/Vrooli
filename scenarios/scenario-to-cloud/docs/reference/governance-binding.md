# Governance binding: evidence, approval, publication and recovery

> [CODE: api/evidence/record.go] · [CODE: api/ramp/driver.go] · [CODE: api/handlers_publication.go] · [CODE: api/handlers_deployment_recovery.go] · [CODE: ../../packages/proto/schemas/scenario-to-cloud/v1/evidence/evidence.proto] · [DOC: reference/health-contract.md] · [DOC: reference/release-identity.md]

scenario-to-cloud is the owner of target execution and target evidence.
Deployment Manager is the owner of the release decision. Neither side
substitutes for the other: the cloud never approves a release, and Deployment
Manager never runs a deployment engine of its own. This page is the contract
between them.

## 1. Vocabulary

| Word | Meaning | Ever a pass? |
|---|---|---|
| `passed` | the cell's assertion was proven by a target-owned receipt | yes |
| `failed` | the assertion was disproven; the record stays in force until an owner withdraws it with a reason | no |
| `skipped` | the owner chose not to run the cell (recorded reason) | no |
| `unsupported` | the combination is declared unsupported by policy | no |
| `unavailable` | the owner tried and could not produce evidence (no receipt, no reach, no bytes) | no |
| `missing` | no record exists for this release and target; also the answer when evidence exists only for another release or target (incompatible) | no |

Dispatch acceptance (an operation was admitted, an SSH command returned) is
never a target assertion. A passed record needs at least one target-owned
receipt reference; without one the driver records `unavailable`.

## 2. Capability profile `cloud-launch-v1`

The profile is the release lane of the certification matrix
(`certification/matrix.json`, revision `cloud-launch-v1`): RELEASE-07/08,
EDGE-07/08, GOV-01..08, RUN-03/04/05, DATA-04/05, each with the lanes the
matrix declares for it. No cell names an installer, window or desktop
capability. `api/evidence.LoadProfile` refuses any other profile id.

## 3. Evidence records

`cloud_evidence_records` is append-only. A record binds one cell to
`{producer_ref, deployment_id, release_digest, target_key, operation_id |
observation_id}` and carries receipt references, assertions, a reason for
anything but `passed`, and the producer signature over its canonical JSON.

- A rerun appends a new record; the original stays.
- A failed record is never masked by a later pass. A withdrawal is a record
  that names the failed id (`withdraws`) and says why.
- Evaluation precedence per cell: failed in force → latest passed → latest
  other → missing. Records for another release digest or target key are
  reported as incompatible and never counted.

`GET /api/v1/releases/{digest}/evidence[?deployment_id=]` and
`GET /api/v1/deployments/{id}/evidence[?release_digest=]` return the per-cell
summary (`ReleaseEvidence` on the wire): `passed` is true only when every
required cell is `passed` for that exact release and target.

## 4. Exact review identity

A governance decision binds to:

```
scenario_id, profile_id, candidate_commit, artifact_digest, release_digest,
configuration_digest, target_set (sorted target keys), environment, channel,
policy_version [, candidate_id, destination_revision_id, authorization_epoch]
```

`api/evidence.ReviewIdentity.Digest()` is sha256 over the canonical JSON. Any
change to any facet is a different identity. The cloud stores the digest at
request time and re-derives it before every effect.

Deployment Manager's readiness identity is the projection
`{scenario, profile_id, candidate_commit, artifact_digest, targets, channel,
policy_version, candidate_id, destination_revision_id, authorization_epoch}`;
`environment` and `configuration_digest` are cloud-side facets included in
the cloud digest. When Deployment Manager drives the release it passes its
identity in the deployment request (`review`), and `artifact_digest` is its
candidate artifact digest; the cloud keeps `release_digest` beside it.

## 5. Publication protocol

```
POST /api/v1/deployments/{id}/publication/request   {request_key, identity}
POST /api/v1/deployments/{id}/publication/apply     {request_key, review_ref, plan_digest}
GET  /api/v1/deployments/{id}/publication[?request_key=]
```

1. **request** evaluates the profile for the deployment's release and target,
   reports one coverage `TargetVerdict` to Deployment Manager's
   `EvidenceService.ReportTargetVerdict` (target platform = the cloud target
   key so the `ramp-evidence-complete` criterion matches the review's target
   set; one ref per cell record), asks `ReadinessService.PrepareReview` for
   the exact identity and stores `review_ref`. State: `review_prepared`
   (`approved` when the owner already approved). A governance outage is
   `governance_unavailable` (503, retryable); the publication is recorded as
   refused with `review_unavailable`.
2. Approval happens in Deployment Manager (`readiness-reviews approve`).
3. **apply** re-reads the review immediately before the effect and refuses
   (`publication_refused`, 409, `details.refusal`) when the review reference
   differs, the review is not `approved` (superseded or promoted count as
   revoked), any identity facet differs from the stored identity, the stored
   identity digest no longer matches, or any required cell is not passed.
   Only then is the reviewed plan admitted (`plan_digest`; a mismatch is
   `plan_digest_mismatch`). State: `activating`.
4. **reconcile** (on apply replay and GET) reads the target's own
   `active-release.json`. The published release and predecessor are what the
   pointer says; a pointer naming another release is `target_release_mismatch`
   and the publication is `failed`, never "published for the approved
   digest". A lost reply leaves the publication `activating` with
   `target_receipt_unavailable`; the next read reconciles. The activation is
   admitted at most once per request key.

The predecessor is the pointer's `previous_release`, else the most recent
published release for the deployment. It is never caller-supplied.

## 6. Receipts

`GET /api/v1/deployments/{id}/receipt[?release_digest=&bundle_sha256=&target_key=]`
returns the deployment receipt bound to `{producer_ref, deployment_id,
release_digest, configuration_digest, target_key, operation_id}` and signed
(purpose `cloud-evidence-receipt-v1`) with the credential-authority Ed25519
signer (identity `scenario-to-cloud/receipt-signing`) when the authority is
available, otherwise the development HMAC signer; `attestation.production_signer`
says which. A receipt that does not match the caller's expectation or whose
signature does not verify is `receipt_invalid` (422). `observed_at` is the
time the record reached its deployed state, never the read time.

Deployment Manager (`deployments.validateCloudReceipt`) refuses a receipt
whose producer is not `scenario-to-cloud`, that names no target, that is
bound to another release than expected, or whose `observed_at` is older than
two hours on the DM clock; it then reads the evidence summary and refuses
promotion unless every required cell is `passed` for the receipt's release
and target (`checkCloudEvidence`). The persisted publication receipt carries
the activated release, the predecessor, the review key and the evidence
reference for the dossier.

## 7. Recovery

Rollback and forward repair (`POST /api/v1/deployments/{id}/recovery`) bind
to `review_ref` of a published release on the deployment (`recovery_unbound`
otherwise). Rollback must target the recorded predecessor exactly
(`rollback_target_unsupported`). A dry run returns `preview_ref`, a digest
over the exact effect (deployment, action, current bundle, repair bundle,
review, route); execution must present it (`preview_required`, 412). The
route is recorded on the operation: `cloud_target_release_rollback` /
`cloud_target_release_activate` when the target binding carries a Bridge
machine identity, `bundle_pipeline` otherwise. Missing authority is refused
by the management boundary (`forbidden_scope`) before any of this runs.

Limitation: the cloud-target route is selected and recorded, but execution
still runs the retained-bundle pipeline until the reach layer (P09) can
address the native `vrooli cloud-target release rollback` verb.

## 8. Delivery-ramp adapters

`api/ramp` implements the shared seams: `Prober` (targets from the cloud
deployments, transport `bridge` or `ssh`, readiness from the typed health
observation with the cause on unavailable targets), `Builder` (immutable
release artifact set, `kind: cloud-release`), `Driver` (one profile cell via
the owner pipeline, target-owned assertions), `Distributor` (activation whose
effect receipt is the target pointer). Conformance:
`api/ramp/ramp_test.go`, modelled on the spine's reference ramp; the desktop
consumer's tests remain the second implementer.

Limitation: no production runner executes profile cells yet; records are
produced by the driver when a qualification lane (P23) drives it. Until
then every cell is honestly `missing` and publication is refused.
