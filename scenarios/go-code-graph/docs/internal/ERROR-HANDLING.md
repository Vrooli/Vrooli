# Error Handling

## Proto-Typed Operations

The scenario uses Connect-RPC for all wire calls. Errors move through three layers:

1. Domain/service code returns typed sentinels such as `graph.ErrNoGoModFound` or `rewrite.ErrPlanContentMismatch`.
2. The API transport edge maps those sentinels to `connect.Error` values in `internal/<domain>/service_error_mapping.go`.
3. The UI receives `ConnectError`, maps `ConnectError.code` to an `errors.<code>` i18n key with `ui/src/lib/errorMessage.ts`, and renders localized copy.

The CLI uses the same `connect.Error` values through cli-core. Human output is English for now; future CLI i18n should use the same code names as the UI catalog instead of string-matching messages.

## Sentinel Mapping

| Domain | Sentinel | Connect code | UI i18n key | Notes |
|---|---|---|---|---|
| graph | `ErrNoGoModFound` | `invalid_argument` | `errors.no_go_mod_found` | No `go.mod` discoverable in the input path. |
| graph | `ErrMultipleGoModFiles` | `invalid_argument` | `errors.multiple_go_mod_files` | Ambiguous — multiple `go.mod` files at the same level. Operator must point at one. |
| graph | `ErrWorkspaceUnsupported` | `invalid_argument` | `errors.workspace_unsupported` | `go.work` detected. P2 capability (OT-P2-005). |
| graph | `ErrPathUnreadable` | `failed_precondition` | `errors.path_unreadable` | Filesystem error on the input path. |
| graph | `ErrProjectLoadFailed` | `internal` | `errors.project_load_failed` | `packages.Load` returned a catastrophic error (not per-file warnings). |
| rewrite | `ErrInvalidOperations` | `invalid_argument` | `errors.invalid_operations` | Duplicate ops, cycles, or self-moves in the operation list. |
| rewrite | `ErrFileNotFound` | `failed_precondition` | `errors.file_not_found` | A `FileMove.from` does not exist. |
| rewrite | `ErrImportPathUnused` | `failed_precondition` | `errors.import_path_unused` | An `ImportRewrite.old_path` is not imported anywhere in the module. |
| rewrite | `ErrPathTraversal` | `permission_denied` | `errors.path_traversal` | A `FileMove.to` or `ImportRewrite.new_path` resolves outside the module root. |
| rewrite | `ErrPlanExpiredOrInvalid` | `not_found` | `errors.plan_expired` | `plan_id` not found in the in-process registry (TTL expired or never existed). |
| rewrite | `ErrPlanContentMismatch` | `aborted` | `errors.plan_content_mismatch` | Source code changed between plan and apply; recompute mismatch. |
| rewrite | `ErrApplyPartial` | `aborted` | `errors.apply_partial` | Mid-apply failure; disk left in mid-state. Response carries `failed_op` and `completed_ops`. |
| Unknown service/repository error | (any) | `internal` | `errors.internal` | Fallback for unmapped errors. |

When you add a domain, keep the mapping file next to that domain's service layer. The handler should call the mapper instead of switching on domain error types inline.

## Multipart REST Exceptions

Go Code Graph has **no multipart REST endpoints**. All operations are pure proto-RPC. There is no opaque-bytes upload path in v1.

If a future capability needs file upload (e.g. uploading a tarball of source for remote analysis — out of scope today), use the standard multipart pattern documented in the react-vite template's `notes` example: REST multipart for bytes, proto metadata response, `internal/httpx.WriteError` for errors.

Use this split:

- Connect-RPC for messages that can be described by proto.
- REST multipart for file bytes (not currently used).
- Proto metadata responses for REST upload results.

Do not introduce a second general JSON transport for internal scenario calls. If the payload is structured and Vrooli-owned, add a proto service method.
