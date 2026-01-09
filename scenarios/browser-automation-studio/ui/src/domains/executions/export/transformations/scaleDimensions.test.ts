import { describe, expect, it } from "vitest";
import {
  calculateScaleFactors,
  DEFAULT_DIMENSIONS,
  extractCanvasDimensions,
  parseDimensionInput,
  PRESET_DIMENSIONS,
  resolveDimensionPreset,
  scaleDimensions,
  scaleFrameRect,
} from "./scaleDimensions";

describe("scaleDimensions", () => {
  describe("calculateScaleFactors", () => {
    it("calculates correct scale factors for upscaling", () => {
      const result = calculateScaleFactors(
        { width: 640, height: 480 },
        { width: 1280, height: 960 },
      );
      expect(result.scaleX).toBe(2);
      expect(result.scaleY).toBe(2);
    });

    it("calculates correct scale factors for downscaling", () => {
      const result = calculateScaleFactors(
        { width: 1920, height: 1080 },
        { width: 960, height: 540 },
      );
      expect(result.scaleX).toBe(0.5);
      expect(result.scaleY).toBe(0.5);
    });

    it("handles non-uniform scaling", () => {
      const result = calculateScaleFactors(
        { width: 800, height: 600 },
        { width: 1600, height: 900 },
      );
      expect(result.scaleX).toBe(2);
      expect(result.scaleY).toBe(1.5);
    });

    it("uses default dimensions for zero source", () => {
      const result = calculateScaleFactors(
        { width: 0, height: 0 },
        { width: 1280, height: 720 },
      );
      expect(result.scaleX).toBe(1280 / DEFAULT_DIMENSIONS.width);
      expect(result.scaleY).toBe(720 / DEFAULT_DIMENSIONS.height);
    });
  });

  describe("scaleDimensions", () => {
    it("scales dimensions by given factors", () => {
      const result = scaleDimensions(
        { width: 100, height: 50 },
        { scaleX: 2, scaleY: 3 },
        { width: 80, height: 40 },
      );
      expect(result.width).toBe(200);
      expect(result.height).toBe(150);
    });

    it("uses fallback for missing dimensions", () => {
      const result = scaleDimensions(
        undefined,
        { scaleX: 2, scaleY: 2 },
        { width: 100, height: 50 },
      );
      expect(result.width).toBe(200);
      expect(result.height).toBe(100);
    });

    it("uses fallback for zero dimensions", () => {
      const result = scaleDimensions(
        { width: 0, height: 0 },
        { scaleX: 2, scaleY: 2 },
        { width: 100, height: 50 },
      );
      expect(result.width).toBe(200);
      expect(result.height).toBe(100);
    });

    it("rounds to integers", () => {
      const result = scaleDimensions(
        { width: 100, height: 100 },
        { scaleX: 1.5, scaleY: 1.5 },
        { width: 100, height: 100 },
      );
      expect(result.width).toBe(150);
      expect(result.height).toBe(150);
    });
  });

  describe("scaleFrameRect", () => {
    it("scales position and dimensions", () => {
      const result = scaleFrameRect(
        { x: 10, y: 20, width: 100, height: 50, radius: 8 },
        { scaleX: 2, scaleY: 2 },
        { width: 100, height: 50 },
      );
      expect(result.x).toBe(20);
      expect(result.y).toBe(40);
      expect(result.width).toBe(200);
      expect(result.height).toBe(100);
      expect(result.radius).toBe(8);
    });

    it("uses fallback dimensions for null rect", () => {
      const result = scaleFrameRect(
        null,
        { scaleX: 2, scaleY: 2 },
        { width: 100, height: 50 },
        12,
      );
      expect(result.x).toBe(0);
      expect(result.y).toBe(0);
      expect(result.width).toBe(200);
      expect(result.height).toBe(100);
      expect(result.radius).toBe(12);
    });

    it("uses default radius when not specified", () => {
      const result = scaleFrameRect(
        { x: 0, y: 0, width: 100, height: 50 },
        { scaleX: 1, scaleY: 1 },
        { width: 100, height: 50 },
      );
      expect(result.radius).toBe(24);
    });
  });

  describe("resolveDimensionPreset", () => {
    const specDims = { width: 800, height: 600 };
    const customDims = { width: 1600, height: 900 };

    it("returns spec dimensions for 'spec' preset", () => {
      const result = resolveDimensionPreset("spec", specDims, customDims);
      expect(result).toEqual(specDims);
    });

    it("returns 1080p preset dimensions", () => {
      const result = resolveDimensionPreset("1080p", specDims, customDims);
      expect(result).toEqual(PRESET_DIMENSIONS["1080p"]);
    });

    it("returns 720p preset dimensions", () => {
      const result = resolveDimensionPreset("720p", specDims, customDims);
      expect(result).toEqual(PRESET_DIMENSIONS["720p"]);
    });

    it("returns custom dimensions for 'custom' preset", () => {
      const result = resolveDimensionPreset("custom", specDims, customDims);
      expect(result).toEqual(customDims);
    });

    it("uses default for invalid custom dimensions", () => {
      const result = resolveDimensionPreset(
        "custom",
        specDims,
        { width: 0, height: -10 },
      );
      expect(result).toEqual(DEFAULT_DIMENSIONS);
    });

    it("uses default for invalid spec dimensions", () => {
      const result = resolveDimensionPreset(
        "spec",
        { width: 0, height: 0 },
        customDims,
      );
      expect(result).toEqual(DEFAULT_DIMENSIONS);
    });
  });

  describe("extractCanvasDimensions", () => {
    it("extracts canvas dimensions when present", () => {
      const result = extractCanvasDimensions({
        canvas: { width: 1920, height: 1080 },
        viewport: { width: 1280, height: 720 },
      });
      expect(result).toEqual({ width: 1920, height: 1080 });
    });

    it("falls back to viewport when canvas is missing", () => {
      const result = extractCanvasDimensions({
        viewport: { width: 1280, height: 720 },
      });
      expect(result).toEqual({ width: 1280, height: 720 });
    });

    it("returns defaults for null/undefined presentation", () => {
      expect(extractCanvasDimensions(null)).toEqual(DEFAULT_DIMENSIONS);
      expect(extractCanvasDimensions(undefined)).toEqual(DEFAULT_DIMENSIONS);
    });

    it("returns defaults for empty presentation", () => {
      const result = extractCanvasDimensions({});
      expect(result).toEqual(DEFAULT_DIMENSIONS);
    });
  });

  describe("parseDimensionInput", () => {
    it("parses valid integer strings", () => {
      expect(parseDimensionInput("1920", 100)).toBe(1920);
      expect(parseDimensionInput("720", 100)).toBe(720);
    });

    it("returns fallback for empty string", () => {
      expect(parseDimensionInput("", 100)).toBe(100);
    });

    it("returns fallback for invalid input", () => {
      expect(parseDimensionInput("abc", 100)).toBe(100);
      expect(parseDimensionInput("12.5px", 100)).toBe(12); // parseInt behavior
    });

    it("returns fallback for zero or negative", () => {
      expect(parseDimensionInput("0", 100)).toBe(100);
      expect(parseDimensionInput("-50", 100)).toBe(100);
    });
  });
});
