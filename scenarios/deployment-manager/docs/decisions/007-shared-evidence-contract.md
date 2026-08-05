# ADR-007: Shared evidence contract location

Status: Accepted and Executed

The neutral target-verdict and evidence-reference messages live in
`packages/proto/schemas/common/v1/evidence.proto`, not in deployment-manager.

Ramps own evidence production and retain artifact bytes. Deployment-manager
stores only producer references, checksums, sizes, and verdict metadata. A
common package lets a ramp compile against a stable neutral contract without
coupling its internal storage model or build to the governance scenario.

Every target is identified by ramp, platform, OS, and device kind. Producers
report a failed disposition for degraded runs; an unreachable governance
service is a reporting failure, never an implicit pass.
