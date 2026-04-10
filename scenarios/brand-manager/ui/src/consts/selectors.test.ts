import { describe, it, expect } from "vitest";
import { selectors, selectorsManifest } from "./selectors";

// [REQ:BM-REQ-UI-DASHBOARD] [REQ:BM-REQ-SCAN-CSS] [REQ:BM-REQ-SCAN-JSON]
// [REQ:BM-REQ-AUDIT-ENDPOINT] [REQ:BM-REQ-API-STANDARDS]

// Helper to safely access nested selector branches in tests.
// The selector tree uses string index signatures, so TypeScript marks nested access
// as possibly undefined. At runtime, all branches exist (verified by these tests).
function branch<T>(value: T, name: string): NonNullable<T> {
  expect(value, `selector branch '${name}' should exist`).toBeDefined();
  // eslint-disable-next-line @typescript-eslint/no-non-null-assertion -- guarded by expect above; test-only
  return value!;
}

describe("selectors registry", () => {
  describe("literal selectors", () => {
    it("has app root selector", () => {
      expect(selectors.app.root).toBe("app-root");
    });

    it("has brand list selectors", () => {
      const bl = branch(selectors.brandList, "brandList");
      expect(bl.page).toBe("brand-list-page");
      expect(bl.createBtn).toBe("create-brand-btn");
      expect(bl.searchInput).toBe("brand-search-input");
      expect(bl.grid).toBe("brand-list-grid");
      expect(bl.empty).toBe("brand-list-empty");
    });

    it("has brand detail selectors", () => {
      const bd = branch(selectors.brandDetail, "brandDetail");
      expect(bd.page).toBe("brand-detail-page");
      expect(bd.editBtn).toBe("edit-brand-btn");
      expect(bd.deleteBtn).toBe("delete-brand-btn");
      expect(bd.colorsSection).toBe("brand-colors-section");
    });

    it("has brand form selectors", () => {
      const bf = branch(selectors.brandForm, "brandForm");
      expect(bf.page).toBe("brand-form-page");
      expect(bf.nameInput).toBe("brand-name-input");
      expect(bf.saveBtn).toBe("save-brand-btn");
    });

    it("has scanner selectors", () => {
      const sc = branch(selectors.scanner, "scanner");
      expect(sc.page).toBe("scanner-page");
      expect(sc.input).toBe("scanner-input");
      expect(sc.scanBtn).toBe("scan-btn");
      expect(sc.scanResults).toBe("scan-results");
    });

    it("has standards selectors", () => {
      const st = branch(selectors.standards, "standards");
      expect(st.page).toBe("standards-page");
      expect(st.list).toBe("standards-list");
    });

    it("has nav selectors", () => {
      const nv = branch(selectors.nav, "nav");
      expect(nv.home).toBe("nav-home");
      expect(nv.scanner).toBe("nav-scanner");
      expect(nv.standards).toBe("nav-standards");
    });
  });

  describe("dynamic selectors", () => {
    it("generates brand card selector by ID", () => {
      const fn = branch(selectors.brands, "brands").cardById;
      expect(typeof fn).toBe("function");
      expect(fn({ id: "b1" })).toBe("brand-card-b1");
    });

    it("generates standards rule selector by ID", () => {
      const fn = branch(selectors.standards, "standards").ruleById;
      expect(typeof fn).toBe("function");
      expect(fn({ id: "has-logo" })).toBe("standard-has-logo");
    });

    it("generates color swatch selector by label", () => {
      const fn = branch(selectors.colors, "colors").swatchByLabel;
      expect(fn({ label: "primary" })).toBe("color-swatch-primary");
    });

    it("generates color picker selector by key", () => {
      const fn = branch(selectors.colors, "colors").pickerByKey;
      expect(fn({ key: "accent" })).toBe("color-picker-accent");
    });

    it("generates color input selector by key", () => {
      const fn = branch(selectors.colors, "colors").inputByKey;
      expect(fn({ key: "background" })).toBe("color-input-background");
    });

    it("throws for missing parameters", () => {
      const fn = branch(selectors.brands, "brands").cardById;
      // Pass empty object to test runtime parameter validation.
      // Object.create(null) bypasses compile-time shape checking.
      const empty: { id: string } = Object.create(null);
      expect(() => fn(empty)).toThrow(/missing parameter/i);
    });
  });

  describe("manifest", () => {
    it("has literal selectors in manifest", () => {
      expect(selectorsManifest.selectors["app.root"]).toBeTruthy();
      expect(selectorsManifest.selectors["app.root"]?.testId).toBe("app-root");
      expect(selectorsManifest.selectors["app.root"]?.selector).toBe('[data-testid="app-root"]');
    });

    it("has dynamic selectors in manifest", () => {
      expect(selectorsManifest.dynamicSelectors["brands.cardById"]).toBeTruthy();
      expect(selectorsManifest.dynamicSelectors["brands.cardById"]?.params).toHaveLength(1);
      expect(selectorsManifest.dynamicSelectors["brands.cardById"]?.params?.[0]?.name).toBe("id");
    });

    it("manifest includes scanner selectors", () => {
      expect(selectorsManifest.selectors["scanner.page"]?.testId).toBe("scanner-page");
      expect(selectorsManifest.selectors["scanner.scanBtn"]?.testId).toBe("scan-btn");
    });

    it("manifest includes nav selectors", () => {
      expect(selectorsManifest.selectors["nav.home"]?.testId).toBe("nav-home");
      expect(selectorsManifest.selectors["nav.scanner"]?.testId).toBe("nav-scanner");
    });
  });
});
