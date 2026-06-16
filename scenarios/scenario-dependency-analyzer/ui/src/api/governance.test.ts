import { afterEach, describe, expect, it, vi } from "vitest";

import type { ApprovedDependencyRecord } from "./governance";

const mocks = vi.hoisted(() => ({
  client: {
    validateFleetApprovedDependencies: vi.fn(),
    listApprovedDependencies: vi.fn(),
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
    const { validateFleetApprovedDependencies } = await import("./governance");
    mocks.client.validateFleetApprovedDependencies.mockResolvedValueOnce({ passed: true });

    await validateFleetApprovedDependencies("strict");

    expect(mocks.client.validateFleetApprovedDependencies).toHaveBeenCalledWith({
      policyMode: "strict"
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
      allowedDependencyGroups: []
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
