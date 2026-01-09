/**
 * Tests for export configuration constants and helper functions.
 */

import { describe, expect, it } from "vitest";
import {
  buildDimensionPresetOptions,
  buildExportFileName,
  coerceMetricNumber,
  DEFAULT_EXPORT_DIMENSIONS,
  DIMENSION_PRESET_CONFIG,
  EXPORT_EXTENSIONS,
  EXPORT_FORMAT_OPTIONS,
  EXPORT_RENDER_SOURCE_OPTIONS,
  generateDefaultFileStem,
  getFormatExtension,
  isBinaryFormat,
  sanitizeFileStem,
} from "./constants";
// Format/status config is now in the presentation module
import {
  formatCapturedLabel,
  FORMAT_CONFIG,
  getFormatConfig,
  EXPORT_STATUS_CONFIG,
  getExportStatusConfig,
} from "@/domains/exports/presentation";

describe("EXPORT_EXTENSIONS", () => {
  it("has correct extensions for all formats", () => {
    expect(EXPORT_EXTENSIONS.mp4).toBe("mp4");
    expect(EXPORT_EXTENSIONS.gif).toBe("gif");
    expect(EXPORT_EXTENSIONS.json).toBe("json");
    expect(EXPORT_EXTENSIONS.html).toBe("html");
  });
});

describe("DEFAULT_EXPORT_DIMENSIONS", () => {
  it("has 720p as default", () => {
    expect(DEFAULT_EXPORT_DIMENSIONS.width).toBe(1280);
    expect(DEFAULT_EXPORT_DIMENSIONS.height).toBe(720);
  });
});

describe("DIMENSION_PRESET_CONFIG", () => {
  it("has correct 1080p dimensions", () => {
    expect(DIMENSION_PRESET_CONFIG["1080p"].width).toBe(1920);
    expect(DIMENSION_PRESET_CONFIG["1080p"].height).toBe(1080);
    expect(DIMENSION_PRESET_CONFIG["1080p"].label).toBe("1080p (Full HD)");
  });

  it("has correct 720p dimensions", () => {
    expect(DIMENSION_PRESET_CONFIG["720p"].width).toBe(1280);
    expect(DIMENSION_PRESET_CONFIG["720p"].height).toBe(720);
    expect(DIMENSION_PRESET_CONFIG["720p"].label).toBe("720p (HD)");
  });
});

describe("EXPORT_FORMAT_OPTIONS", () => {
  it("has mp4, gif, and json options", () => {
    const ids = EXPORT_FORMAT_OPTIONS.map((o) => o.id);
    expect(ids).toContain("mp4");
    expect(ids).toContain("gif");
    expect(ids).toContain("json");
  });

  it("mp4 is marked as default", () => {
    const mp4 = EXPORT_FORMAT_OPTIONS.find((o) => o.id === "mp4");
    expect(mp4?.badge).toBe("Default");
  });

  it("each option has required properties", () => {
    for (const option of EXPORT_FORMAT_OPTIONS) {
      expect(option.id).toBeTruthy();
      expect(option.label).toBeTruthy();
      expect(option.description).toBeTruthy();
      expect(option.icon).toBeTruthy();
    }
  });
});

describe("EXPORT_RENDER_SOURCE_OPTIONS", () => {
  it("has auto, recorded_video, and replay_frames options", () => {
    const ids = EXPORT_RENDER_SOURCE_OPTIONS.map((o) => o.id);
    expect(ids).toContain("auto");
    expect(ids).toContain("recorded_video");
    expect(ids).toContain("replay_frames");
  });
});

describe("FORMAT_CONFIG", () => {
  it("has config for all formats", () => {
    expect(FORMAT_CONFIG.mp4).toBeTruthy();
    expect(FORMAT_CONFIG.gif).toBeTruthy();
    expect(FORMAT_CONFIG.json).toBeTruthy();
    expect(FORMAT_CONFIG.html).toBeTruthy();
  });

  it("each config has required properties", () => {
    for (const config of Object.values(FORMAT_CONFIG)) {
      expect(config.icon).toBeTruthy();
      expect(config.color).toBeTruthy();
      expect(config.bgColor).toBeTruthy();
      expect(config.label).toBeTruthy();
    }
  });
});

describe("EXPORT_STATUS_CONFIG", () => {
  it("has config for all statuses", () => {
    expect(EXPORT_STATUS_CONFIG.completed).toBeTruthy();
    expect(EXPORT_STATUS_CONFIG.processing).toBeTruthy();
    expect(EXPORT_STATUS_CONFIG.pending).toBeTruthy();
    expect(EXPORT_STATUS_CONFIG.failed).toBeTruthy();
  });
});

describe("getFormatConfig", () => {
  it("returns correct config for known formats", () => {
    expect(getFormatConfig("mp4").label).toBe("MP4 Video");
    expect(getFormatConfig("gif").label).toBe("Animated GIF");
    expect(getFormatConfig("json").label).toBe("JSON Package");
    expect(getFormatConfig("html").label).toBe("HTML Bundle");
  });

  it("falls back to json for unknown formats", () => {
    expect(getFormatConfig("unknown").label).toBe("JSON Package");
  });
});

describe("getExportStatusConfig", () => {
  it("returns correct config for known statuses", () => {
    expect(getExportStatusConfig("completed").label).toBe("Ready");
    expect(getExportStatusConfig("processing").label).toBe("Processing");
    expect(getExportStatusConfig("pending").label).toBe("Pending");
    expect(getExportStatusConfig("failed").label).toBe("Failed");
  });

  it("falls back to pending for unknown statuses", () => {
    expect(getExportStatusConfig("unknown").label).toBe("Pending");
  });
});

