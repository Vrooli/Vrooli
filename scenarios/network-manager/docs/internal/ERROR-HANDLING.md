# Error Handling — Network Manager

## Proto-Typed Operations

Owned API operations should return typed Connect errors. Expected categories:

- `invalid_argument`: malformed policy, snapshot, or adapter request.
- `failed_precondition`: action requires approval, capability, or configured resolver.
- `permission_denied`: operator lacks permission for sensitive visibility or mutation.
- `unavailable`: resolver, router, or external adapter is unreachable.
- `deadline_exceeded`: probe or candidate exceeded timeout.
- `internal`: unexpected server error.

## Sentinel Mapping

Domain services should expose typed sentinel errors for predictable states:

- unsupported adapter action,
- approval required,
- rollback unavailable,
- resolver unreachable,
- ambiguous device identity,
- retention policy violation.

Handlers translate these sentinels into Connect codes. UI and CLI render actionable next steps.

## Multipart REST Exceptions

No multipart REST exceptions are planned for P0. If a future report import/export or packet-capture upload needs opaque bytes, document the exception in API metadata and keep the response proto-typed.

## Cross-references

- [`SEAMS.md`](SEAMS.md)
- [`../reference/api-endpoints.md`](../reference/api-endpoints.md)
