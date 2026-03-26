import { describe, it, expect } from "vitest";
import { ApiRequestError } from "./api";

// [REQ:BM-REQ-API-BRANDS] [REQ:BM-REQ-UI-DASHBOARD]

describe("ApiRequestError", () => {
  it("sets status and message from apiError", () => {
    const err = new ApiRequestError(400, { code: "validation", message: "Name required" });
    expect(err.status).toBe(400);
    expect(err.message).toBe("Name required");
    expect(err.name).toBe("ApiRequestError");
  });

  it("falls back to status-based message when no apiError", () => {
    const err = new ApiRequestError(500);
    expect(err.message).toBe("Request failed with status 500");
  });

  it("exposes recovery hint", () => {
    const err = new ApiRequestError(422, {
      code: "validation",
      message: "Invalid color",
      recovery: "Use hex format",
    });
    expect(err.recovery).toBe("Use hex format");
  });

  it("recovery is undefined when no apiError", () => {
    const err = new ApiRequestError(500);
    expect(err.recovery).toBeUndefined();
  });

  it("isRetryable for internal errors", () => {
    const err = new ApiRequestError(500, { code: "internal", message: "db error" });
    expect(err.isRetryable).toBe(true);
  });

  it("isRetryable for dependency errors", () => {
    const err = new ApiRequestError(502, { code: "dependency", message: "upstream down" });
    expect(err.isRetryable).toBe(true);
  });

  it("not retryable for validation errors", () => {
    const err = new ApiRequestError(400, { code: "validation", message: "bad input" });
    expect(err.isRetryable).toBe(false);
  });

  it("not retryable for not_found errors", () => {
    const err = new ApiRequestError(404, { code: "not_found", message: "missing" });
    expect(err.isRetryable).toBe(false);
  });

  it("treats 5xx without apiError as retryable", () => {
    const err = new ApiRequestError(503);
    expect(err.isRetryable).toBe(true);
  });

  it("treats 4xx without apiError as not retryable", () => {
    const err = new ApiRequestError(400);
    expect(err.isRetryable).toBe(false);
  });
});
