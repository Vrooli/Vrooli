// DOC: docs/internal/ERROR_SEMANTICS.md#structured-error-type-typescript

/** API error response shape from the backend. */
export interface APIErrorBody {
  error: string;
  code?: string;
  category?: "validation" | "resource_limit" | "dependency" | "internal";
  recovery?: string;
  retry?: boolean;
  upgrade_path?: string;
}

/**
 * An API error with structured fields for programmatic handling.
 * UI components can inspect `code`, `category`, and `recovery` to
 * choose the right recovery action instead of just showing a message.
 */
export class APIError extends Error {
  readonly code: string;
  readonly category: string;
  readonly recovery: string;
  readonly retry: boolean;
  readonly status: number;
  readonly upgradePath: string;

  constructor(status: number, body: APIErrorBody) {
    super(body.error);
    this.name = "APIError";
    this.status = status;
    this.code = body.code ?? "unknown";
    this.category = body.category ?? "internal";
    this.recovery = body.recovery ?? "";
    this.retry = body.retry ?? false;
    this.upgradePath = body.upgrade_path ?? "";
  }
}

/**
 * Extract a structured error from an HTTP response body. Used by
 * non-Connect HTTP endpoints (uploads, health). Connect-RPC clients
 * surface their own ConnectError; convert via `toErrorInfo`.
 */
export async function extractAPIError(res: Response, fallback: string): Promise<APIError> {
  try {
    const body = (await res.json()) as APIErrorBody;
    if (body.error) return new APIError(res.status, body);
  } catch {
    // Response body was not JSON — use fallback
  }
  return new APIError(res.status, {
    error: `${fallback}: ${res.status}`,
    code: "unknown",
    category: "internal",
    recovery: "Try again. If the problem persists, check server logs.",
    retry: true,
  });
}

/**
 * Structured error info for display. Produced by `toErrorInfo` and
 * consumed by error display components (ErrorBanner, useSessionManager).
 */
export interface ErrorInfo {
  message: string;
  recovery?: string;
  retry?: boolean;
  upgradePath?: string;
}

/**
 * Convert an unknown caught error into a plain object with message,
 * recovery hint, and retry flag. Extracts structured fields from
 * APIError; falls back to generic message for other error types.
 */
export function toErrorInfo(err: unknown): ErrorInfo {
  const message = err instanceof Error ? err.message : "Unknown error";
  if (err instanceof APIError) {
    return { message, recovery: err.recovery || undefined, retry: err.retry || undefined, upgradePath: err.upgradePath || undefined };
  }
  return { message };
}
