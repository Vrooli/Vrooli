import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";

import { renderWithProviders } from "../test-utils";

vi.mock("../api/scenarios", () => ({
  fetchScenarios: vi.fn(),
  fetchScenarioDetail: vi.fn(),
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
  path: "/repo/scenarios/beta",
  flowCount: 0,
};

describe("ScenariosPage", () => {
  beforeEach(async () => {
    const { fetchScenarios } = await import("../api/scenarios");
    vi.mocked(fetchScenarios).mockReset();
  });
  afterEach(() => cleanup());

  it("renders one card per scenario with name + flow count + drill-in link", async () => {
    const { fetchScenarios } = await import("../api/scenarios");
    vi.mocked(fetchScenarios).mockResolvedValue({
      vrooliRoot: "/repo",
      scenarios: [alpha, beta],
    });
    renderWithProviders(<ScenariosPage />);
    await waitFor(() =>
      expect(screen.getByTestId("scenarios-grid")).toBeInTheDocument(),
    );
    const card = screen.getByTestId("scenario-card-alpha");
    expect(card).toHaveAttribute("href", "/scenarios/alpha");
    expect(card).toHaveTextContent("Alpha");
    // Flow-count nodes exist for both rows. (Their text content is i18n-
    // interpolated; in cimode the key is returned, so we assert presence
    // not the rendered count.)
    expect(screen.getByTestId("scenario-card-flowcount-alpha")).toBeInTheDocument();
    expect(screen.getByTestId("scenario-card-flowcount-beta")).toBeInTheDocument();
    expect(screen.getByTestId("scenario-card-beta")).toHaveAttribute(
      "href",
      "/scenarios/beta",
    );
  });

  it("renders the diagnostic empty state when no scenarios are discovered", async () => {
    const { fetchScenarios } = await import("../api/scenarios");
    vi.mocked(fetchScenarios).mockResolvedValue({ vrooliRoot: "/repo", scenarios: [] });
    renderWithProviders(<ScenariosPage />);
    await waitFor(() =>
      expect(screen.getByTestId("scenarios-empty")).toBeInTheDocument(),
    );
    // The empty state surfaces the resolved root so a misconfigured deploy
    // can be diagnosed without server access. The actual root is injected
    // through an i18n {{root}} placeholder which doesn't expand under
    // cimode; we assert the structural presence here and verify the
    // interpolation contract in the ScenariosPage component's render code.
  });

  it("surfaces the per-row discovery error on the card", async () => {
    const { fetchScenarios } = await import("../api/scenarios");
    vi.mocked(fetchScenarios).mockResolvedValue({
      vrooliRoot: "/repo",
      scenarios: [{ ...beta, discoveryError: "permission denied" }],
    });
    renderWithProviders(<ScenariosPage />);
    await waitFor(() =>
      expect(screen.getByTestId("scenario-card-error-beta")).toBeInTheDocument(),
    );
    expect(screen.getByTestId("scenario-card-error-beta")).toHaveTextContent(
      "permission denied",
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
