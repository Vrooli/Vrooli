import { afterEach, describe, expect, it, vi } from "vitest";

import type { ApprovedDependencyRecord } from "./governance";

const mocks = vi.hoisted(() => ({
  client: {
    validateFleetApprovedDependencies: vi.fn(),
    listApprovedDependencies: vi.fn(),
    getApprovedDependencyTriage: vi.fn(),
    listSecurityGovernanceGaps: vi.fn(),
    upsertApprovedDependency: vi.fn(),
    previewVulnerabilityRemediation: vi.fn(),
    denyVulnerableDependency: vi.fn()
  }
}));

vi.mock("@connectrpc/connect", () => ({
  createClient: vi.fn(() => mocks.client)
}));

describe("api/governance", () => {
  afterEach(() => {
    vi.clearAllMocks();
  });

  it("exports a generated Connect governance client", async () => {
    const { governanceClient } = await import("./governance");

    await governanceClient.validateFleetApprovedDependencies({ policyMode: "advisory" });

    expect(mocks.client.validateFleetApprovedDependencies).toHaveBeenCalledWith({
      policyMode: "advisory"
    });
  });

  it("wraps fleet validation with the selected policy mode", async () => {
    const { getApprovedDependencyTriage, listSecurityGovernanceGaps, validateFleetApprovedDependencies } = await import("./governance");
    mocks.client.validateFleetApprovedDependencies.mockResolvedValueOnce({ passed: true });
    mocks.client.getApprovedDependencyTriage.mockResolvedValueOnce({ guidance: "" });
    mocks.client.listSecurityGovernanceGaps.mockResolvedValueOnce({ gaps: [] });

    await validateFleetApprovedDependencies("strict");
    await getApprovedDependencyTriage({ policyMode: "review_gate", limit: 5, ecosystem: "npm" });
    await listSecurityGovernanceGaps({ ecosystem: "npm", packageName: "vite", minimumSeverity: "high", limit: 3 });

    expect(mocks.client.validateFleetApprovedDependencies).toHaveBeenCalledWith({
      policyMode: "strict"
    });
    expect(mocks.client.getApprovedDependencyTriage).toHaveBeenCalledWith({
      policyMode: "review_gate",
      section: "",
      ecosystem: "npm",
      packageName: "",
      limit: 5
    });
    expect(mocks.client.listSecurityGovernanceGaps).toHaveBeenCalledWith({
      ecosystem: "npm",
      packageName: "vite",
      scenario: "",
      vulnerabilityId: "",
      minimumSeverity: "high",
      limit: 3
    });
  });

  it("keeps governance mutations on the typed Connect service", async () => {
    const { upsertApprovedDependency, denyVulnerableDependency } = await import("./governance");
    const record: ApprovedDependencyRecord = {
      $typeName: "vrooli.scenario_dependency_analyzer.v1.dependency_governance.ApprovedDependencyRecord",
      ecosystem: "npm",
      packageName: "react",
      versionRange: "^18.0.0",
      state: "approved",
      allowedSurfaces: [],
      useCases: [],
      rationale: "reviewed",
      approvedBy: "tester",
      approvedDate: "2026-06-16",
      lastReviewed: "2026-06-16",
      reviewExpires: "",
      licenseNotes: "",
      securityNotes: "",
      exampleScenarios: [],
      replacement: "",
      keywords: [],
      allowedScenarios: [],
      deniedScenarios: [],
      allowedDependencyGroups: [],
      rangePolicy: ""
    };

    await upsertApprovedDependency(record, true);
    await denyVulnerableDependency({
      ecosystem: "npm",
      packageName: "react",
      vulnerabilityId: "GHSA-test",
      affectedRange: "<18.3.1",
      fixedRange: ">=18.3.1",
      rationale: "gating vulnerability",
      approvedBy: "tester",
      dryRun: false
    });

    expect(mocks.client.upsertApprovedDependency).toHaveBeenCalledWith({ record, dryRun: true });
    expect(mocks.client.denyVulnerableDependency).toHaveBeenCalledWith({
      ecosystem: "npm",
      packageName: "react",
      vulnerabilityId: "GHSA-test",
      affectedRange: "<18.3.1",
      fixedRange: ">=18.3.1",
      rationale: "gating vulnerability",
      approvedBy: "tester",
      dryRun: false
    });
  });
});
