// @vitest-environment node
import { describe, expect, it } from "vitest";
import { selectorsManifest } from "../consts/selectors";

// [REQ:REQ-UI-008] Circuit breaker dashboard selectors and structure
describe("circuit breaker dashboard selectors", () => {
  it("includes circuit breakers page selector", () => {
    expect(selectorsManifest.selectors["circuitBreakers.page"]).toBeDefined();
    expect(selectorsManifest.selectors["circuitBreakers.page"]?.testId).toBe("circuit-breakers-page");
  });

  it("includes circuit breakers table selector", () => {
    expect(selectorsManifest.selectors["circuitBreakers.table"]).toBeDefined();
    expect(selectorsManifest.selectors["circuitBreakers.table"]?.testId).toBe("circuit-breakers-table");
  });

  it("includes override form selector", () => {
    expect(selectorsManifest.selectors["circuitBreakers.overrideForm"]).toBeDefined();
    expect(selectorsManifest.selectors["circuitBreakers.overrideForm"]?.testId).toBe("cb-override-form");
  });

  it("all circuit breaker selectors produce correct data-testid format", () => {
    const keys = Object.keys(selectorsManifest.selectors).filter((k) => k.startsWith("circuitBreakers."));
    expect(keys.length).toBeGreaterThanOrEqual(3);
    for (const key of keys) {
      const sel = selectorsManifest.selectors[key];
      expect(sel?.selector).toMatch(/^\[data-testid="/);
    }
  });

  it("override form testId follows naming convention", () => {
    const form = selectorsManifest.selectors["circuitBreakers.overrideForm"];
    expect(form?.selector).toBe('[data-testid="cb-override-form"]');
  });
});
