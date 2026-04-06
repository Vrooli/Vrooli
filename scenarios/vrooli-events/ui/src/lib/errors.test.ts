// [REQ:REQ-UI-001] Error categorization maps errors to recovery paths
import { describe, it, expect } from "vitest";
import { categorizeError } from "./errors";

describe("categorizeError", () => {
  it("classifies fetch failures as connection errors", () => {
    const result = categorizeError(new Error("Failed to fetch"));
    expect(result.category).toBe("connection");
    expect(result.userMessage).toBe("Cannot reach the server");
    expect(result.guidance).toContain("API is running");
  });

  it("classifies NetworkError as connection errors", () => {
    const result = categorizeError(new Error("NetworkError when attempting to fetch"));
    expect(result.category).toBe("connection");
  });

  it("classifies load failed as connection errors", () => {
    const result = categorizeError(new Error("Load failed"));
    expect(result.category).toBe("connection");
  });

  it("classifies 503 as server unavailable", () => {
    const result = categorizeError(new Error("API health check failed: 503"));
    expect(result.category).toBe("server");
    expect(result.userMessage).toBe("Service temporarily unavailable");
  });

  it("classifies 500 as server error", () => {
    const result = categorizeError(new Error("Event query failed: 500"));
    expect(result.category).toBe("server");
    expect(result.userMessage).toBe("Server error");
  });

  it("classifies 400 as validation error", () => {
    const result = categorizeError(new Error("Bad request: 400"));
    expect(result.category).toBe("validation");
    expect(result.userMessage).toBe("Invalid request");
    expect(result.guidance).toContain("filters");
  });

  it("falls back to unknown for unrecognized errors", () => {
    const result = categorizeError(new Error("cosmic rays"));
    expect(result.category).toBe("unknown");
    expect(result.userMessage).toBe("Something went wrong");
  });

  it("returns guidance for every category", () => {
    const errors = [
      new Error("Failed to fetch"),
      new Error("503"),
      new Error("500"),
      new Error("400"),
      new Error("unknown"),
    ];
    for (const err of errors) {
      const result = categorizeError(err);
      expect(result.guidance.length).toBeGreaterThan(0);
    }
  });
});
