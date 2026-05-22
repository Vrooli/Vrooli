import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";

vi.mock("../api/graph", () => ({
  graphClient: {
    listGraphSnapshots: vi.fn().mockResolvedValue({ snapshots: [], nextPageToken: "" }),
    extractGraph: vi.fn(),
  },
}));
vi.mock("../api/health", () => ({
  fetchHealth: vi.fn().mockResolvedValue({
    status: "ok",
    service: "architecture-cartographer-api",
    timestamp: new Date(0).toISOString(),
  }),
}));

import { renderWithProviders } from "../test-utils";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { OverviewPage } from "./OverviewPage";

afterEach(() => cleanup());

describe("OverviewPage", () => {
  it("renders the overview heading and a link to start a new extraction", async () => {
    renderWithProviders(<OverviewPage />);
    expect(screen.getByTestId(selectors.pages.overview)).toBeInTheDocument();
    // cimode renders the key path; we only assert the link target.
    expect(
      screen.getByRole("link", { name: strings.pages.overview.startExtraction }),
    ).toHaveAttribute("href", "/targets/new");
    // Empty state inside the snapshots panel confirms the query ran.
    await waitFor(() =>
      expect(
        screen.getByTestId(selectors.features.targets.activeSnapshots.empty),
      ).toBeInTheDocument(),
    );
    // Empty state inside the recent targets list confirms the recent
    // localStorage seam started clean.
    expect(
      screen.getByTestId(selectors.features.targets.recent.empty),
    ).toBeInTheDocument();
  });
});
