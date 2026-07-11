import { fireEvent, render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { RemediationPanel } from "./RemediationPanel";
import { useRemediation } from "../../hooks/useRemediation";

vi.mock("../../hooks/useRemediation", () => ({ useRemediation: vi.fn() }));

const mockedUseRemediation = vi.mocked(useRemediation);
const create = { mutate: vi.fn(), isPending: false };
const cancel = { mutate: vi.fn(), isPending: false };
const refresh = { mutate: vi.fn(), isPending: false };
const verify = { mutate: vi.fn(), isPending: false };

const plan = {
  sourceExecutionId: "execution-1",
  sourceRunId: "run-1",
  scenario: "demo",
  createdAt: "2026-07-11T00:00:00Z",
  phases: [],
  findings: [{ stableId: "afid:blocker", code: "missing-evidence", severity: "error", class: "deterministic", phase: "unit", gating: true, locations: ["api/service.go"], message: "Missing evidence" }],
  bundles: [{ id: "bundle:1", reason: "shared location", findingIds: ["afid:blocker"], phaseNames: ["unit"], rank: 1, gating: true }],
  requirements: [{ id: "REQ-1", title: "Evidence requirement", liveStatus: "failed", validations: ["test:unit:failed"] }],
  degraded: false
};

function renderPanel(overrides: Record<string, unknown> = {}) {
  mockedUseRemediation.mockReturnValue({
    plan: { data: plan, isLoading: false, isError: false },
    jobs: { data: [] },
    roles: { data: [{ id: "code.default", label: "Default coding" }] },
    create,
    cancel,
    refresh,
    verify,
    activeJob: undefined,
    ...overrides
  } as unknown as ReturnType<typeof useRemediation>);
  return render(<RemediationPanel scenarioName="demo" executionId="execution-1" />);
}

describe("RemediationPanel", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("launches the selected immutable evidence bundle with a portable role", () => {
    renderPanel();
    expect(screen.getByText("Missing evidence")).toBeInTheDocument();
    expect(screen.getByText("REQ-1 · Evidence requirement")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Launch remediation" }));

    expect(create.mutate).toHaveBeenCalledWith({
      findingIds: ["afid:blocker"],
      requirementIds: [],
      roleRef: "code.default",
      additionalContext: ""
    });
  });

  it("keeps an agent result provisional and exposes server-owned verification", () => {
    renderPanel({
      activeJob: { id: "job-1", status: "agent_completed", selectedFindingIds: ["afid:blocker"], selectedRequirementIds: [] }
    });

    expect(screen.getByText(/provisional until Test Genie verifies a rerun/i)).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Verify with rerun" }));
    expect(verify.mutate).toHaveBeenCalledWith("job-1");
  });

  it("states explicitly when immutable evidence is degraded", () => {
    renderPanel({ plan: { data: { ...plan, degraded: true, degradedReasons: ["descriptor snapshot unavailable"] }, isLoading: false, isError: false } });
    expect(screen.getByText("Evidence needs attention")).toBeInTheDocument();
    expect(screen.getByText("descriptor snapshot unavailable")).toBeInTheDocument();
  });
});
