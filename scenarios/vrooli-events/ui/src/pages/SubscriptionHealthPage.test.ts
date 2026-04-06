// @vitest-environment node
import { describe, expect, it } from "vitest";
import { selectorsManifest } from "../consts/selectors";

// [REQ:REQ-UI-010] Subscription health detail selectors and structure
describe("subscription health detail selectors", () => {
  it("subscriptions page selector exists for navigation", () => {
    expect(selectorsManifest.selectors["subscriptions.page"]).toBeDefined();
    expect(selectorsManifest.selectors["subscriptions.page"]?.testId).toBe("subscriptions-page");
  });

  it("subscriptions table selector exists for row click-through to health", () => {
    expect(selectorsManifest.selectors["subscriptions.table"]).toBeDefined();
  });

  it("all subscription selectors have valid test IDs", () => {
    const keys = Object.keys(selectorsManifest.selectors).filter((k) => k.startsWith("subscriptions."));
    for (const key of keys) {
      const sel = selectorsManifest.selectors[key];
      expect(typeof sel?.testId).toBe("string");
      expect(sel?.testId.length).toBeGreaterThan(0);
    }
  });

  it("subscriptions table testId follows naming convention", () => {
    const table = selectorsManifest.selectors["subscriptions.table"];
    expect(table?.selector).toBe('[data-testid="subscriptions-table"]');
  });
});
