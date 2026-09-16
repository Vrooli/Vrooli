# Commercial release runbook

This runbook drives one governed Tier-2 desktop release from a built scenario
into the Landing Page Business Suite (LPBS) download catalog. It is the operator
sequence for the commercial path. It repeats no target implementation details:
the desktop ramp is owned by
[scenario-to-desktop](../../../scenario-to-desktop/docs/OVERVIEW.md), the catalog
and update feed are owned by
[landing-page-business-suite](../../landing-page-business-suite/docs/reference/api/OVERVIEW.md),
and the release identity and evidence rules are owned by the
[evidence contract](../guides/evidence-contract.md).

## Planes and ownership

| Plane | Owns | Repo surface |
| --- | --- | --- |
| deployment-manager | Profile, readiness review, release identity, publish gate, release record | `api/releases`, `api/readiness` |
| scenario-to-desktop | Electron build, packaging, signing, publish handoff | `api/pipeline`, `api/deploy` |
| landing-page-business-suite | Artifact bytes, catalog, immutable channel revisions, update feed | `api/internal/delivery` |

deployment-manager owns the decision to publish. The desktop ramp performs the
publication only after the release gate selects it.

## Platform and architecture rule

Release targets are architecture-qualified identifiers such as `linux-x64`,
`darwin-arm64`, and `win-x64`. The LPBS catalog and the electron update feed are
operating-system scoped. One canonical projection
(`api-core/targetmodel.ProjectDesktopPlatform`) maps a target identifier to
`windows`, `mac`, or `linux` plus `amd64` or `arm64`.

- The catalog stores one artifact per operating system per channel. A release
  that maps two architectures onto one operating system is refused before the
  owner effect starts.
- The architecture is retained in artifact metadata (`architecture` and
  `target_id`) so the published bytes remain attributable to a machine class.
- Always pass the same target identifiers to the readiness review, the release
  start, and the candidate. They are compared as an exact set.

## Prerequisites

Local services run through the lifecycle:

```bash
vrooli scenario start deployment-manager
vrooli scenario start scenario-to-desktop
vrooli scenario start landing-page-business-suite
```

Every profile, release, and approval mutation requires a verified human
operator. Authenticate against the identity provider and pin the token to the
CLI before any mutating command:

```bash
scenario-authenticator auth login --email <operator> --resource deployment-manager
deployment-manager configure token <access_token>
```

The deployment-manager audience is `scenario-authenticator:deployment-manager`.
Read-only commands may work signed out; mutations do not. This is the boundary
that a first release always stops at until the operator supplies the credential.

LPBS must have all of the following before a governed release can complete:

- a `download_apps` row for the bundle/app pair (web-console is seeded under
  `business_suite`), and configured storage settings with working S3 credentials;
- an active remote-profile session and service-to-service auth enabled;
- `LPBS_SERVICE_SECRET` provisioned to scenario-to-desktop
  (`scenario-to-desktop deploy-target doctor <target>` reports readiness).

Authority that is not code:

- operating-system installer signing (Windows Authenticode, macOS Developer
  ID/notarization, Linux package signing); web-console currently declares Linux
  GPG only in `signing.json`;
- release-manifest signing through `vrooli release-authority`;
- a reachable staging destination and an independent reviewer.

## 1. Create the profile

```bash
deployment-manager profiles create web-console-desktop web-console --tier 2
deployment-manager profiles show web-console-desktop
```

Record the profile ID. It is the key for the release config, candidate, and
release start.

## 2. Configure LPBS release coordinates

Open the profile in the deployment-manager UI and fill the LPBS release config
card, or `PUT /api/v1/profiles/{profile_id}/lpbs-config`:

```json
{
  "lpbs_domain": "https://<lpbs-host>",
  "lpbs_remote_profile": "<remote-profile-tag>",
  "lpbs_app_key": "web-console",
  "default_channel": "stable",
  "update_url": ""
}
```

The publish step refuses to start without an exact app key and remote profile.
The remote profile may be session-authenticated or service-authenticated. For
unattended releases, configure the profile with the destination's
`LPBS_SERVICE_SECRET`; that mode has no seven-day session expiry and requires no
human login.
Leave `update_url` blank for a governed release: the owner-derived update URL is
bound at publish time.

## 3. Register the candidate

The candidate is the immutable source, build-input, and final-signed-bytes
identity. Candidate registration hashes the canonical bytes, so the returned
`candidate_id` and `artifact_manifest_digest` are the values every later step
must bind.

```bash
deployment-manager releases register-candidate --file candidate.json
```

