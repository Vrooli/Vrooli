import { describe, expect, it } from "vitest";
import { selectors, selectorsManifest } from "./selectors";

// [REQ:ONBOARD-SMART-FLOW-002] Selector registry validation

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
      expect(selectors.wizard.scenarios).toBe("step-select-scenarios");
      expect(selectors.wizard.resources).toBe("step-derived-resources");
      expect(selectors.wizard.readiness).toBe("step-readiness");
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
    it("wizard.scenarioCard returns testId with name", () => {
      const fn = selectors.wizard.scenarioCard;
      expect(fn({ name: "writer" })).toBe("scenario-card-writer");
    });

    it("dashboard.healthCard returns testId with name", () => {
      const fn = selectors.dashboard.healthCard;
      expect(fn({ name: "postgres" })).toBe("health-card-postgres");
    });

    it("dashboard.statusIndicator returns testId with name", () => {
      const fn = selectors.dashboard.statusIndicator;
      expect(fn({ name: "redis" })).toBe("status-indicator-redis");
    });

    it("glossary.entry returns testId with term", () => {
      const fn = selectors.glossary.entry;
      expect(fn({ term: "container" })).toBe("glossary-entry-container");
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
      const entry = selectorsManifest.dynamicSelectors["wizard.scenarioCard"];
      expect(entry).toBeDefined();
      expect(entry?.description).toBe("Scenario card by scenario name");
      expect(entry?.params).toEqual([{ name: "name", type: "string" }]);
      expect(entry?.testIdPattern).toBe("scenario-card-${name}");
    });

    it("maps all literal selector keys", () => {
      const keys = Object.keys(selectorsManifest.selectors);
      expect(keys.length).toBeGreaterThan(10);
      expect(keys).toContain("nav.wizard");
      expect(keys).toContain("wizard.scenarios");
      expect(keys).toContain("glossary.empty");
    });

    it("maps all dynamic selector keys", () => {
      const dynamicKeys = Object.keys(selectorsManifest.dynamicSelectors);
      expect(dynamicKeys).toContain("wizard.scenarioCard");
      expect(dynamicKeys).toContain("dashboard.healthCard");
      expect(dynamicKeys).toContain("glossary.entry");
    });
  });
});
