# Phase A2 platform-vocabulary evidence — 2026-08-25

- `go test ./internal/deployability/...` — passed.
- The drift gate now checks the Go `PlatformStatuses()` vocabulary and the
  `$ref` of service, resource, tool, and safeguard consumers. It passed.
- `rg -l '^  "platforms":' internal/tools internal/safeguards` — `0`.
  The duplicate applicability lists were removed; status maps remain the
  source from which observed platform lists are derived.
- `vrooli capability ledger --json` — passed; the generated ledger contains
  114 manifests and 45 capability rows plus 29 observed safeguard rows. The
  capability/status cells remained at the Phase 0 counts; no status value was
  changed by this schema-only cleanup.
- `vrooli capability conformance` — six findings, exactly the known Phase 0
  red set: adguard-home and kopia `go mod tidy` findings on macOS/Windows and
  the secrets-manager credentialauthority import on macOS/Windows. No new
  finding was introduced.
