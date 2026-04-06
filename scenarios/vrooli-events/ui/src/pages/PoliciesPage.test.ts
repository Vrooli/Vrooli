// @vitest-environment node
import { describe, expect, it } from "vitest";
import { selectorsManifest } from "../consts/selectors";

// [REQ:REQ-UI-006] Policy management — rule list selectors and structure
describe("policy management selectors", () => {
  it("includes policies page selector", () => {
    expect(selectorsManifest.selectors["policies.page"]).toBeDefined();
  });

  it("includes policies table selector", () => {
    expect(selectorsManifest.selectors["policies.table"]).toBeDefined();
    expect(selectorsManifest.selectors["policies.table"]?.testId).toBe("policies-table");
  });

  it("includes new rule button selector", () => {
    expect(selectorsManifest.selectors["policies.newButton"]).toBeDefined();
    expect(selectorsManifest.selectors["policies.newButton"]?.testId).toBe("policies-new-button");
  });

  it("includes new rule form selector", () => {
    expect(selectorsManifest.selectors["policies.newForm"]).toBeDefined();
    expect(selectorsManifest.selectors["policies.newForm"]?.testId).toBe("policies-new-form");
  });

  it("all policies selectors produce correct data-testid format", () => {
    const keys = Object.keys(selectorsManifest.selectors).filter((k) => k.startsWith("policies."));
    expect(keys.length).toBeGreaterThanOrEqual(4);
    for (const key of keys) {
      const sel = selectorsManifest.selectors[key];
      expect(sel?.selector).toMatch(/^\[data-testid="/);
    }
  });
});
