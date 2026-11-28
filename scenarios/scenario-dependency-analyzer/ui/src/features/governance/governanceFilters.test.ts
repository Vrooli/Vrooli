import { create } from "@bufbuild/protobuf";
import { ApprovedDependencyFindingSchema, ApprovedDependencyRecordSchema, DependencyUsageGroupSchema } from "@vrooli/proto-types/scenario-dependency-analyzer/v1/dependency_governance/dependency_governance_pb";
import { describe, expect, it } from "vitest";

import { matchesFinding, matchesRecord, matchesUsageGroup } from "./governanceFilters";
import { defaultGovernanceFilters, type GovernanceFilters } from "./governanceTypes";

const finding = create(ApprovedDependencyFindingSchema, { ecosystem: "npm", packageName: "example-library", severity: "error", scenario: "alpha", findingClass: "out_of_range", expected: "approved", title: "Version policy", description: "Unsafe release", remediation: "Upgrade dependency", filePath: "ui/package.json" });
const group = create(DependencyUsageGroupSchema, { ecosystem: "npm", packageName: "example-library", highestSeverity: "warning", scenarios: ["alpha", "beta"], state: "needs_review" });
const record = create(ApprovedDependencyRecordSchema, { ecosystem: "npm", packageName: "example-library", state: "approved", versionRange: "^2.0.0", rationale: "Accessible controls", replacement: "native-platform", keywords: ["keyboard"], useCases: ["menus"], exampleScenarios: ["gamma"] });
const filters = (partial: Partial<GovernanceFilters> = {}) => ({ ...defaultGovernanceFilters, ...partial });

describe("governance search and filter semantics", () => {
  it("shows all three record types without restrictions, including whitespace-only searches", () => {
    for (const query of ["", "   "]) {
      expect(matchesFinding(finding, filters({ query }))).toBe(true);
      expect(matchesUsageGroup(group, filters({ query }))).toBe(true);
      expect(matchesRecord(record, filters({ query }))).toBe(true);
    }
  });

  it.each(["EXAMPLE", "NPM", "version", "unsafe", "upgrade", "ALPHA", "package.json", "out_of_range"])("finds findings by relevant text: %s", (query) => {
    expect(matchesFinding(finding, filters({ query: `  ${query}  ` }))).toBe(true);
  });
  it.each(["example", "npm", "needs_review", "warning", "BETA"])("finds usage groups by relevant text: %s", (query) => {
    expect(matchesUsageGroup(group, filters({ query }))).toBe(true);
  });
  it.each(["example", "npm", "APPROVED", "2.0", "accessible", "native-platform", "keyboard", "menus", "gamma"])("finds registry records by relevant text: %s", (query) => {
    expect(matchesRecord(record, filters({ query }))).toBe(true);
  });
  it("combines finding ecosystem, severity, scenario, state and query restrictions", () => {
    const matching = filters({ ecosystem: "npm", severity: "error", scenario: "alpha", state: "approved", query: "version" });
    expect(matchesFinding(finding, matching)).toBe(true);
    for (const mismatch of [{ ecosystem: "go" }, { severity: "info" as const }, { scenario: "beta" }, { state: "denied" as const }, { query: "unrelated" }]) expect(matchesFinding(finding, { ...matching, ...mismatch })).toBe(false);
  });
  it("can filter findings by their class independently from the expected registry state", () => {
    expect(matchesFinding(create(ApprovedDependencyFindingSchema, { ...finding, findingClass: "unrecorded", expected: "" }), filters({ state: "unrecorded" }))).toBe(true);
  });
  it("combines usage filters and accepts any scenario in a multi-scenario group", () => {
    const matching = filters({ ecosystem: "npm", severity: "warning", scenario: "beta", state: "needs_review", query: "example" });
    expect(matchesUsageGroup(group, matching)).toBe(true);
    for (const mismatch of [{ ecosystem: "go" }, { severity: "error" as const }, { scenario: "gamma" }, { state: "denied" as const }, { query: "unrelated" }]) expect(matchesUsageGroup(group, { ...matching, ...mismatch })).toBe(false);
  });
  it("retains globally applicable records and filters explicit scenario grants", () => {
    expect(matchesRecord(record, filters({ scenario: "alpha" }))).toBe(true);
    const scoped = create(ApprovedDependencyRecordSchema, { ...record, allowedScenarios: ["alpha"] });
    expect(matchesRecord(scoped, filters({ scenario: "alpha" }))).toBe(true);
    expect(matchesRecord(scoped, filters({ scenario: "beta" }))).toBe(false);
    expect(matchesRecord(scoped, filters())).toBe(true);
  });
  it("combines record ecosystem, state and text restrictions", () => {
    const matching = filters({ ecosystem: "npm", state: "approved", query: "menus" });
    expect(matchesRecord(record, matching)).toBe(true);
    for (const mismatch of [{ ecosystem: "go" }, { state: "denied" as const }, { query: "unrelated" }]) expect(matchesRecord(record, { ...matching, ...mismatch })).toBe(false);
  });
});
