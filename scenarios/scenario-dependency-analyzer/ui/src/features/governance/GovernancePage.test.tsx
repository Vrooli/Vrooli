import { fireEvent, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { renderWithProviders } from "../../test-utils/renderWithProviders";
import { GovernancePage } from "./GovernancePage";

const mocks = vi.hoisted(() => ({
  validateFleetApprovedDependencies: vi.fn(),
  listApprovedDependencies: vi.fn(),
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

describe("GovernancePage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.validateFleetApprovedDependencies.mockResolvedValue(fleetResponse);
    mocks.listApprovedDependencies.mockResolvedValue({ records: [], guidance: "" });
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

  it("renders fleet posture, filters findings, and opens dependency details", async () => {
    renderWithProviders(<GovernancePage />);

    await waitFor(() => expect(screen.getByText("Dependency Governance")).toBeInTheDocument());
    await waitFor(() => expect(screen.getAllByText("Dependency needs governance review").length).toBeGreaterThan(0));

    fireEvent.click(firstElement(screen.getAllByRole("button", { name: "npm/mermaid" })));

    expect(screen.getByText("Scenarios using this dependency")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Approve" })).toBeInTheDocument();
  });

  it("previews governance decisions through the typed mutation seam", async () => {
    renderWithProviders(<GovernancePage />);

    await waitFor(() => expect(screen.getAllByText("Dependency needs governance review").length).toBeGreaterThan(0));
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
      rationale: "reviewed for graph rendering"
    });
    expect(firstCall[1]).toBe(true);
  });
});

function firstElement<T>(elements: T[]): T {
  const first = elements[0];
  if (!first) {
    throw new Error("Expected at least one matching element");
  }
  return first;
}
