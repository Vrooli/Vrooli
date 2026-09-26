// DOC: docs/internal/ERROR-SEMANTICS.md
/**
 * Error categorization for user-facing display.
 *
 * Categories map to distinct recovery paths:
 * - connection: API unreachable → check server, retry
 * - server: 5xx responses → wait/retry, check logs
 * - validation: 4xx responses → fix input
 * - unknown: fallback → refresh page
 */

export type ErrorCategory = "connection" | "server" | "validation" | "unknown";

export interface CategorizedError {
  category: ErrorCategory;
  userMessage: string;
  guidance: string;
}

export function categorizeError(error: Error): CategorizedError {
  const msg = error.message.toLowerCase();

  if (msg.includes("failed to fetch") || msg.includes("networkerror") || msg.includes("load failed")) {
    return {
      category: "connection",
      userMessage: "Cannot reach the server",
      guidance: "Check that the API is running and try again.",
    };
  }
  if (msg.includes("503") || msg.includes("service unavailable")) {
    return {
      category: "server",
      userMessage: "Service temporarily unavailable",
      guidance: "The event store may be recovering. This usually resolves within a few seconds.",
    };
  }
  if (msg.includes("500") || msg.includes("internal server error")) {
    return {
      category: "server",
      userMessage: "Server error",
      guidance: "An internal error occurred. If this persists, check the API logs.",
    };
  }
  if (msg.includes("400") || msg.includes("bad request")) {
    return {
      category: "validation",
      userMessage: "Invalid request",
      guidance: "The request parameters may be incorrect. Adjust your filters and try again.",
    };
  }
  return {
    category: "unknown",
    userMessage: "Something went wrong",
    guidance: "An unexpected error occurred. Try refreshing the page.",
  };
}