describe("isBinaryFormat", () => {
  it("returns true for mp4 and gif", () => {
    expect(isBinaryFormat("mp4")).toBe(true);
    expect(isBinaryFormat("gif")).toBe(true);
  });

  it("returns false for json and html", () => {
    expect(isBinaryFormat("json")).toBe(false);
    expect(isBinaryFormat("html")).toBe(false);
  });
});

describe("getFormatExtension", () => {
  it("returns correct extension for each format", () => {
    expect(getFormatExtension("mp4")).toBe("mp4");
    expect(getFormatExtension("gif")).toBe("gif");
    expect(getFormatExtension("json")).toBe("json");
    expect(getFormatExtension("html")).toBe("html");
  });
});

describe("buildExportFileName", () => {
  it("combines stem and format correctly", () => {
    expect(buildExportFileName("my-export", "mp4")).toBe("my-export.mp4");
    expect(buildExportFileName("replay", "json")).toBe("replay.json");
  });
});

describe("sanitizeFileStem", () => {
  it("returns fallback for empty string", () => {
    expect(sanitizeFileStem("", "fallback")).toBe("fallback");
  });

  it("returns fallback for whitespace-only string", () => {
    expect(sanitizeFileStem("   ", "fallback")).toBe("fallback");
  });

  it("preserves alphanumeric characters", () => {
    expect(sanitizeFileStem("abc123", "fallback")).toBe("abc123");
  });

  it("preserves dashes and underscores", () => {
    expect(sanitizeFileStem("my-file_name", "fallback")).toBe("my-file_name");
  });

  it("replaces special characters with dashes", () => {
    expect(sanitizeFileStem("hello world!", "fallback")).toBe("hello-world-");
    expect(sanitizeFileStem("file@name.test", "fallback")).toBe("file-name-test");
  });

  it("trims whitespace before sanitizing", () => {
    expect(sanitizeFileStem("  valid  ", "fallback")).toBe("valid");
  });
});

describe("generateDefaultFileStem", () => {
  it("generates stem with first 8 chars of execution ID", () => {
    expect(generateDefaultFileStem("12345678-abcd-efgh")).toBe(
      "browser-automation-replay-12345678",
    );
  });

  it("handles short execution IDs", () => {
    expect(generateDefaultFileStem("abc")).toBe("browser-automation-replay-abc");
  });
});

describe("buildDimensionPresetOptions", () => {
  const specDimensions = { width: 1440, height: 900 };
  const customDimensions = { width: 800, height: 600 };

  it("includes spec, 1080p, 720p, and custom options", () => {
    const options = buildDimensionPresetOptions(specDimensions, customDimensions);
    const ids = options.map((o) => o.id);
    expect(ids).toEqual(["spec", "1080p", "720p", "custom"]);
  });

  it("uses spec dimensions for spec option", () => {
    const options = buildDimensionPresetOptions(specDimensions, customDimensions);
    const spec = options.find((o) => o.id === "spec");
    expect(spec?.width).toBe(1440);
    expect(spec?.height).toBe(900);
    expect(spec?.label).toContain("1440×900");
  });

  it("uses custom dimensions for custom option", () => {
    const options = buildDimensionPresetOptions(specDimensions, customDimensions);
    const custom = options.find((o) => o.id === "custom");
    expect(custom?.width).toBe(800);
    expect(custom?.height).toBe(600);
  });

  it("uses preset config for 1080p and 720p", () => {
    const options = buildDimensionPresetOptions(specDimensions, customDimensions);
    const hd1080 = options.find((o) => o.id === "1080p");
    const hd720 = options.find((o) => o.id === "720p");
    expect(hd1080?.width).toBe(1920);
    expect(hd1080?.height).toBe(1080);
    expect(hd720?.width).toBe(1280);
    expect(hd720?.height).toBe(720);
  });
});

describe("formatCapturedLabel", () => {
  it("formats singular correctly", () => {
    expect(formatCapturedLabel(1, "frame")).toBe("1 frame");
    expect(formatCapturedLabel(1, "asset")).toBe("1 asset");
  });

  it("formats plural correctly", () => {
    expect(formatCapturedLabel(2, "frame")).toBe("2 frames");
    expect(formatCapturedLabel(10, "asset")).toBe("10 assets");
  });

  it("formats zero as plural", () => {
    expect(formatCapturedLabel(0, "frame")).toBe("0 frames");
  });

  it("rounds decimal values", () => {
    expect(formatCapturedLabel(1.4, "frame")).toBe("1 frame");
    expect(formatCapturedLabel(1.6, "frame")).toBe("2 frames");
  });
});

describe("coerceMetricNumber", () => {
  it("returns number for valid number input", () => {
    expect(coerceMetricNumber(42)).toBe(42);
    expect(coerceMetricNumber(3.14)).toBe(3.14);
  });

  it("returns 0 for NaN", () => {
    expect(coerceMetricNumber(NaN)).toBe(0);
  });

  it("returns 0 for Infinity", () => {
    expect(coerceMetricNumber(Infinity)).toBe(0);
    expect(coerceMetricNumber(-Infinity)).toBe(0);
  });

  it("parses valid string numbers", () => {
    expect(coerceMetricNumber("42")).toBe(42);
    expect(coerceMetricNumber("3.14")).toBe(3.14);
  });

  it("returns 0 for invalid string", () => {
    expect(coerceMetricNumber("not a number")).toBe(0);
    expect(coerceMetricNumber("")).toBe(0);
  });

  it("returns 0 for null and undefined", () => {
    expect(coerceMetricNumber(null)).toBe(0);
    expect(coerceMetricNumber(undefined)).toBe(0);
  });

  it("returns 0 for objects", () => {
    expect(coerceMetricNumber({})).toBe(0);
    expect(coerceMetricNumber([])).toBe(0);
  });
});
