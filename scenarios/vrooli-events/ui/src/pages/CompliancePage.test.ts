// @vitest-environment node
import { describe, expect, it } from "vitest";
import { selectorsManifest } from "../consts/selectors";

// [REQ:REQ-UI-011] Compliance & audit view selectors and structure
describe("compliance audit view selectors", () => {
  it("includes compliance page selector", () => {
    expect(selectorsManifest.selectors["compliance.page"]).toBeDefined();
    expect(selectorsManifest.selectors["compliance.page"]?.testId).toBe("compliance-page");
  });

  it("includes compliance table selector", () => {
    expect(selectorsManifest.selectors["compliance.table"]).toBeDefined();
    expect(selectorsManifest.selectors["compliance.table"]?.testId).toBe("compliance-table");
  });

  it("compliance selectors produce correct data-testid format", () => {
    const keys = Object.keys(selectorsManifest.selectors).filter((k) => k.startsWith("compliance."));
    expect(keys.length).toBeGreaterThanOrEqual(2);
    for (const key of keys) {
      const sel = selectorsManifest.selectors[key];
      expect(sel?.selector).toMatch(/^\[data-testid="/);
    }
  });

  it("compliance page testId follows naming convention", () => {
    const page = selectorsManifest.selectors["compliance.page"];
    expect(page?.selector).toBe('[data-testid="compliance-page"]');
  });

  it("compliance table testId follows naming convention", () => {
    const table = selectorsManifest.selectors["compliance.table"];
    expect(table?.selector).toBe('[data-testid="compliance-table"]');
  });
});
