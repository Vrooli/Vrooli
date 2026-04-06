// @vitest-environment node
import { describe, expect, it } from "vitest";
import { selectorsManifest } from "../consts/selectors";

// [REQ:REQ-UI-009] Subscription management selectors and structure
describe("subscription management selectors", () => {
  it("includes subscriptions page selector", () => {
    expect(selectorsManifest.selectors["subscriptions.page"]).toBeDefined();
    expect(selectorsManifest.selectors["subscriptions.page"]?.testId).toBe("subscriptions-page");
  });

  it("includes subscriptions table selector", () => {
    expect(selectorsManifest.selectors["subscriptions.table"]).toBeDefined();
    expect(selectorsManifest.selectors["subscriptions.table"]?.testId).toBe("subscriptions-table");
  });

  it("includes new subscription button selector", () => {
    expect(selectorsManifest.selectors["subscriptions.newButton"]).toBeDefined();
    expect(selectorsManifest.selectors["subscriptions.newButton"]?.testId).toBe("subscriptions-new-button");
  });

  it("includes new subscription form selector", () => {
    expect(selectorsManifest.selectors["subscriptions.newForm"]).toBeDefined();
    expect(selectorsManifest.selectors["subscriptions.newForm"]?.testId).toBe("subscriptions-new-form");
  });

  it("all subscription selectors produce correct data-testid format", () => {
    const keys = Object.keys(selectorsManifest.selectors).filter((k) => k.startsWith("subscriptions."));
    expect(keys.length).toBeGreaterThanOrEqual(4);
    for (const key of keys) {
      const sel = selectorsManifest.selectors[key];
      expect(sel?.selector).toMatch(/^\[data-testid="/);
    }
  });
});
