import { fireEvent, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { renderWithProviders } from "@vrooli/api-base/testing";
import { selectors } from "../../consts/selectors";
import { GovernancePage } from "./GovernancePage";

const mocks = vi.hoisted(() => ({
  validateFleetApprovedDependencies: vi.fn(),
  listApprovedDependencies: vi.fn(),
  getApprovedDependencyTriage: vi.fn(),
  listSecurityGovernanceGaps: vi.fn(),
  upsertApprovedDependency: vi.fn(),
  previewVulnerabilityRemediation: vi.fn(),
  denyVulnerableDependency: vi.fn()
}));

vi.mock("../../api/governance", async () => {
  const actual = await vi.importActual<typeof import("../../api/governance")>("../../api/governance");
  return {
    ...actual,
    validateFleetApprovedDependencies: mocks.validateFleetApprovedDependencies,
    listApprovedDependencies: mocks.listApprovedDependencies,
    getApprovedDependencyTriage: mocks.getApprovedDependencyTriage,
    listSecurityGovernanceGaps: mocks.listSecurityGovernanceGaps,
    upsertApprovedDependency: mocks.upsertApprovedDependency,
    previewVulnerabilityRemediation: mocks.previewVulnerabilityRemediation,
    denyVulnerableDependency: mocks.denyVulnerableDependency
  };
});

const fleetResponse = {
  passed: true,
  summary: {
    status: "warn",
    approved: 1,
    approvedWithConstraints: 0,
    needsReview: 1,
    blocked: 0,
    deprecated: 0,
    unrecorded: 1,
    observed: 2,
    policyMode: "advisory",
    denied: 0,
    outOfRange: 0,
    outOfScope: 0,
    expired: 0,
    scenarioCount: 1,
    dependencyCount: 2,
    findingCount: 1,
    errorCount: 0,
    warningCount: 1,
    infoCount: 0
  },
  scenarios: [],
  usageGroups: [
    {
      ecosystem: "npm",
      packageName: "mermaid",
      scenarioCount: 1,
      usageCount: 1,
      scenarios: ["scenario-dependency-analyzer"],
      observedDependencies: [],
      findingCount: 1,
      highestSeverity: "warning",
      state: "needs_review"
    },
    {
      ecosystem: "npm",
      packageName: "vite",
      scenarioCount: 1,
      usageCount: 1,
      scenarios: ["scenario-dependency-analyzer"],
      observedDependencies: [],
      findingCount: 1,
      highestSeverity: "error",
      state: "approved"
    }
  ],
  findings: [
    {
      id: "finding-1",
      severity: "warning",
      title: "Dependency needs governance review",
      description: "mermaid is not recorded yet.",
      remediation: "Review and approve or deny this dependency.",
      filePath: "ui/package.json",
      ecosystem: "npm",
      packageName: "mermaid",
      observed: "^11.0.0",
      expected: "needs_review",
      scenario: "scenario-dependency-analyzer",
      findingClass: "unrecorded_direct",
      policyMode: "advisory"
    }
  ],
  guidance: "Review dependency decisions before strict mode."
};

const triageResponse = {
  summary: fleetResponse.summary,
  securityActions: [],
  registrySeeding: [
    {
      groupId: "seeding/npm/mermaid",
      title: "Approve observed direct dependency",
      actionType: "approve_observed",
      section: "seeding",
      ecosystem: "npm",
      packageName: "mermaid",
      findingCount: 1,
      scenarioCount: 1,
      usageCount: 1,
      highestSeverity: "warning",
      findingClasses: ["unrecorded_direct"],
      scenarios: ["scenario-dependency-analyzer"],
      observedVersions: ["^11.0.0"],
      vulnerabilityIds: [],
      recommendedCommand: "scenario-dependency-analyzer deps approved approve-observed npm/mermaid --from-findings",
      rationale: "Direct dependency is used by the SDA UI."
    }
  ],
  rangePolicy: [],
  scenarioHotspots: [],
  staleOrExpiredReviews: [],
  guidance: "Review grouped governance decisions first."
};

const securityGapsResponse = {
  gaps: [
    {
      gapId: "npm/vite/GHSA-4w7w-66w2-5vf9",
      ecosystem: "npm",
      packageName: "vite",
      observedVersion: "4.5.14",
      vulnerabilityIds: ["GHSA-4w7w-66w2-5vf9"],
      severity: "high",
      normalizedSeverity: "error",
      affectedRanges: [">=0 <6.4.2"],
      fixedRanges: [">=6.4.2"],
      scenarios: ["scenario-dependency-analyzer"],
      sourceFiles: ["ui/pnpm-lock.yaml"],
      deniedRecordCovers: false,
      approvedRecordOverlaps: true,
      signalCategory: "direct_dev",
      suggestedCommand: "scenario-dependency-analyzer deps approved deny-vulnerable npm/vite --vulnerability GHSA-4w7w-66w2-5vf9",
      remediation: "Create a denied vulnerable range for affected Vite versions."
    }
  ],
  total: 1,
  uncoveredCount: 1,
  deniedCoveredCount: 0,
  approvedOverlapCount: 1,
  warningCount: 0,
  warnings: [],
  guidance: "Review vulnerable dependencies before strict mode."
};

describe("GovernancePage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.validateFleetApprovedDependencies.mockResolvedValue(fleetResponse);
    mocks.listApprovedDependencies.mockResolvedValue({ records: [], guidance: "" });
    mocks.getApprovedDependencyTriage.mockResolvedValue(triageResponse);
    mocks.listSecurityGovernanceGaps.mockResolvedValue(securityGapsResponse);
    mocks.upsertApprovedDependency.mockResolvedValue({
      dryRun: true,
      changed: true,
      message: "Dry-run decision is valid.",
      record: undefined,
      previousRecord: undefined,
      summary: fleetResponse.summary,
      guidance: ""
    });
  });

  it("renders triage-first governance and opens dependency details", async () => {
    renderWithProviders(<GovernancePage />);

    await waitFor(() => expect(screen.getByText("Dependency Governance")).toBeInTheDocument());
    expect(screen.getByText(/not an exhaustive allowlist/i)).toBeInTheDocument();
    await waitFor(() => expect(screen.getByText("Registry seeding")).toBeInTheDocument());
    expect(screen.getByText("Security Health evidence gaps")).toBeInTheDocument();
    expect(screen.getByTestId(selectors.governance.triagePanel)).toBeInTheDocument();

    fireEvent.click(firstElement(screen.getAllByRole("button", { name: "npm/mermaid" })));

    expect(screen.getByText("Scenarios using this dependency")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Approve" })).toBeInTheDocument();
  });

  it("previews governance decisions through the typed mutation seam", async () => {
    renderWithProviders(<GovernancePage />);

    await waitFor(() => expect(screen.getByText("Registry seeding")).toBeInTheDocument());
    fireEvent.click(firstElement(screen.getAllByRole("button", { name: "npm/mermaid" })));
    fireEvent.click(screen.getByRole("button", { name: "Approve" }));
    fireEvent.change(screen.getByLabelText("Rationale"), { target: { value: "reviewed for graph rendering" } });
    fireEvent.click(screen.getByRole("button", { name: "Dry-run preview" }));

    await waitFor(() => expect(mocks.upsertApprovedDependency).toHaveBeenCalled());
    const firstCall = mocks.upsertApprovedDependency.mock.calls[0];
    if (!firstCall) {
      throw new Error("Expected upsertApprovedDependency to be called");
    }
    expect(firstCall[0]).toMatchObject({
      ecosystem: "npm",
      packageName: "mermaid",
      state: "approved",
      rationale: "reviewed for graph rendering",
      allowedScenarios: []
    });
    expect(firstCall[1]).toBe(true);
  });

  it("opens security gap remediation with the vulnerability id prefilled", async () => {
    renderWithProviders(<GovernancePage />);

    await waitFor(() => expect(screen.getByTestId(selectors.governance.triagePanel)).toBeInTheDocument());
    fireEvent.click(screen.getByRole("button", { name: "Deny range" }));

    expect(screen.getByText("Security Health evidence decision")).toBeInTheDocument();
    expect(screen.getByLabelText("Vulnerability ID")).toHaveValue("GHSA-4w7w-66w2-5vf9");
  });
});

function firstElement<T>(elements: T[]): T {
  const first = elements[0];
  if (!first) {
    throw new Error("Expected at least one matching element");
  }
  return first;
}
