import { fromJson } from "@bufbuild/protobuf";
import {
  ErrorSchema,
  type Error as TypedError,
} from "@vrooli/proto-types/scenario-to-cloud/v1/errors/errors_pb";

/**
 * Stable error codes shared with api/apierrors/codes.go. The UI branches on
 * these strings, never on message text.
 */
export const ErrorCodes = {
  invalidJson: "invalid_json",
  invalidRequest: "invalid_request",
  internal: "internal",
  unauthenticated: "unauthenticated",
  forbiddenScope: "forbidden_scope",
  forbiddenTarget: "forbidden_target",
  deploymentNotFound: "deployment_not_found",
  deploymentSelectorAmbiguous: "deployment_selector_ambiguous",
  deploymentSelectorInvalid: "deployment_selector_invalid",
  deploymentIdentityConflict: "deployment_identity_conflict",
  manifestInvalid: "manifest_invalid",
  requestKeyConflict: "request_key_conflict",
  planStale: "plan_stale",
  planDigestMismatch: "plan_digest_mismatch",
  operationConflict: "operation_conflict",
  fenceStale: "fence_stale",
  closureUnavailable: "closure_unavailable",
  unsupportedCapability: "unsupported_capability",
  unsupportedSchemaVersion: "unsupported_schema_version",
  needsInput: "needs_input",
  releaseVerificationFailed: "release_verification_failed",
  reachUnavailable: "reach_unavailable",
  healthUnknown: "health_unknown",
} as const;

/** ApiError is the client-side view of one typed failure. */
export class ApiError extends Error {
  readonly code: string;
  readonly status: number;
  readonly retryable: boolean;
  readonly typed: TypedError;

  constructor(status: number, typed: TypedError) {
    super(typed.message || typed.code);
    this.name = "ApiError";
    this.code = typed.code;
    this.status = status;
    this.retryable = typed.retryable;
    this.typed = typed;
  }

  /** Owner-provided pointer to the next thing to do, when the API gave one. */
  get nextAction() {
    return this.typed.nextAction ?? null;
  }
}

/**
 * parseApiError decodes a non-2xx body written by api/apierrors.Write through
 * the generated schema. A body that is not the typed envelope still yields a
 * stable code derived from the status so callers never branch on free text.
 */
export function parseApiError(status: number, body: string): ApiError {
  const trimmed = body.trim();
  if (trimmed !== "") {
    try {
      const envelope = JSON.parse(trimmed) as { error?: unknown };
      if (envelope && typeof envelope === "object" && envelope.error && typeof envelope.error === "object") {
        const typed = fromJson(ErrorSchema, envelope.error as never, { ignoreUnknownFields: true });
        if (typed.code !== "") {
          return new ApiError(status, typed);
        }
      }
    } catch {
      // fall through to the status-derived code
    }
  }
  const fallback = fromJson(ErrorSchema, { code: codeForStatus(status), message: trimmed === "" ? `HTTP ${status}` : trimmed });
  return new ApiError(status, fallback);
}

function codeForStatus(status: number): string {
  switch (status) {
    case 400:
      return ErrorCodes.invalidRequest;
    case 401:
      return ErrorCodes.unauthenticated;
    case 403:
      return ErrorCodes.forbiddenScope;
    case 404:
      return ErrorCodes.deploymentNotFound;
    case 409:
      return ErrorCodes.operationConflict;
    default:
      return ErrorCodes.internal;
  }
}
