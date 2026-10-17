/**
 * Presentation-entry coverage for the breadth ops added to the discovery-driven
 * catalogs: the new generation/enhancement AI ops and the new analysis ops must
 * each resolve a label/desc/icon so the panels render them with real copy
 * (not raw op names).
 */
import { describe, expect, it } from "vitest";

import { AI_CATALOG, aiPresentation } from "./aiCatalog";
import { ANALYZE_CATALOG, analyzePresentation } from "./analyzeCatalog";

describe("aiCatalog breadth ops", () => {
  it("maps colorize and depth_map to a label/desc/icon", () => {
    for (const op of ["colorize", "depth_map"]) {
      const meta = aiPresentation(op);
      expect(meta).toBeDefined();
      expect(meta?.Icon).toBeTruthy();
      expect(meta?.labelKey).toBeTruthy();
      expect(meta?.descKey).toBeTruthy();
    }
    expect(Object.keys(AI_CATALOG)).toEqual(
      expect.arrayContaining(["colorize", "depth_map"]),
    );
  });
});

describe("analyzeCatalog breadth ops", () => {
  it("maps duplicate_detect and quality_assessment to a label/desc/icon", () => {
    for (const op of ["duplicate_detect", "quality_assessment"]) {
      const meta = analyzePresentation(op);
      expect(meta).toBeDefined();
      expect(meta?.Icon).toBeTruthy();
      expect(meta?.labelKey).toBeTruthy();
      expect(meta?.descKey).toBeTruthy();
    }
    expect(Object.keys(ANALYZE_CATALOG)).toEqual(
      expect.arrayContaining(["duplicate_detect", "quality_assessment"]),
    );
  });

  it("returns undefined for an uncatalogued analysis op", () => {
    expect(analyzePresentation("nope")).toBeUndefined();
  });
});
