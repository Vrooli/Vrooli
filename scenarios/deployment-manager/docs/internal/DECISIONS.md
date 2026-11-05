# Internal Decisions

## Domain ownership during the proto-first re-platform

The governance plane owns these domains: `profiles`, `fitness`, `dependencies`,
`swaps`, `bundles`, `secrets`, `approvals`, `releases`, `evidence`, and
`telemetry`. Requirement modules assign each requirement to one of those names.

`build` remains a release-supporting adapter and is not promoted to a domain;
its profile/build coordination stays behind the release boundary. `migrationtasks`
is retained as a compatibility adapter until its callers are removed, but it
does not own persistent state. `codesigning` is a proxy boundary: signing
execution and credentials remain owned by scenario-to-desktop, while the
deployment-manager adapter validates requests and forwards them.
