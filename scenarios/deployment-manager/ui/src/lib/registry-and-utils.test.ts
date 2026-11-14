import { describe, expect, it } from "vitest";
import { selectorsManifest } from "../consts/selectors";
import { buildScenarioOptions, getFirstMatchingTierKey, getTierById, getTierByKey, getFitnessColor, isTierKey } from "./tiers";
import { cn, getErrorMessage } from "./utils";

describe("selector registry and UI utilities", () => {
  it("exposes the literal selector contract and manifest", () => {
    expect(selectorsManifest.selectors).toEqual(expect.objectContaining({
      "layout.rootContainer": { testId: "app-root", selector: '[data-testid="app-root"]' },
      "profileForm.createSubmit": { testId: "create-profile-submit", selector: '[data-testid="create-profile-submit"]' },
    }));
  });

  it("handles tier lookup, scoring, and scenario suggestions", () => {
    expect(getTierByKey("desktop")?.id).toBe(2);
    expect(getTierById(4)?.key).toBe("saas");
    expect(getFirstMatchingTierKey([99, 3])).toBe("mobile");
    expect(getFirstMatchingTierKey([])).toBe("desktop");
    expect(getFitnessColor(90)).toContain("green");
    expect(getFitnessColor(60)).toContain("yellow");
    expect(getFitnessColor(10)).toContain("orange");
    expect(getFitnessColor(0)).toContain("red");
    expect(isTierKey("enterprise")).toBe(true);
    expect(isTierKey("unknown")).toBe(false);
    expect(buildScenarioOptions(undefined, "new-scenario")[0]).toBe("new-scenario");
    expect(buildScenarioOptions(undefined, "PICKER")).toContain("picker-wheel");
  });

  it("merges class names and safely formats unknown errors", () => {
    expect(cn("rounded", "rounded-lg", undefined)).toContain("rounded-lg");
    expect(getErrorMessage(new Error("broken"))).toBe("broken");
    expect(getErrorMessage({ reason: "broken" })).toBe("[object Object]");
  });
});
