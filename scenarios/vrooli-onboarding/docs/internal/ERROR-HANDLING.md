# Error Handling

Handlers return actionable, metadata-safe Connect errors. Domain services keep
sentinel errors transport-free; handlers map them at the boundary. The CLI and
UI consume the same code, so callers can choose a recovery action without
matching human message text.

## Sentinel-to-Connect mapping

| Domain | Sentinel or condition | Connect code | Recovery |
|---|---|---|---|
| operator inputs | invalid answer | `invalid_argument` | Correct the value and submit again. |
| operator inputs | queue or state failure | `internal` | Inspect the API log and retry after the state store is available. |
| readiness | missing digest | `invalid_argument` | Re-read readiness and acknowledge the returned digest. |
| readiness | digest changed before acknowledgement | `aborted` | Re-read the current readiness report. |
| apply | invalid or unknown run id | `invalid_argument` / `not_found` | Use the run id returned by `StartApply`. |
| selection | missing required node identity | `invalid_argument` | Supply the target node identity. |
| capabilities | invalid action or missing confirmation | `invalid_argument` | Review the preview and provide the required inputs or confirmation. |
| capabilities | provider cannot perform the action | `failed_precondition` | Follow the provider remediation in the response. |
| credentials | invalid identity or value | `invalid_argument` | Supply the logical id and value through the write-only path. |
| credentials | authority cannot provision or diagnose | `unavailable` | Repair or unlock the credential authority. |
| host | control-plane failure | `internal` / `failed_precondition` | Follow the host requirement remediation. |
| resources and glossary | catalog or control-plane failure | `internal` | Repair the catalog or control-plane dependency, then retry. |
| operator state | invalid patch or state encoding | `invalid_argument` | Correct the field-scoped patch; no write occurs on rejection. |
| any target-aware call | missing namespace grant | `permission_denied` | Grant the scenario namespace to the owner on the target node. |
| any target-aware call | node offline or unreachable | `unavailable` | Bring the node and Bridge connection online. |
| any target-aware call | deadline exceeded | `deadline_exceeded` | Retry with a live target or a longer request deadline. |

Authorization failures use `unauthenticated` when the owner token is missing
or invalid. Internal errors preserve operation context but redact request
values, especially credentials.
