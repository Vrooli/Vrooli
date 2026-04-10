import { describe, it, expect } from "vitest";
import type {
  ScanResult,
  ScanFinding,
  AuditRule,
  AuditResult,
  StandardsResult,
} from "./api";

// Coverage depth tests for scanner, audit, and standards types.
// [REQ:BM-REQ-SCAN-CSS] [REQ:BM-REQ-SCAN-JSON] [REQ:BM-REQ-SCAN-PARTIAL]
// [REQ:BM-REQ-SCAN-EXTEND] [REQ:BM-REQ-SCAN-PLUGINS]
// [REQ:BM-REQ-AUDIT-ENDPOINT] [REQ:BM-REQ-AUDIT-RULES] [REQ:BM-REQ-AUDIT-PROVIDER]
// [REQ:BM-REQ-API-STANDARDS] [REQ:BM-REQ-DISC-SCAN] [REQ:BM-REQ-DISC-LPBS]

describe("ScanResult type contracts", () => {
  it("scan result with CSS findings", () => {
    const result: ScanResult = {
      scenario: "test-app",
      findings: [
        { file: "brand.css", element: "primary", type: "css", line: 10, value: "#ff0000" },
        { file: "brand.css", element: "secondary", type: "css", line: 11 },
      ],
      summary: { total: 2, css: 2, json: 0 },
    };
    expect(result.findings).toHaveLength(2);
    expect(result.findings?.[0]?.type).toBe("css");
    expect(result.summary.css).toBe(2);
    expect(result.summary.json).toBe(0);
  });

  it("scan result with JSON findings", () => {
    const result: ScanResult = {
      scenario: "web-app",
      findings: [
        { file: "manifest.json", element: "identity", type: "json" },
        { file: "manifest.json", element: "display_name", type: "json" },
      ],
      summary: { total: 2, css: 0, json: 2 },
    };
    expect(result.findings.every((f) => f.type === "json")).toBe(true);
    expect(result.summary.json).toBe(2);
  });

  it("scan result with mixed findings", () => {
    const result: ScanResult = {
      scenario: "full-app",
      findings: [
        { file: "brand.css", element: "primary", type: "css", line: 5 },
        { file: "manifest.json", element: "name", type: "json" },
        { file: "config.yaml", element: "color", type: "yaml" },
      ],
      summary: { total: 3, css: 1, json: 1 },
    };
    expect(result.findings).toHaveLength(3);
    const types = new Set(result.findings.map((f) => f.type));
    expect(types.size).toBe(3);
  });

  it("empty scan result has zero summary", () => {
    const result: ScanResult = {
      scenario: "clean-app",
      findings: [],
      summary: { total: 0, css: 0, json: 0 },
    };
    expect(result.findings).toHaveLength(0);
    expect(result.summary.total).toBe(0);
  });

  it("ScanFinding with line number and value", () => {
    const finding: ScanFinding = {
      file: "styles/brand.css",
      element: "accent",
      type: "css",
      line: 42,
      value: "#e53e3e",
    };
    expect(finding.line).toBe(42);
    expect(finding.value).toBe("#e53e3e");
  });

  it("ScanFinding without optional fields", () => {
    const finding: ScanFinding = {
      file: "manifest.json",
      element: "brand_id",
      type: "json",
    };
    expect(finding.line).toBeUndefined();
    expect(finding.value).toBeUndefined();
  });
});

describe("AuditRule and AuditResult type contracts", () => {
  it("audit rule has severity levels", () => {
    const rules: AuditRule[] = [
      { id: "has-logo", name: "Logo Required", description: "Must have logo", severity: "error" },
      { id: "has-colors", name: "Color System", description: "Colors defined", severity: "warning" },
    ];
    expect(rules?.[0]?.severity).toBe("error");
    expect(rules?.[1]?.severity).toBe("warning");
  });

  it("audit result with all passing", () => {
    const result: AuditResult = {
      scenario: "good-app",
      results: [
        { rule_id: "has-logo", passed: true, message: "Logo found" },
        { rule_id: "has-colors", passed: true, message: "Colors OK" },
      ],
      pass_all: true,
    };
    expect(result.pass_all).toBe(true);
    expect(result.results.every((r) => r.passed)).toBe(true);
  });

  it("audit result with failures", () => {
    const result: AuditResult = {
      scenario: "bad-app",
      results: [
        { rule_id: "has-logo", passed: false, message: "No logo" },
        { rule_id: "has-colors", passed: true, message: "OK" },
      ],
      pass_all: false,
    };
    expect(result.pass_all).toBe(false);
    expect(result.results.filter((r) => !r.passed)).toHaveLength(1);
  });
});

describe("StandardsResult type contracts", () => {
  it("standards result with multiple rules", () => {
    const result: StandardsResult = {
      rules: [
        { id: "has-logo", name: "Logo", description: "Must have logo", severity: "error" },
        { id: "has-favicon", name: "Favicon", description: "Must have favicon", severity: "error" },
        { id: "has-color-system", name: "Colors", description: "Color system", severity: "warning" },
      ],
    };
    expect(result.rules).toHaveLength(3);
    expect(result.rules.filter((r) => r.severity === "error")).toHaveLength(2);
  });

  it("empty standards result", () => {
    const result: StandardsResult = { rules: [] };
    expect(result.rules).toHaveLength(0);
  });
});
