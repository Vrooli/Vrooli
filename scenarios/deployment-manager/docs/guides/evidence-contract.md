# Shared evidence contract

Ramps produce evidence; deployment-manager governs it. The wire contract is
defined in `packages/proto/schemas/common/v1/evidence.proto` and ingested by
the deployment-manager `EvidenceService`.

A producer reports one `TargetVerdict` for a target and an exact source
commit. The verdict identifies the ramp, platform, OS, device kind, run, and
disposition. `EvidenceRef` identifies producer-owned artifacts by producer,
artifact ID, checksum, kind, and size. The governance plane stores this
metadata and reference only; it never proxies or stores artifact bytes.

## Producer manifest lifecycle

Before mapping to `TargetVerdict`, scenario-to-desktop emits its versioned
producer manifest (`schema_version: 1`). The manifest has one terminal state and
cannot skip a required gate:

```text
created -> protocol_ready -> visual_launched -> journey_passed
        -> capture_integrity_passed -> artifacts_persisted
        -> governance_reported -> passed
```

`failed`, `degraded`, and `unavailable` are terminal non-pass states. The
`protocol` profile requires only protocol readiness; `visual` additionally
requires visual launch, semantic journey, capture integrity, and artifact
persistence; `release_visual` also requires governance reporting. A protocol
or compile-only result therefore cannot serialize as a visual pass.

Each manifest records the run, target, runner and target OS, artifact digest,
gate dispositions, and producer-owned artifact metadata. Recordings must carry
an immutable reference, absolute local operator path when available, checksum,
size, dimensions, duration, container, codec, creation time, and a useful-frame
result. These fields are validated before a verdict is reported. Deployment-
manager receives only the resulting references and never the MP4 or screenshot
bytes.

## Producer example

scenario-to-desktop runs its smoke-test journey, writes the recording and
screenshots to its capture store, and maps the result to the shared contract:

```text
Target:      ramp=scenario-to-desktop, platform=desktop, os=linux
Device kind: host
Disposition: PASSED or FAILED
Run ID:      smoke-<id>
Refs:        capture:<id> and screenshots, each with checksum and size
Detail:      ordered journey steps and named degraded reason, if any
```

The producer then calls `EvidenceService.ReportTargetVerdict`. If the
governance service is unreachable, the producer reports the failure to its
caller; it does not convert the run into an implicit pass.

## Degraded journeys

A journey that cannot interact with a real application window is a failed
disposition with a named reason such as `xdotool_unavailable`,
`window_manager_not_started`, or `no_visible_window`. The evidence review
surface shows that reason and the ordered steps, while the recording link
continues to resolve through the producer's artifact route.

## Review and release

The release gate matches required targets and evidence by profile and exact
commit. Every required target must have a passed verdict and a human approval;
missing, failed, or stale evidence keeps the gate closed.
