# Source distribution integration contract

Git Control Tower exposes source distributions as a read-only repository
workspace capability. It does not own source recipes, artifacts, approval, or
publication state. The authoritative owner is the
`scenario-to-repository` source-ramp.

## Ownership map

| Concern | Authoritative owner | GCT responsibility |
| --- | --- | --- |
| Source closure and dependency obligations | `scenario-to-repository` | Render the returned closure and unresolved/runtime obligations. |
| Recipe and policy identity | `scenario-to-repository` | Render `recipe_digest` and `policy_digest`; do not reinterpret policy. |
| Deterministic artifact bytes | `scenario-to-repository` | Render `artifact_id` and `artifact_digest`; never assemble or cache a second artifact. |
| Verification receipt | `scenario-to-repository` | Render the receipt and verification status. |
| Deployment Manager decision | Deployment Manager / source-ramp handoff | Render the decision/reference, including an explicit unknown or pending value. |
| Destination and read-back | Human operator through the source-ramp handoff | Render destination, destination revision, and read-back receipt only when supplied. |
| Source/destination drift | `scenario-to-repository` | Render the drift state and recommended review actions. |
| Git, host, credential, or repository writes | No GCT authority | No route, UI action, or adapter performs these writes. |

The typed source-ramp contract is
`packages/proto/schemas/scenario-to-repository/v1/source/source.proto`.
GCT consumes its Connect read methods through the following REST projection:

| GCT route | Source-ramp read | Purpose |
| --- | --- | --- |
| `GET /api/v1/source-distributions` | `ListDistributions` | List durable candidates, preserving the exact tuple. |
| `GET /api/v1/source-distributions/{id}` | `GetDistribution`, `GetDistributionContents`, `GetPublicationHandoff`, `GetDistributionDrift` | Open one candidate and show contents, handoff, and drift as separate evidence. |

The `repository_context` and `scenario` query values are filters/context only;
they do not change source-ramp authority or create a new record.

## Evidence and operator journey

The workspace entry is labelled **Source distributions** and is available from
desktop and mobile chrome. The operator can:

1. Open the candidate list and see source authority, freshness, publication
   standing, artifact identity, and drift.
2. Open a candidate and inspect the complete tuple: scenario, source digest,
   closure digest, recipe digest, policy digest, artifact digest, verification
   receipt, Deployment Manager decision, destination, publication standing,
   and drift.
3. Distinguish admitted application/shared/generated/documentation content
   from excluded private/secret content and unresolved or runtime obligations.
4. Read the human-only handoff and the exact destination read-back requirement.

Approval is not publication. A source is not treated as published or current
unless the source-ramp returns destination evidence and an exact read-back
receipt. An unavailable or partial source-ramp read is surfaced as unavailable;
GCT does not invent a local record.

## Non-goals and support boundary

This integration does not invoke a packager, create a repository, handle
credentials, publish bytes, modify Git, or repair host state. Unsupported
connectors remain unavailable. Deployment Manager may reference the same
distribution identity through `delivery.source_repository`, but it must not
create a parallel source wizard or infer source readiness from a visual or
binary release.
