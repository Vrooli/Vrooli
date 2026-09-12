# Flows

## Purpose Of This Document

Describe the lifecycle rules that preserve correct browser automation evidence.

## Flow Inventory

Author workflow → validate/compile → create execution/session → execute typed actions → persist outcome/evidence → replay or export.

## Flow Details

Compilation rejects invalid or untyped action input before a browser session is used. Replay consumes persisted evidence and must preserve integrity metadata.

## State Machines

Sessions transition through ready, recording, resetting, and closed states. An instruction updates activity only for an existing active session; teardown is idempotent.

## Maturity Ladder

V2 typed action execution is implemented and validated by API, UI, and driver tests. Deployment SLOs and commercial capacity are separate readiness work.

## Production Shape

The API supervises the driver as a sidecar; it owns retry/persistence decisions while the driver owns browser resource cleanup.

## Deferred / Unmodeled Flows

Hosted tenancy, billing enforcement, and video-rendering production jobs remain deferred.

## Cross-References

- [Architecture](ARCHITECTURE.md)
- [Data](DATA.md)
- [Testing](../../../../docs/TESTING.md)
