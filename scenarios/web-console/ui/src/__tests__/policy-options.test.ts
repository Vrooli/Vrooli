import { describe, expect, it } from "vitest";
import {
  parsePolicySelection,
  policyKey,
  POLICY_OPTIONS,
} from "../consts/policy-options";

describe("policy-options seam", () => {
  it("policyKey encodes never mode", () => {
    expect(policyKey("never")).toBe("never");
  });

  it("policyKey encodes preset mode", () => {
    expect(policyKey("preset", "1h")).toBe("preset:1h");
  });

  it("parsePolicySelection parses never mode", () => {
    expect(parsePolicySelection("never")).toEqual({ mode: "never" });
  });

  it("parsePolicySelection parses preset mode with duration", () => {
    expect(parsePolicySelection("preset:8h")).toEqual({
      mode: "preset",
      duration: "8h",
    });
  });

  it("parsePolicySelection returns null for invalid values", () => {
    expect(parsePolicySelection("preset")).toBeNull();
    expect(parsePolicySelection("bad-value")).toBeNull();
    expect(parsePolicySelection("custom:4h")).toBeNull();
  });

  it("all POLICY_OPTIONS values round-trip through parser", () => {
    for (const option of POLICY_OPTIONS) {
      const key = policyKey(option.mode, option.duration);
      const parsed = parsePolicySelection(key);
      expect(parsed).not.toBeNull();
      expect(parsed).toEqual({
        mode: option.mode,
        duration: option.duration,
      });
    }
  });
});