```json
{
  "source_revision": "<git-commit>",
  "profile_revision": "<profile-version-id>",
  "build_inputs": {"node": "22", "electron": "38"},
  "dependency_lock_digest": "sha256:<lockfile-digest>",
  "policy_digest": "sha256:<readiness-policy-digest>",
  "capability_declaration": {
    "support_owner": "vrooli-support",
    "incident_owner": "vrooli-oncall",
    "customer_contact": "support@vrooli.com",
    "release_authority": "deployment-manager",
    "rollback_authority": "deployment-manager",
    "degraded_mode_authority": "vrooli-oncall"
  },
  "artifacts": [
    {
      "target": {
        "id": "linux-x64",
        "platform": "linux",
        "os": "linux",
        "architecture": "amd64",
        "format": "appimage"
      },
      "immutable_ref": "s3://<bucket>/<final-artifact>",
      "digest": "sha512:<final-artifact-digest>",
      "size_bytes": 123456789,
      "signature_digest": "sha256:<signature-digest>",
      "signer_ref": "vrooli/desktop-signing"
    }
  ]
}
```

Add one artifact per target. The artifact `target.id` values are the exact
release target set.

## 4. Register the destination revision

```bash
deployment-manager releases register-destination --file destination.json
```

```json
{
  "kind": "lpbs",
  "destination_id": "<remote-profile-tag>",
  "configuration_digest": "sha256:<lpbs-config-digest>",
  "channel": "stable",
  "destination_revision_id": "web-console-stable-1"
}
```

`destination_id` must equal the LPBS remote profile in the profile release
config. The channel must match the release channel.

## 5. Prepare and approve readiness

Prepare the review with the exact candidate commit, artifact manifest digest,
channel, and target set:

```bash
deployment-manager readiness-reviews prepare \
  web-console <profile_id> <git-commit> <artifact_manifest_digest> stable linux-x64 \
  --fact commercial_release=true --fact paid_release=true \
  --candidate-id <candidate_id> \
  --destination-revision-id <destination_revision_id> \
  --authorization-epoch 1
```

Collect or report producer evidence, then inspect and approve:

```bash
deployment-manager --json readiness-reviews get <review_key>
deployment-manager readiness-reviews approve \
  <review_key> web-console <profile_id> <git-commit> <artifact_manifest_digest> stable <actor> linux-x64 \
  --candidate-id <candidate_id> \
  --destination-revision-id <destination_revision_id> \
  --authorization-epoch 1
```

Approval requires the required criteria to be satisfied or explicitly waived
under policy. A missing owner producer stays visible as missing proof; never
waive a criterion only to make the gate pass. See the
[evidence contract](../guides/evidence-contract.md) and the
[qualification gap register](../internal/QUALIFICATION-GAP-REGISTER.md) for which
owners still need live provisioning.

## 6. Start the release

```bash
deployment-manager releases start <profile_id> \
  --commit <git-commit> \
  --version <semver> \
  --channel stable \
  --platforms linux-x64 \
  --artifact-digest <artifact_manifest_digest> \
  --candidate-id <candidate_id> \
  --destination-revision-id <destination_revision_id> \
  --readiness-review-key <review_key> \
  --authorization-epoch 1 \
  --idempotency-key web-console-<semver>-stable
```

deployment-manager validates the approved readiness binding, then runs the
desktop ramp as a deploy-only pipeline with production trust. The ramp uploads
each artifact through the LPBS proxy (presign, direct S3 PUT, commit), promotes
the complete artifact set atomically, and reports the LPBS publication receipt
back to the release record. The release then re-verifies every published target
with a deep LPBS check that re-reads and hashes the stored bytes.

If the pipeline is durable, `releases start` returns an operation ID. Block on
it once; do not poll:

```bash
deployment-manager releases operation <operation_id>
```

## 7. Verify and inspect

```bash
deployment-manager releases get <release_id>
deployment-manager releases verify <release_id> --deep
deployment-manager releases dossier <release_id>
deployment-manager releases health <release_id>
```

The release reaches `published` only when every requested target matches the
expected version and SHA-512 on the channel. A mismatch records
`verify_failed`; an uncertain owner effect records `ambiguous` and requires
reconciliation.

## What LPBS stores

- `download_artifacts`: one immutable row per uploaded artifact with `platform`
  (operating system), `release_version`, `release_id`, `git_commit_hash`, and
  `sha512`, plus `metadata.architecture` and `metadata.target_id`.
- `download_assets`: the customer-facing asset per `(bundle, app, platform,
  variant)` where `variant` is the channel.
- `download_channel_heads` and `download_channel_revisions`: the mutable head
  and the immutable, predecessor-bound history.

## Recovery

```bash
deployment-manager releases recover <release_id> \
  --review-key <review_key> \
  --candidate-id <candidate_id> \
  --destination-revision-id <destination_revision_id> \
  --action halt|withdraw|rollback|forward_repair \
  --dry-run
```

Rollback and forward repair require explicit data compatibility and an exact
predecessor revision. A halt is a durable offer gate; it does not mutate
installed clients or delete revisions.

## Still requires external execution

The following are not satisfied by local tests and must be earned against real
authority or fixtures before the first commercial release can be certified:

- hosted staging publication and an installed-client update/relaunch receipt;
- Windows and macOS installer signing (or an explicit Linux-only first release);
- an independent reviewer for the final dossier.

See the [qualification gap register](../internal/QUALIFICATION-GAP-REGISTER.md)
for the exact receipts.
