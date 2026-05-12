import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import { Route, Routes } from "react-router-dom";

import { renderWithProviders } from "../test-utils";
import { makeApiError } from "../api/client";

vi.mock("../api/scenarios", () => ({
  fetchScenarios: vi.fn(),
  fetchScenarioDetail: vi.fn(),
}));
vi.mock("../api/inventory", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../api/inventory")>();
  return { ...actual, fetchRuns: vi.fn(), verifyFlow: vi.fn() };
});
vi.mock("../api/artifacts", () => ({
  generateScenarioArtifacts: vi.fn(),
  clearScenarioArtifacts: vi.fn(),
}));

import { ScenarioDetailPage } from "./ScenarioDetailPage";

const flow = {
  flowId: "alpha.flow",
  contractPath: "api/feature/flow/flow.json",
  language: "go",
  schemaVersion: 6,
};

const detail = {
  id: "alpha",
  displayName: "Alpha",
  description: "First scenario",
  path: "/repo/scenarios/alpha",
  flowCount: 1,
  flows: [flow],
};

function renderAt(path: string) {
  return renderWithProviders(
    <Routes>
      <Route path="/scenarios/:scenarioId" element={<ScenarioDetailPage />} />
    </Routes>,
    { routerEntries: [path] },
  );
}

describe("ScenarioDetailPage", () => {
  beforeEach(async () => {
    const { fetchScenarioDetail } = await import("../api/scenarios");
    const { fetchRuns } = await import("../api/inventory");
    vi.mocked(fetchScenarioDetail).mockReset();
    vi.mocked(fetchRuns).mockResolvedValue([]);
  });
  afterEach(() => cleanup());

  it("renders scenario header + flows table", async () => {
    const { fetchScenarioDetail } = await import("../api/scenarios");
    vi.mocked(fetchScenarioDetail).mockResolvedValue(detail);
    renderAt("/scenarios/alpha");
    await waitFor(() =>
      expect(screen.getByTestId("scenario-detail-heading")).toHaveTextContent("Alpha"),
    );
    expect(screen.getByTestId(`scenario-detail-row-${flow.flowId}`)).toBeInTheDocument();
    // Flow rows link to /flows/<id>?scenario=<scenarioId> so the detail page
    // can skip the cross-scenario lookup on the server.
    const link = screen
      .getByTestId(`scenario-detail-row-${flow.flowId}`)
      .querySelector("a");
    expect(link?.getAttribute("href")).toContain("/flows/alpha.flow");
    expect(link?.getAttribute("href")).toContain("scenario=alpha");
  });

  it("renders the empty-flows state when a scenario has no flows yet", async () => {
    const { fetchScenarioDetail } = await import("../api/scenarios");
    vi.mocked(fetchScenarioDetail).mockResolvedValue({ ...detail, flows: [], flowCount: 0 });
    renderAt("/scenarios/alpha");
    await waitFor(() =>
      expect(screen.getByTestId("scenario-detail-empty")).toBeInTheDocument(),
    );
  });

  it("renders the not-found state on a 404", async () => {
    const { fetchScenarioDetail } = await import("../api/scenarios");
    vi.mocked(fetchScenarioDetail).mockRejectedValue(makeApiError("not_found", "missing", 404));
    renderAt("/scenarios/missing");
    await waitFor(() =>
      expect(screen.getByTestId("scenario-detail-error")).toBeInTheDocument(),
    );
  });

  it("renders Generate all and Clear all bulk buttons; Clear gates a confirmation", async () => {
    const { fetchScenarioDetail } = await import("../api/scenarios");
    const { generateScenarioArtifacts, clearScenarioArtifacts } = await import("../api/artifacts");
    vi.mocked(fetchScenarioDetail).mockResolvedValue(detail);
    vi.mocked(generateScenarioArtifacts).mockResolvedValue({ scenarioId: "alpha", flows: [] });
    vi.mocked(clearScenarioArtifacts).mockResolvedValue({ scenarioId: "alpha", flows: [] });
    const user = (await import("@testing-library/user-event")).default.setup();
    renderAt("/scenarios/alpha");
    await waitFor(() =>
      expect(screen.getByTestId("scenario-detail-generate-all")).toBeInTheDocument(),
    );
    await user.click(screen.getByTestId("scenario-detail-generate-all"));
    await waitFor(() => expect(generateScenarioArtifacts).toHaveBeenCalledWith("alpha"));

    await user.click(screen.getByTestId("scenario-detail-clear-all"));
    expect(screen.getByTestId("scenario-detail-clear-confirm")).toBeInTheDocument();
    await user.click(screen.getByTestId("scenario-detail-clear-all-yes"));
    await waitFor(() => expect(clearScenarioArtifacts).toHaveBeenCalledWith("alpha"));
  });

  it("renders a needs_generate pill when the latest run has a missing_artifacts failureReason", async () => {
    const { fetchScenarioDetail } = await import("../api/scenarios");
    const { fetchRuns } = await import("../api/inventory");
    vi.mocked(fetchScenarioDetail).mockResolvedValue(detail);
    vi.mocked(fetchRuns).mockResolvedValue([
      {
        id: "run-x",
        flowId: flow.flowId,
        flowPath: flow.contractPath,
        root: ".",
        mode: "check",
        status: "failed",
        startedAt: "2026-05-12T10:00:00Z",
        finishedAt: "2026-05-12T10:00:01Z",
        durationMs: 1000,
        failureReason: "missing_artifacts",
        missingArtifacts: ["runtime.go"],
      },
    ]);
    renderAt("/scenarios/alpha");
    await waitFor(() =>
      expect(screen.getByTestId(`scenario-detail-pill-${flow.flowId}`)).toBeInTheDocument(),
    );
  });

  it("renders the breadcrumb back to scenarios", async () => {
    const { fetchScenarioDetail } = await import("../api/scenarios");
    vi.mocked(fetchScenarioDetail).mockResolvedValue(detail);
    renderAt("/scenarios/alpha");
    await waitFor(() =>
      expect(screen.getByTestId("scenario-detail-breadcrumb")).toHaveAttribute(
        "href",
        "/scenarios",
      ),
    );
  });
});
