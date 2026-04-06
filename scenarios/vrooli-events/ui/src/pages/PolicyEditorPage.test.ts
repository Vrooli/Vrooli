// @vitest-environment node
import { describe, expect, it } from "vitest";
import { selectorsManifest } from "../consts/selectors";

// [REQ:REQ-UI-007] Policy management — rule editor
describe("policy editor page", () => {
  it("policies page selector is available for navigation to editor", () => {
    expect(selectorsManifest.selectors["policies.page"]).toBeDefined();
    expect(selectorsManifest.selectors["policies.page"]?.testId).toBe("policies-page");
  });

  it("policies table selector exists for row click-through to editor", () => {
    expect(selectorsManifest.selectors["policies.table"]).toBeDefined();
  });

  it("all selector test IDs are non-empty strings", () => {
    const keys = Object.keys(selectorsManifest.selectors).filter((k) => k.startsWith("policies."));
    for (const key of keys) {
      const sel = selectorsManifest.selectors[key];
      expect(typeof sel?.testId).toBe("string");
      expect(sel?.testId.length).toBeGreaterThan(0);
    }
  });

  it("selector format is consistent with data-testid convention", () => {
    const page = selectorsManifest.selectors["policies.page"];
    expect(page?.selector).toBe('[data-testid="policies-page"]');
  });
});
