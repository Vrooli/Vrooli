import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { afterEach, describe, expect, it, vi } from "vitest";

import { RelationshipsPanel } from "./RelationshipsPanel";

// provider-free-exception: graph panels use an isolated QueryClientProvider and MemoryRouter to test their typed read-model states without mounting the full application shell.
const getAssetRelationships = vi.hoisted(() => vi.fn());
vi.mock("../../api/catalogGraph", () => ({ getAssetRelationships }));

function renderPanel(assetId = "root") {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        <RelationshipsPanel assetId={assetId} />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

const result = {
  root: { assetId: "root", name: "Root", rung: 3 },
  directDependencies: [{ assetId: "dep", name: "Dependency", rung: 2 }],
  closure: [
    { assetId: "root", name: "Root", rung: 3 },
    { assetId: "dep", name: "Dependency", rung: 2 },
  ],
  closureBands: [
    {
      rung: 3,
      rungName: "component",
      count: 1,
      assets: [{ assetId: "root", name: "Root", rung: 3 }],
    },
    {
      rung: 2,
      rungName: "primitive",
      count: 1,
      assets: [{ assetId: "dep", name: "Dependency", rung: 2 }],
    },
  ],
  directDependents: [{ assetId: "consumer", name: "Consumer", rung: 4 }],
  transitiveDependents: [{ assetId: "consumer", name: "Consumer", rung: 4 }],
} as never;

describe("RelationshipsPanel", () => {
  afterEach(() => vi.clearAllMocks());

  it("renders a purposeful loading state", () => {
    getAssetRelationships.mockReturnValue(new Promise(() => undefined));
    renderPanel();
    expect(screen.getByTestId("relationships-panel")).toHaveAttribute("data-state", "loading");
    expect(screen.getByText("Loading asset relationships…")).toBeInTheDocument();
  });

  it("renders an error state", async () => {
    getAssetRelationships.mockRejectedValue(new Error("offline"));
    renderPanel();
    expect(await screen.findByRole("alert")).toHaveTextContent(
      "Unable to load asset relationships",
    );
    expect(screen.getByTestId("relationships-panel")).toHaveAttribute("data-state", "error");
  });

  it("renders an empty state", async () => {
    getAssetRelationships.mockResolvedValue(null);
    renderPanel();
    await waitFor(() =>
      expect(screen.getByTestId("relationships-panel")).toHaveAttribute("data-state", "empty"),
    );
    expect(screen.getByText("No relationship data")).toBeInTheDocument();
  });

  it("renders rung-labelled relationships and stable relationship links", async () => {
    getAssetRelationships.mockResolvedValue(result);
    renderPanel();
    expect(await screen.findByText("Depends on")).toBeInTheDocument();
    expect(screen.getByText("Used by")).toBeInTheDocument();
    expect(screen.getByText("Blast radius")).toBeInTheDocument();
    expect(screen.getAllByLabelText("Rung 2").length).toBeGreaterThan(0);
    expect(
      screen
        .getAllByRole("link", { name: /Dependency/ })
        .every((link) => link.getAttribute("href") === "/assets/dep?tab=relationships"),
    ).toBe(true);
    expect(screen.getByText("Rung 3 · component")).toBeInTheDocument();
  });
});
