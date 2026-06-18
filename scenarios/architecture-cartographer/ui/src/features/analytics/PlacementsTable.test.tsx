import { cleanup, fireEvent, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

vi.mock("./controllers/useAnalyticsController", () => ({
  usePlacements: vi.fn(),
}));

import { selectors } from "../../consts/selectors";
import { renderWithProviders } from "../../test-utils";
import { PlacementsTable } from "./PlacementsTable";
import { usePlacements } from "./controllers/useAnalyticsController";

afterEach(() => {
  cleanup();
  vi.mocked(usePlacements).mockReset();
});

function mockPlacements(state: Partial<ReturnType<typeof usePlacements>>) {
  vi.mocked(usePlacements).mockReturnValue({
    isPending: false,
    isError: false,
    data: { placements: [] },
    error: null,
    refetch: vi.fn(),
    ...state,
  } as unknown as ReturnType<typeof usePlacements>);
}

describe("PlacementsTable", () => {
  it("renders loading, retryable error, and empty states", () => {
    mockPlacements({ isPending: true });
    const { rerender } = renderWithProviders(<PlacementsTable scenario="demo" />);
    expect(screen.getByTestId(selectors.features.analytics.placements.loading)).toBeInTheDocument();

    const refetch = vi.fn();
    mockPlacements({ isError: true, error: "placements unavailable" as never, refetch });
    rerender(<PlacementsTable scenario="demo" />);
    expect(screen.getByTestId(selectors.features.analytics.placements.error)).toBeInTheDocument();
    expect(screen.getByText("placements unavailable")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /retry/i }));
    expect(refetch).toHaveBeenCalledTimes(1);

    mockPlacements({ data: { placements: [] } as never });
    rerender(<PlacementsTable scenario="demo" />);
    expect(screen.getByTestId(selectors.features.analytics.placements.empty)).toBeInTheDocument();
  });

  it("renders placement rows and falls back for blank outcomes", () => {
    mockPlacements({
      data: {
        placements: [
          {
            id: "placement-1",
            chunkId: "chunk-a",
            chunkPath: "api/internal/foo/foo.go",
            outcome: "auto_place",
          },
          {
            id: "placement-2",
            chunkId: "chunk-b",
            chunkPath: "api/internal/bar/bar.go",
            outcome: "",
          },
        ],
      } as never,
    });

    renderWithProviders(<PlacementsTable scenario="demo" />);

    expect(screen.getByTestId(selectors.features.analytics.placements.root)).toBeInTheDocument();
    expect(screen.getByText("chunk-a")).toBeInTheDocument();
    expect(screen.getByText("api/internal/foo/foo.go")).toBeInTheDocument();
    expect(screen.getByText("auto_place")).toBeInTheDocument();
    expect(screen.getByText("—")).toBeInTheDocument();
  });
});
