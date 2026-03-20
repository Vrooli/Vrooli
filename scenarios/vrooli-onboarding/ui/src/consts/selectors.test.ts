import { describe, expect, it } from "vitest";
import { selectors, selectorsManifest } from "./selectors";

// [REQ:ONBOARD-SMART-FLOW-002] Selector registry validation

// The merged selector tree produces complex conditional types that TypeScript
// cannot fully resolve statically. Dynamic selector functions are present at
// runtime but typed as `undefined` at compile time. We use a runtime guard to
// extract them without dangerous cast patterns.
type AnyFn = (...args: unknown[]) => string;

/** Runtime extraction of dynamic selectors that TypeScript cannot statically resolve. */
function getDynamic(value: unknown): AnyFn {
  if (typeof value !== "function") {
    throw new Error(`Expected dynamic selector function, got ${typeof value}`);
  }
  return value;
}

describe("selectors registry", () => {
  describe("literal selectors", () => {
    it("exposes app-level selectors as plain strings", () => {
      expect(selectors.app.skipToContent).toBe("skip-to-content");
      expect(selectors.app.nav).toBe("app-nav");
    });

    it("exposes nav selectors", () => {
      expect(selectors.nav.wizard).toBe("nav-wizard");
      expect(selectors.nav.dashboard).toBe("nav-dashboard");
      expect(selectors.nav.glossary).toBe("nav-glossary");
      expect(selectors.nav.wizardBadge).toBe("nav-wizard-badge");
    });

    it("exposes wizard selectors", () => {
      expect(selectors.wizard.shell).toBe("wizard-shell");
      expect(selectors.wizard.prev).toBe("wizard-prev");
      expect(selectors.wizard.next).toBe("wizard-next");
      expect(selectors.wizard.welcome).toBe("step-welcome");
      expect(selectors.wizard.selectResources).toBe("step-select-resources");
      expect(selectors.wizard.review).toBe("step-review");
      expect(selectors.wizard.complete).toBe("step-complete");
    });

    it("exposes dashboard selectors", () => {
      expect(selectors.dashboard.root).toBe("health-dashboard");
      expect(selectors.dashboard.summary).toBe("health-summary");
      expect(selectors.dashboard.grid).toBe("health-grid");
    });

    it("exposes glossary selectors", () => {
      expect(selectors.glossary.root).toBe("glossary-panel");
      expect(selectors.glossary.search).toBe("glossary-search");
      expect(selectors.glossary.list).toBe("glossary-list");
    });
  });

  describe("dynamic selectors", () => {
    it("wizard.stepIndicator returns testId with index", () => {
      const fn = getDynamic(selectors.wizard.stepIndicator);
      expect(fn({ index: 0 })).toBe("step-indicator-0");
      expect(fn({ index: 3 })).toBe("step-indicator-3");
    });

    it("wizard.resourceCard returns testId with name", () => {
      const fn = getDynamic(selectors.wizard.resourceCard);
      expect(fn({ name: "postgres" })).toBe("resource-card-postgres");
      expect(fn({ name: "redis" })).toBe("resource-card-redis");
    });

    it("wizard.categoryToggle returns testId with category", () => {
      const fn = getDynamic(selectors.wizard.categoryToggle);
      expect(fn({ category: "databases" })).toBe("category-toggle-databases");
    });

    it("wizard.removeResource returns testId with name", () => {
      const fn = getDynamic(selectors.wizard.removeResource);
      expect(fn({ name: "redis" })).toBe("remove-resource-redis");
    });

    it("dashboard.healthCard returns testId with name", () => {
      const fn = getDynamic(selectors.dashboard.healthCard);
      expect(fn({ name: "postgres" })).toBe("health-card-postgres");
    });

    it("dashboard.statusIndicator returns testId with name", () => {
      const fn = getDynamic(selectors.dashboard.statusIndicator);
      expect(fn({ name: "redis" })).toBe("status-indicator-redis");
    });

    it("glossary.entry returns testId with term", () => {
      const fn = getDynamic(selectors.glossary.entry);
      expect(fn({ term: "container" })).toBe("glossary-entry-container");
    });

    it("throws for missing required parameter", () => {
      const fn = getDynamic(selectors.wizard.stepIndicator);
      expect(() => fn({})).toThrow(/missing parameter/i);
    });

    it("throws for wrong parameter type (expects number)", () => {
      const fn = getDynamic(selectors.wizard.stepIndicator);
      expect(() => fn({ index: "abc" })).toThrow(/must be numeric/i);
    });

    it("throws for unknown extra parameters", () => {
      const fn = getDynamic(selectors.wizard.stepIndicator);
      expect(() => fn({ index: 0, extra: "nope" })).toThrow(/unknown parameter/i);
    });
  });

  describe("manifest", () => {
    it("contains literal selectors with testId and CSS selector", () => {
      const entry = selectorsManifest.selectors["app.skipToContent"];
      expect(entry).toBeDefined();
      expect(entry?.testId).toBe("skip-to-content");
      expect(entry?.selector).toBe('[data-testid="skip-to-content"]');
    });

    it("flattens nested literal selectors with dot notation", () => {
      expect(selectorsManifest.selectors["wizard.shell"]).toBeDefined();
      expect(selectorsManifest.selectors["dashboard.root"]).toBeDefined();
      expect(selectorsManifest.selectors["glossary.search"]).toBeDefined();
    });

    it("contains dynamic selectors with description and params", () => {
      const entry = selectorsManifest.dynamicSelectors["wizard.stepIndicator"];
      expect(entry).toBeDefined();
      expect(entry?.description).toBe("Step indicator circle by index");
      expect(entry?.params).toEqual([{ name: "index", type: "number" }]);
      expect(entry?.testIdPattern).toBe("step-indicator-${index}");
    });

    it("maps all literal selector keys", () => {
      const keys = Object.keys(selectorsManifest.selectors);
      expect(keys.length).toBeGreaterThan(10);
      expect(keys).toContain("nav.wizard");
      expect(keys).toContain("wizard.progressBar");
      expect(keys).toContain("glossary.empty");
    });

    it("maps all dynamic selector keys", () => {
      const dynamicKeys = Object.keys(selectorsManifest.dynamicSelectors);
      expect(dynamicKeys).toContain("wizard.stepIndicator");
      expect(dynamicKeys).toContain("wizard.resourceCard");
      expect(dynamicKeys).toContain("dashboard.healthCard");
      expect(dynamicKeys).toContain("glossary.entry");
    });
  });
});
