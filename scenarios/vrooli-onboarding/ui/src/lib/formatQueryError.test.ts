// [REQ:REQ-P1-002] Error display utilities
import { formatQueryError } from "./formatQueryError";

describe("formatQueryError", () => {
  it("returns null for falsy error", () => {
    expect(formatQueryError(null, "fallback")).toBeNull();
    expect(formatQueryError(undefined, "fallback")).toBeNull();
    expect(formatQueryError("", "fallback")).toBeNull();
    expect(formatQueryError(0, "fallback")).toBeNull();
  });

  it("extracts message from Error instances", () => {
    expect(formatQueryError(new Error("Network failure"), "fallback")).toBe("Network failure");
  });

  it("returns fallback for non-Error truthy values", () => {
    expect(formatQueryError("some string", "fallback")).toBe("fallback");
    expect(formatQueryError(42, "fallback")).toBe("fallback");
    expect(formatQueryError({ code: 500 }, "fallback")).toBe("fallback");
    expect(formatQueryError(true, "fallback")).toBe("fallback");
  });

  it("handles Error subclasses", () => {
    expect(formatQueryError(new TypeError("type issue"), "fallback")).toBe("type issue");
    expect(formatQueryError(new RangeError("out of range"), "fallback")).toBe("out of range");
  });
});
