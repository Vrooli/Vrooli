import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../test-utils";

vi.mock("../api/scenarios", () => ({
  fetchScenarios: vi.fn(),
  fetchScenarioDetail: vi.fn(),
}));

vi.mock("../api/artifacts", () => ({
  generateScenarioArtifacts: vi.fn(),
  clearScenarioArtifacts: vi.fn(),
}));

import { ScenariosPage } from "./ScenariosPage";

const alpha = {
  id: "alpha",
  displayName: "Alpha",
  description: "First scenario",
  path: "/repo/scenarios/alpha",
  flowCount: 2,
};
const beta = {
  id: "beta",
  displayName: "Beta",
  description: "Second scenario",
  path: "/repo/scenarios/beta",
  flowCount: 0,
};
const gamma = {
  id: "gamma",
  displayName: "Gamma",
  description: "Third (broken)",
  path: "/repo/scenarios/gamma",
  flowCount: 0,
  discoveryError: "permission denied",
};

describe("ScenariosPage", () => {
  beforeEach(async () => {
    const { fetchScenarios } = await import("../api/scenarios");
    vi.mocked(fetchScenarios).mockReset();
    const { generateScenarioArtifacts, clearScenarioArtifacts } = await import("../api/artifacts");
    vi.mocked(generateScenarioArtifacts).mockReset();
    vi.mocked(clearScenarioArtifacts).mockReset();
  });
  afterEach(() => cleanup());

  it("renders one row per scenario with a drill-in link", async () => {
    const { fetchScenarios } = await import("../api/scenarios");
    vi.mocked(fetchScenarios).mockResolvedValue({
      vrooliRoot: "/repo",
      scenarios: [alpha, beta],
    });
    renderWithProviders(<ScenariosPage />);
    await waitFor(() =>
      expect(screen.getByTestId("scenario-table")).toBeInTheDocument(),
    );
    expect(screen.getByTestId("scenario-link-alpha")).toHaveAttribute(
      "href",
      "/scenarios/alpha",
    );
    expect(screen.getByTestId("scenario-link-beta")).toHaveAttribute(
      "href",
      "/scenarios/beta",
    );
  });

  it("filters by search text against id, displayName, and description", async () => {
    const { fetchScenarios } = await import("../api/scenarios");
    vi.mocked(fetchScenarios).mockResolvedValue({
      vrooliRoot: "/repo",
      scenarios: [alpha, beta],
    });
    const user = userEvent.setup();
    renderWithProviders(<ScenariosPage />);
    await waitFor(() => expect(screen.getByTestId("scenario-table")).toBeInTheDocument());
    await user.type(screen.getByTestId("scenario-search"), "Beta");
    await waitFor(() =>
      expect(screen.queryByTestId("scenario-row-alpha")).not.toBeInTheDocument(),
    );
    expect(screen.getByTestId("scenario-row-beta")).toBeInTheDocument();
  });

  it("filters by flows=has and by flows=empty", async () => {
    const { fetchScenarios } = await import("../api/scenarios");
    vi.mocked(fetchScenarios).mockResolvedValue({
      vrooliRoot: "/repo",
      scenarios: [alpha, beta],
    });
    const user = userEvent.setup();
    renderWithProviders(<ScenariosPage />);
    await waitFor(() => expect(screen.getByTestId("scenario-table")).toBeInTheDocument());

    await user.selectOptions(screen.getByTestId("scenario-flows"), "has");
    await waitFor(() =>
      expect(screen.queryByTestId("scenario-row-beta")).not.toBeInTheDocument(),
    );
    expect(screen.getByTestId("scenario-row-alpha")).toBeInTheDocument();

    await user.selectOptions(screen.getByTestId("scenario-flows"), "empty");
    await waitFor(() =>
      expect(screen.queryByTestId("scenario-row-alpha")).not.toBeInTheDocument(),
    );
    expect(screen.getByTestId("scenario-row-beta")).toBeInTheDocument();
  });

  it("filters by errors=with to surface only scenarios with a discovery error", async () => {
    const { fetchScenarios } = await import("../api/scenarios");
    vi.mocked(fetchScenarios).mockResolvedValue({
      vrooliRoot: "/repo",
      scenarios: [alpha, gamma],
    });
    const user = userEvent.setup();
    renderWithProviders(<ScenariosPage />);
    await waitFor(() => expect(screen.getByTestId("scenario-table")).toBeInTheDocument());
    await user.selectOptions(screen.getByTestId("scenario-errors"), "with");
    await waitFor(() =>
      expect(screen.queryByTestId("scenario-row-alpha")).not.toBeInTheDocument(),
    );
    expect(screen.getByTestId("scenario-row-gamma")).toBeInTheDocument();
  });

  it("sorts by flowCount descending when key=flowCount + dir flipped", async () => {
    const { fetchScenarios } = await import("../api/scenarios");
    vi.mocked(fetchScenarios).mockResolvedValue({
      vrooliRoot: "/repo",
      scenarios: [beta, alpha], // initial unsorted
    });
    const user = userEvent.setup();
    renderWithProviders(<ScenariosPage />);
    await waitFor(() => expect(screen.getByTestId("scenario-table")).toBeInTheDocument());
    await user.selectOptions(screen.getByTestId("scenario-sort-key"), "flowCount");
    await user.click(screen.getByTestId("scenario-sort-dir"));
    const rows = screen.getAllByTestId(/^scenario-row-(alpha|beta)$/);
    expect(rows[0]).toHaveAttribute("data-testid", "scenario-row-alpha");
    expect(rows[1]).toHaveAttribute("data-testid", "scenario-row-beta");
  });

  it("invokes generateScenarioArtifacts for selected scenarios when Generate is clicked", async () => {
    const { fetchScenarios } = await import("../api/scenarios");
    const { generateScenarioArtifacts } = await import("../api/artifacts");
    vi.mocked(fetchScenarios).mockResolvedValue({
      vrooliRoot: "/repo",
      scenarios: [alpha, beta],
    });
    vi.mocked(generateScenarioArtifacts).mockResolvedValue({ scenarioId: "alpha", flows: [] });
    const user = userEvent.setup();
    renderWithProviders(<ScenariosPage />);
    await waitFor(() => expect(screen.getByTestId("scenario-table")).toBeInTheDocument());
    await user.click(screen.getByTestId("scenario-select-alpha"));
    await user.click(screen.getByTestId("scenario-generate-all"));
    await waitFor(() => expect(generateScenarioArtifacts).toHaveBeenCalledWith("alpha"));
    expect(generateScenarioArtifacts).not.toHaveBeenCalledWith("beta");
  });

  it("invokes generateScenarioArtifacts for every filtered scenario when nothing is selected", async () => {
    const { fetchScenarios } = await import("../api/scenarios");
    const { generateScenarioArtifacts } = await import("../api/artifacts");
    vi.mocked(fetchScenarios).mockResolvedValue({
      vrooliRoot: "/repo",
      scenarios: [alpha, beta],
    });
    vi.mocked(generateScenarioArtifacts).mockResolvedValue({ scenarioId: "alpha", flows: [] });
    const user = userEvent.setup();
    renderWithProviders(<ScenariosPage />);
    await waitFor(() => expect(screen.getByTestId("scenario-table")).toBeInTheDocument());
    await user.click(screen.getByTestId("scenario-generate-all"));
    await waitFor(() => expect(generateScenarioArtifacts).toHaveBeenCalledWith("alpha"));
    expect(generateScenarioArtifacts).toHaveBeenCalledWith("beta");
  });

  it("renders the diagnostic empty state when no scenarios are discovered", async () => {
    const { fetchScenarios } = await import("../api/scenarios");
    vi.mocked(fetchScenarios).mockResolvedValue({ vrooliRoot: "/repo", scenarios: [] });
    renderWithProviders(<ScenariosPage />);
    await waitFor(() =>
      expect(screen.getByTestId("scenarios-empty")).toBeInTheDocument(),
    );
  });

  it("renders a no-match hint when scenarios exist but filters exclude all", async () => {
    const { fetchScenarios } = await import("../api/scenarios");
    vi.mocked(fetchScenarios).mockResolvedValue({
      vrooliRoot: "/repo",
      scenarios: [alpha],
    });
    const user = userEvent.setup();
    renderWithProviders(<ScenariosPage />);
    await waitFor(() => expect(screen.getByTestId("scenario-table")).toBeInTheDocument());
    await user.type(screen.getByTestId("scenario-search"), "zzznomatch");
    await waitFor(() =>
      expect(screen.getByTestId("scenarios-no-match")).toBeInTheDocument(),
    );
  });

  it("renders the error state when the fetch fails", async () => {
    const { fetchScenarios } = await import("../api/scenarios");
    vi.mocked(fetchScenarios).mockRejectedValue(new Error("network"));
    renderWithProviders(<ScenariosPage />);
    await waitFor(() =>
      expect(screen.getByTestId("scenarios-error")).toBeInTheDocument(),
    );
  });
});
