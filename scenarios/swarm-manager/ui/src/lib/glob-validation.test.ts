import { describe, it, expect } from "vitest";
import {
  validateGlobLine,
  validateGlobLines,
  parseGlobTextarea,
} from "./glob-validation";

// ---------------------------------------------------------------------------
// validateGlobLine
// ---------------------------------------------------------------------------

describe("validateGlobLine", () => {
  it("returns error for empty string", () => {
    expect(validateGlobLine("")).toBeDefined();
    expect(validateGlobLine("   ")).toBeDefined();
  });

  it("returns error for absolute path", () => {
    expect(validateGlobLine("/etc/passwd")).toContain("absolute");
  });

  it("returns error for unclosed bracket", () => {
    expect(validateGlobLine("[invalid")).toContain("[");
  });

  it("returns error for unclosed brace", () => {
    expect(validateGlobLine("src/{a,b")).toContain("{");
  });

  it("returns error for unbalanced closing brace", () => {
    expect(validateGlobLine("src/a}")).toContain("}");
  });

  it("returns undefined for valid glob patterns", () => {
    expect(validateGlobLine("**/*.ts")).toBeUndefined();
    expect(validateGlobLine("src/components/**")).toBeUndefined();
    expect(validateGlobLine("src/{a,b}/**")).toBeUndefined();
    expect(validateGlobLine("*.go")).toBeUndefined();
    expect(validateGlobLine("docs/README.md")).toBeUndefined();
  });

  it("accepts escaped characters", () => {
    expect(validateGlobLine("src/\\[special\\]")).toBeUndefined();
  });
});

// ---------------------------------------------------------------------------
// validateGlobLines
// ---------------------------------------------------------------------------

describe("validateGlobLines", () => {
  it("returns valid for empty text", () => {
    const result = validateGlobLines("");
    expect(result.valid).toBe(true);
    expect(result.errors).toHaveLength(0);
  });

  it("skips blank lines (not treated as errors)", () => {
    const result = validateGlobLines("src/**\n\n\ndocs/**");
    expect(result.valid).toBe(true);
    expect(result.errors).toHaveLength(0);
  });

  it("reports errors with correct 1-based line numbers", () => {
    const result = validateGlobLines("src/**\n/absolute\n[bad");
    expect(result.valid).toBe(false);
    expect(result.errors).toHaveLength(2);
    expect(result.errors[0]?.line).toBe(2);
    expect(result.errors[0]?.error).toContain("absolute");
    expect(result.errors[1]?.line).toBe(3);
    expect(result.errors[1]?.error).toContain("[");
  });

  it("handles trailing newlines gracefully", () => {
    const result = validateGlobLines("src/**\n");
    expect(result.valid).toBe(true);
  });

  it("handles whitespace-only lines by skipping them", () => {
    const result = validateGlobLines("src/**\n   \ndocs/**");
    expect(result.valid).toBe(true);
  });
});

// ---------------------------------------------------------------------------
// parseGlobTextarea
// ---------------------------------------------------------------------------

describe("parseGlobTextarea", () => {
  it("splits by newline and trims", () => {
    expect(parseGlobTextarea("  src/**  \n  docs/**  ")).toEqual([
      "src/**",
      "docs/**",
    ]);
  });

  it("filters out empty lines", () => {
    expect(parseGlobTextarea("src/**\n\n\ndocs/**\n")).toEqual([
      "src/**",
      "docs/**",
    ]);
  });

  it("returns empty array for empty input", () => {
    expect(parseGlobTextarea("")).toEqual([]);
    expect(parseGlobTextarea("  \n  \n  ")).toEqual([]);
  });
});
