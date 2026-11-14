import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import { Route, Routes } from "react-router-dom";

vi.mock("../api/graph", () => ({
  graphClient: {
    getGraphSnapshot: vi.fn(),
    listGraphSnapshots: vi.fn(),
  },
}));

vi.mock("../api/domains", () => ({
  domainsClient: {
    getDomainMap: vi.fn().mockResolvedValue({ domainMap: { domains: [] } }),
  },
}));

vi.mock("../api/conflicts", () => ({
  conflictsClient: {
    listConflicts: vi.fn().mockResolvedValue({ conflicts: [], nextPageToken: "" }),
  },
}));

import { graphClient } from "../api/graph";
import { renderWithProviders } from "../test-utils";
import { selectors } from "../consts/selectors";
import { TargetGraphPage } from "./TargetGraphPage";

type ListResult = Awaited<ReturnType<typeof graphClient.listGraphSnapshots>>;

afterEach(() => {
  cleanup();
  vi.mocked(graphClient.listGraphSnapshots).mockReset();
});

const Wrapper = () => (
  <Routes>
    <Route path="/targets/:encodedPath/graph" element={<TargetGraphPage />} />
  </Routes>
);

describe("TargetGraphPage", () => {
  it("renders the page heading + filter bar when no snapshot exists", async () => {
    vi.mocked(graphClient.listGraphSnapshots).mockResolvedValue({
      snapshots: [],
      nextPageToken: "",
    } as unknown as ListResult);

    renderWithProviders(<Wrapper />, { routerEntries: ["/targets/demo/graph"] });

    expect(await screen.findByTestId(selectors.pages.targetGraph)).toBeInTheDocument();
    // Even an empty snapshot renders the legend.
    await waitFor(() =>
      expect(screen.getByTestId(selectors.features.graph.legend.root)).toBeInTheDocument(),
    );
  });
});
