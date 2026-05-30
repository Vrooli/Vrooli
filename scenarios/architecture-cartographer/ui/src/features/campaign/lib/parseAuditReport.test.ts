import { describe, expect, it } from "vitest";

import { FindingSeverity, FindingSource } from "@vrooli/proto-types/architecture/v1/findings_pb";

import { AuditReportParseError, parseAuditReport } from "./parseAuditReport";

describe("parseAuditReport", () => {
  it("flattens findings across every phase", () => {
    const report = JSON.stringify({
      phases: [
        {
          name: "architecture",
          findings: [
            {
              scenario: "swarm-manager",
              source: FindingSource.ARCHITECTURE,
              code: "cycle/cross-domain",
              severity: FindingSeverity.BLOCKER,
              locations: ["api/a", "api/b"],
              domains: ["a", "b"],
              message: "import cycle",
            },
          ],
        },
        {
          name: "structure",
          findings: [
            {
              scenario: "swarm-manager",
              source: FindingSource.STRUCTURE,
              code: "mislocated_file",
              severity: FindingSeverity.WARNING,
              locations: ["api/x.go"],
            },
          ],
        },
      ],
    });

    const findings = parseAuditReport(report);
    expect(findings).toHaveLength(2);
    const [first, second] = findings;
    expect(first!.code).toBe("cycle/cross-domain");
    // Enum ints survive as protobuf-es enum values.
    expect(first!.source).toBe(FindingSource.ARCHITECTURE);
    expect(first!.severity).toBe(FindingSeverity.BLOCKER);
    expect(first!.locations).toEqual(["api/a", "api/b"]);
    expect(second!.code).toBe("mislocated_file");
  });

  it("accepts a bare findings array", () => {
    const findings = parseAuditReport(
      JSON.stringify({ findings: [{ scenario: "demo", code: "x", source: 1, severity: 1 }] }),
    );
    expect(findings).toHaveLength(1);
    expect(findings[0]!.scenario).toBe("demo");
  });

  it("returns [] for a valid report with no findings", () => {
    expect(parseAuditReport(JSON.stringify({ phases: [{ name: "unit" }] }))).toEqual([]);
  });

  it("throws on empty input", () => {
    expect(() => parseAuditReport("   ")).toThrow(AuditReportParseError);
  });

  it("throws on invalid JSON", () => {
    expect(() => parseAuditReport("{not json")).toThrow(AuditReportParseError);
  });

  it("throws when neither phases nor findings is present", () => {
    expect(() => parseAuditReport(JSON.stringify({ summary: "nope" }))).toThrow(AuditReportParseError);
  });
});
