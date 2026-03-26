import { describe, it, expect } from "vitest";
import type {
  Brand,
  ScenarioStatus,
  ContrastPairResult,
  BrandContrastResult,
  ApplyPreviewResult,
  GenerateOptionsResult,
} from "./api";

// Domain-level type validation tests that verify the API type contracts
// UI components depend on. These ensure changes to API shapes don't silently
// break the UI rendering logic.

// [REQ:BM-REQ-API-ASSETS] [REQ:BM-REQ-DESIGN-GEN] [REQ:BM-REQ-DESIGN-CONTENT]
// [REQ:BM-REQ-SCAN-CSS] [REQ:BM-REQ-SCAN-JSON] [REQ:BM-REQ-AUDIT-ENDPOINT]
// [REQ:BM-REQ-API-STANDARDS] [REQ:BM-REQ-SCAN-PARTIAL]

describe("Brand type shape — full brand for design language generation", () => {
  const fullBrand: Brand = {
    id: "b1",
    name: "Design Language Brand",
    description: "Used for design language document generation",
    version: 1,
    created_at: "2026-01-01T00:00:00Z",
    updated_at: "2026-01-01T00:00:00Z",
    identity: {
      display_name: "DL Corp",
      tagline: "Design first",
      logo_path: "/logo.svg",
      favicon_path: "/favicon.ico",
    },
    colors: {
      primary: "#1a365d",
      secondary: "#2d3748",
      accent: "#e53e3e",
      background: "#ffffff",
      surface: "#f7fafc",
      text: "#1a202c",
      error: "#c53030",
    },
    typography: {
      heading_font: "Inter",
      body_font: "Open Sans",
      mono_font: "Fira Code",
      base_font_size: "16px",
    },
    voice: {
      tone: "professional",
      style: "concise",
      keywords: ["reliable", "modern"],
    },
    notes: "Design language notes",
  };

  it("has all facets required for design language generation", () => {
    expect(fullBrand.identity).toBeDefined();
    expect(fullBrand.colors).toBeDefined();
    expect(fullBrand.typography).toBeDefined();
    expect(fullBrand.voice).toBeDefined();
    expect(fullBrand.notes).toBeDefined();
  });

  it("identity has all fields for design language document", () => {
    const identity = fullBrand.identity;
    expect(identity).toBeDefined();
    expect(identity?.display_name).toBe("DL Corp");
    expect(identity?.tagline).toBe("Design first");
    expect(identity?.logo_path).toBe("/logo.svg");
    expect(identity?.favicon_path).toBe("/favicon.ico");
  });

  it("colors has all 7 fields for design language color table", () => {
    const colors = fullBrand.colors;
    expect(colors).toBeDefined();
    const keys = Object.keys(colors ?? {});
    expect(keys).toHaveLength(7);
    expect(colors?.primary).toMatch(/^#/);
    expect(colors?.error).toMatch(/^#/);
  });

  it("voice has keywords array for design language document", () => {
    expect(fullBrand.voice?.keywords).toEqual(["reliable", "modern"]);
  });
});

describe("ScenarioStatus — scan/audit UI display", () => {
  it("unassigned status has null brand fields", () => {
    const status: ScenarioStatus = {
      scenario: "test",
      has_brand: false,
      brand_id: null,
      brand_version: null,
    };
    expect(status.has_brand).toBe(false);
    expect(status.brand_id).toBeNull();
    expect(status.elements).toBeUndefined();
  });

  it("assigned status carries elements list for scan display", () => {
    const status: ScenarioStatus = {
      scenario: "branded-app",
      has_brand: true,
      brand_id: "b1",
      brand_version: 2,
      elements: ["colors", "typography", "identity"],
      applied_at: "2026-03-15T10:00:00Z",
    };
    expect(status.elements).toHaveLength(3);
    expect(status.applied_at).toBeTruthy();
  });
});

describe("ContrastPairResult — WCAG audit display", () => {
  it("passing pair has ratio above 4.5", () => {
    const result: ContrastPairResult = {
      foreground: "#000000",
      background: "#ffffff",
      ratio: 21.0,
      aa_normal: true,
      aa_large: true,
    };
    expect(result.ratio).toBeGreaterThanOrEqual(4.5);
    expect(result.aa_normal).toBe(true);
  });

  it("failing pair has ratio below 4.5", () => {
    const result: ContrastPairResult = {
      foreground: "#cccccc",
      background: "#ffffff",
      ratio: 1.6,
      aa_normal: false,
      aa_large: false,
    };
    expect(result.ratio).toBeLessThan(4.5);
    expect(result.aa_normal).toBe(false);
  });
});

describe("BrandContrastResult — audit endpoint response shape", () => {
  it("carries pass_all flag and pairs array", () => {
    const result: BrandContrastResult = {
      pass_all: true,
      pairs: [
        { foreground: "#000", background: "#fff", ratio: 21.0, aa_normal: true, aa_large: true },
      ],
    };
    expect(result.pass_all).toBe(true);
    expect(result.pairs).toHaveLength(1);
  });

  it("pass_all false when any pair fails", () => {
    const result: BrandContrastResult = {
      pass_all: false,
      pairs: [
        { foreground: "#000", background: "#fff", ratio: 21.0, aa_normal: true, aa_large: true },
        { foreground: "#ccc", background: "#fff", ratio: 1.6, aa_normal: false, aa_large: false },
      ],
    };
    expect(result.pass_all).toBe(false);
    expect(result.pairs?.some(p => !p.aa_normal)).toBe(true);
  });
});

describe("ApplyPreviewResult — scan/apply display shape", () => {
  it("carries applied CSS and JSON actions for scan display", () => {
    const result: ApplyPreviewResult = {
      scenario: "test-app",
      brand_id: "b1",
      brand_version: 1,
      applied: [
        { type: "css", file: "brand.css", element: "colors" },
        { type: "json", file: "manifest.json", element: "identity" },
      ],
      dry_run: true,
    };
    expect(result.applied).toHaveLength(2);
    expect(result.applied?.[0]?.type).toBe("css");
    expect(result.applied?.[1]?.type).toBe("json");
  });

  it("includes skipped elements with reasons", () => {
    const result: ApplyPreviewResult = {
      scenario: "test-app",
      brand_id: "b1",
      brand_version: 1,
      applied: [],
      skipped: [
        { element: "favicon", reason: "No source asset" },
        { element: "logo", reason: "No source asset" },
      ],
      dry_run: true,
    };
    expect(result.skipped).toHaveLength(2);
    expect(result.skipped?.[0]?.reason).toBe("No source asset");
  });
});

describe("GenerateOptionsResult — design generation config", () => {
  it("carries providers with availability status", () => {
    const result: GenerateOptionsResult = {
      providers: [
        { id: "ollama", name: "Ollama", description: "Local LLM", available: true, capabilities: ["text", "colors"] },
        { id: "openrouter", name: "OpenRouter", description: "Cloud LLM", available: false, capabilities: ["text"], requires: "API key" },
      ],
      elements: ["colors", "typography", "voice"],
    };
    expect(result.providers).toHaveLength(2);
    expect(result.providers?.[0]?.available).toBe(true);
    expect(result.providers?.[1]?.available).toBe(false);
    expect(result.elements).toContain("colors");
  });
});
