import { describe, it, expect } from "vitest";
import { ApiRequestError } from "./api";
import type { Brand, BrandColors, BrandTypography, BrandVoice } from "./api";

// Additional type-level and behavior tests for API module.
// [REQ:BM-REQ-API-BRANDS] [REQ:BM-REQ-WCAG-CALC]

describe("ApiRequestError edge cases", () => {
  it("conflict error is not retryable", () => {
    const err = new ApiRequestError(409, { code: "conflict", message: "version mismatch" });
    expect(err.isRetryable).toBe(false);
  });

  it("409 without apiError body is not retryable", () => {
    const err = new ApiRequestError(409);
    expect(err.isRetryable).toBe(false);
  });

  it("502 with dependency code is retryable", () => {
    const err = new ApiRequestError(502, { code: "dependency", message: "upstream" });
    expect(err.isRetryable).toBe(true);
  });

  it("has correct name property", () => {
    const err = new ApiRequestError(400, { code: "validation", message: "bad" });
    expect(err.name).toBe("ApiRequestError");
    expect(err instanceof Error).toBe(true);
  });

  it("recovery undefined for non-error codes", () => {
    const err = new ApiRequestError(500, { code: "internal", message: "crash" });
    expect(err.recovery).toBeUndefined();
  });

  it("recovery present when provided", () => {
    const err = new ApiRequestError(400, {
      code: "validation",
      message: "bad color",
      recovery: "Use #RRGGBB format",
    });
    expect(err.recovery).toBe("Use #RRGGBB format");
  });
});

describe("Brand type shape validation", () => {
  it("brand with all optional fields undefined is valid", () => {
    const brand: Brand = {
      id: "b1",
      name: "Test",
      version: 1,
      created_at: "2026-01-01T00:00:00Z",
      updated_at: "2026-01-01T00:00:00Z",
    };
    expect(brand.identity).toBeUndefined();
    expect(brand.colors).toBeUndefined();
    expect(brand.typography).toBeUndefined();
    expect(brand.voice).toBeUndefined();
    expect(brand.notes).toBeUndefined();
  });

  it("BrandColors supports all color fields", () => {
    const colors: BrandColors = {
      primary: "#ff0000",
      secondary: "#00ff00",
      accent: "#0000ff",
      background: "#ffffff",
      surface: "#f5f5f5",
      text: "#333333",
      error: "#cc0000",
    };
    expect(Object.keys(colors)).toHaveLength(7);
  });

  it("BrandTypography supports all font fields", () => {
    const typo: BrandTypography = {
      heading_font: "Inter",
      body_font: "Open Sans",
      mono_font: "Fira Code",
      base_font_size: "16px",
    };
    expect(Object.keys(typo)).toHaveLength(4);
  });

  it("BrandVoice supports keywords array", () => {
    const voice: BrandVoice = {
      tone: "professional",
      style: "concise",
      keywords: ["modern", "reliable"],
    };
    expect(voice.keywords).toHaveLength(2);
  });
});
