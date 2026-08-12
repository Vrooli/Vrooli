// DOC: docs/internal/ERROR_SEMANTICS.md#structured-error-type-typescript
/**
 * An API error with structured fields for programmatic handling.
 * UI components can inspect `code`, `category`, and `recovery` to
 * choose the right recovery action instead of just showing a message.
 */
export class APIError extends Error {
    constructor(status, body) {
        super(body.error);
        this.name = "APIError";
        this.status = status;
        this.code = body.code ?? "unknown";
        this.category = body.category ?? "internal";
        this.recovery = body.recovery ?? "";
        this.retry = body.retry ?? false;
    }
}
/**
 * Extract a structured error from an HTTP response body. Used by
 * non-Connect HTTP endpoints (uploads, health). Connect-RPC clients
 * surface their own ConnectError; convert via `toErrorInfo`.
 */
export async function extractAPIError(res, fallback) {
    try {
        const body = (await res.json());
        if (body.error)
            return new APIError(res.status, body);
    }
    catch {
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
 * Convert an unknown caught error into a plain object with message,
 * recovery hint, and retry flag. Extracts structured fields from
 * APIError; falls back to generic message for other error types.
 */
export function toErrorInfo(err) {
    const message = err instanceof Error ? err.message : "Unknown error";
    if (err instanceof APIError) {
        return { message, recovery: err.recovery || undefined, retry: err.retry || undefined };
    }
    return { message };
}
