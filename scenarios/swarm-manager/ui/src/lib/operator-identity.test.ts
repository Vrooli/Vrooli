import { afterEach, describe, expect, it, vi } from "vitest";
import { readOperatorIdentity, rememberOperatorIdentity } from "./operator-identity";

afterEach(() => {
  window.localStorage.clear();
  vi.restoreAllMocks();
});

describe("operator identity", () => {
  it("returns an empty string before anything is remembered", () => {
    expect(readOperatorIdentity()).toBe("");
  });

  it("round-trips a remembered identity", () => {
    rememberOperatorIdentity("matthalloran8");
    expect(readOperatorIdentity()).toBe("matthalloran8");
  });

  it("trims surrounding whitespace so one operator cannot become two", () => {
    rememberOperatorIdentity("  matthalloran8  ");
    expect(readOperatorIdentity()).toBe("matthalloran8");
  });

  it("clears the stored value when given a blank identity", () => {
    rememberOperatorIdentity("someone");
    rememberOperatorIdentity("   ");
    expect(readOperatorIdentity()).toBe("");
  });

  it("returns an empty string rather than throwing when storage is blocked", () => {
    // Private browsing and hardened profiles throw on access; the decision
    // surface must still render.
    vi.spyOn(Storage.prototype, "getItem").mockImplementation(() => { throw new Error("denied"); });
    expect(readOperatorIdentity()).toBe("");
  });

  it("does not throw when a write is blocked", () => {
    vi.spyOn(Storage.prototype, "setItem").mockImplementation(() => { throw new Error("denied"); });
    expect(() => rememberOperatorIdentity("someone")).not.toThrow();
  });
});
