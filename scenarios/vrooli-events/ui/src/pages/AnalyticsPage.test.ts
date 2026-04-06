// @vitest-environment node
import { describe, expect, it } from "vitest";
import { selectorsManifest } from "../consts/selectors";

// [REQ:REQ-UI-013] System health overview — selector and stat card coverage
describe("system health overview selectors", () => {
  it("includes system health selectors", () => {
    expect(selectorsManifest.selectors["analytics.systemStatus"]).toBeDefined();
    expect(selectorsManifest.selectors["analytics.subscribers"]).toBeDefined();
    expect(selectorsManifest.selectors["analytics.storeSize"]).toBeDefined();
    expect(selectorsManifest.selectors["analytics.totalEvents"]).toBeDefined();
  });

  it("system status selector produces correct data-testid format", () => {
    const status = selectorsManifest.selectors["analytics.systemStatus"];
    expect(status?.selector).toBe('[data-testid="analytics-system-status"]');
  });

  it("all four analytics stat cards have selectors", () => {
    const keys = Object.keys(selectorsManifest.selectors).filter((k) => k.startsWith("analytics."));
    expect(keys.length).toBeGreaterThanOrEqual(4);
  });

  it("subscriber count selector is addressable", () => {
    const subs = selectorsManifest.selectors["analytics.subscribers"];
    expect(subs?.testId).toBe("analytics-subscribers");
  });
});
