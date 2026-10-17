# Host-neutral collaboration contract

Git Control Tower consumes a provider-neutral host seam. Integration Hub owns
credentials, token refresh, installation identity, transport retries, and
provider SDKs. GCT receives normalized identities and capability standing.

| Capability | GitHub | GitLab | Bitbucket | GCT behavior when unavailable |
|---|---|---|---|---|
| Change metadata and base/head | conformance fixture | conformance fixture | conformance fixture | local range draft remains available |
| Comments | supported by seam | supported by seam | supported by seam | show unsupported/denied/disconnected separately |
| Reviews | provider-specific normalization | provider-specific normalization | provider-specific normalization | no generic broken button |
| Checks/statuses | normalized receipt | normalized receipt | normalized receipt | preserve unavailable checks |
| Releases | normalized draft destination | normalized draft destination | normalized draft destination | draft release notes locally |
| Publication | human-verified external write | human-verified external write | human-verified external write | preview and handoff only |

Every hosted change carries host kind, instance URL, installation/account,
provider repository ID, change number, base revision, and head revision.
Repository names are display text and never identity. Capability standing is
one of `available`, `unsupported`, `denied`, `disconnected`, `revoked`, or
`rate_limited`; `configured` alone is not successful access.

Publication is represented by an exact draft digest, destination, expected
revision, and `human_verified` authority. GCT has no provider SDK or secret
lifecycle implementation and cannot turn an advisory mention into a write.
