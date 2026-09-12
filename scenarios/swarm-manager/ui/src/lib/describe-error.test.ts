/**
 * describeError is the single derivation every async surface uses to turn a
 * thrown value into words. These tests pin the precedence rules, because the
 * wrong choice either hides the server's explanation or leaks internals.
 */

import { describe, expect, it } from "vitest";
import { ApiError } from "./api-client";
import { describeError, errorMessageOf } from "./error-utils";

describe("describeError", () => {
  it("prefers the server's own explanation on a 4xx", () => {
    // ApiError.userMessage collapses this to "The request failed. Please try
    // again.", which tells the operator nothing about what to fix.
    const error = new ApiError("http", "milestone has no acceptance criteria", { status: 400 });
    expect(describeError(error).message).toBe("milestone has no acceptance criteria");
  });

  it("keeps entity refs intact in server messages", () => {
    // sanitizeErrorMessage would rewrite "/release-1" to "[PATH]" and destroy
    // the most actionable part of the sentence.
    const error = new ApiError("http", "milestone not found: goal/release-1", { status: 404 });
    expect(describeError(error).message).toContain("goal/release-1");
  });

  it("uses the friendly message for 401 and 403 rather than the server body", () => {
    const forbidden = new ApiError("http", "token lacks scope backlog:write", { status: 403 });
    expect(describeError(forbidden).message).toBe("You don't have permission to access this resource.");

    const expired = new ApiError("http", "jwt expired at 1730000000", { status: 401 });
    expect(describeError(expired).message).toBe("Your session has expired. Please refresh the page.");
  });

  it("ignores placeholder bodies the client synthesized", () => {
    const empty = new ApiError("http", "Request failed with status 400", { status: 400 });
    expect(describeError(empty).message).toBe("The request failed. Please try again.");

    const html = new ApiError("http", "The server returned an HTML error page with status 400", { status: 400 });
    expect(describeError(html).message).toBe("The request failed. Please try again.");
  });

  it("does not show a raw 5xx body to the operator", () => {
    const error = new ApiError("http", "panic: runtime error: index out of range", { status: 500 });
    expect(describeError(error).message).toBe("The server encountered an error. Please try again later.");
  });

  it("marks transport failures retryable and validation failures not", () => {
    expect(describeError(new ApiError("network", "Network request failed")).canRetry).toBe(true);
    expect(describeError(new ApiError("timeout", "Request timed out")).canRetry).toBe(true);
    expect(describeError(new ApiError("http", "bad input", { status: 422 })).canRetry).toBe(false);
  });

  it("carries the machine-readable code through", () => {
    const error = new ApiError("http", "plan is stale", { status: 409, code: "plan_stale" });
    const described = describeError(error);
    expect(described.code).toBe("plan_stale");
    expect(described.status).toBe(409);
  });

  it("sanitizes non-API errors, which can carry internal paths", () => {
    const described = describeError(new Error("failed reading /home/op/.secrets/token"));
    expect(described.message).not.toContain("/home/op");
    expect(described.message).toContain("[PATH]");
  });

  it("elides an unreasonably long server message", () => {
    const error = new ApiError("http", "x".repeat(500), { status: 400 });
    const { message } = describeError(error);
    expect(message.length).toBeLessThanOrEqual(240);
    expect(message.endsWith("…")).toBe(true);
  });

  it("handles values that are not Errors at all", () => {
    // Rejecting with a string or undefined is rare but must not crash the
    // renderer that displays the result.
    expect(describeError("boom").message).toBe("An unexpected error occurred.");
    expect(describeError(undefined).message).toBe("An unexpected error occurred.");
    expect(describeError(null).message).toBe("An unexpected error occurred.");
  });

  it("always yields recovery guidance", () => {
    for (const error of [new ApiError("network", "x"), new Error("y"), "z"]) {
      expect(describeError(error).recovery).toBeTruthy();
    }
  });
});

describe("errorMessageOf", () => {
  it("returns the described sentence", () => {
    expect(errorMessageOf(new ApiError("http", "nope", { status: 400 }))).toBe("nope");
  });

  it("uses the caller's fallback for a non-Error throw", () => {
    expect(errorMessageOf("boom", "Unable to save follow-up.")).toBe("Unable to save follow-up.");
  });

  it("does not let the fallback override a real message", () => {
    expect(errorMessageOf(new Error("disk full"), "Unable to save.")).toBe("disk full");
  });
});
