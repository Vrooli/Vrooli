import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, screen, waitFor } from "@testing-library/react";
import type { UseQueryResult } from "@tanstack/react-query";
import { useLocation } from "react-router-dom";

import { renderWithProviders } from "../../test-utils";

vi.mock("../data", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../data")>();
  return {
    ...actual,
    useCatalog: vi.fn(),
    useCoverage: vi.fn(),
    useCapabilities: vi.fn(),
    useDesignScenarios: vi.fn(),
    useNextWork: vi.fn(),
    useDesignPages: vi.fn(),
  };
});

vi.mock("../../api/libraryHealth", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/libraryHealth")>();
  return {
    ...actual,
    getLibraryHealth: vi.fn(),
  };
});

import App from "../../App";
import type { CapabilityDescription, ListNextWorkResponse } from "../../api/catalog";
import * as healthApi from "../../api/libraryHealth";
import { strings } from "../../consts/strings";
import * as data from "../data";
import { StatsPage } from "./StatsPage";

function queryResult<T>(
  value: T | undefined,
  options: { isLoading?: boolean; isError?: boolean } = {},
): UseQueryResult<T> {
  return {
    data: value,
    error: options.isError ? new Error("unavailable") : null,
    isError: options.isError ?? false,
    isLoading: options.isLoading ?? false,
    isFetching: false,
    isSuccess: !options.isError && value !== undefined,
    refetch: vi.fn(),
  } as unknown as UseQueryResult<T>;
}

function LocationProbe() {
  const location = useLocation();
  return (
    <div data-testid="location-probe">
      {location.pathname}
      {location.search}
    </div>
  );
}

describe("StatsPage", () => {
  beforeEach(() => {
    vi.mocked(data.useCatalog).mockReturnValue(queryResult([]));
    vi.mocked(data.useCoverage).mockReturnValue(queryResult<data.CoverageReport>(undefined));
    vi.mocked(data.useCapabilities).mockReturnValue(
      queryResult<CapabilityDescription>(undefined),
    );
    vi.mocked(data.useDesignScenarios).mockReturnValue(queryResult([]));
    vi.mocked(data.useNextWork).mockReturnValue(queryResult<ListNextWorkResponse>(undefined));
    vi.mocked(data.useDesignPages).mockReturnValue(queryResult([]));
    vi.mocked(healthApi.getLibraryHealth).mockResolvedValue(null);
  });

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("defaults to the Overview tab", () => {
    renderWithProviders(<StatsPage />, { routerEntries: ["/stats"] });
    expect(screen.getByTestId("stats-page")).toBeInTheDocument();
    expect(screen.getByTestId("stats-overview")).toBeInTheDocument();
    expect(screen.getByRole("tabpanel")).toBeInTheDocument();
    expect(screen.getByRole("tab", { name: "workbench.stats.tabOverview" })).toHaveAttribute(
      "aria-selected",
      "true",
    );
  });

  it("maps the view query to Quality, Activity and Services", () => {
    renderWithProviders(<StatsPage />, { routerEntries: ["/stats?view=quality"] });
    expect(screen.getByTestId("coverage-page")).toBeInTheDocument();
    cleanup();

    renderWithProviders(<StatsPage />, { routerEntries: ["/stats?view=activity"] });
    expect(screen.getByTestId("activity-page")).toBeInTheDocument();
    cleanup();

    renderWithProviders(<StatsPage />, { routerEntries: ["/stats?view=services"] });
    expect(screen.getByTestId("capabilities-page")).toBeInTheDocument();
  });

  it("writes the selected tab back to the query string", async () => {
    renderWithProviders(
      <>
        <StatsPage />
        <LocationProbe />
      </>,
      { routerEntries: ["/stats"] },
    );

    fireEvent.click(screen.getByRole("tab", { name: "workbench.stats.tabQuality" }));
    await waitFor(() => {
      expect(screen.getByTestId("location-probe")).toHaveTextContent("/stats?view=quality");
      expect(screen.getByTestId("coverage-page")).toBeInTheDocument();
    });
  });

  it("renders every section's unavailable state without inventing zero", async () => {
    vi.mocked(data.useCoverage).mockReturnValue(
      queryResult<data.CoverageReport>(undefined, { isError: true }),
    );
    vi.mocked(data.useCapabilities).mockReturnValue(
      queryResult<CapabilityDescription>(undefined, { isError: true }),
    );
    vi.mocked(healthApi.getLibraryHealth).mockRejectedValue(new Error("unavailable"));

    renderWithProviders(<StatsPage />, { routerEntries: ["/stats"] });
    expect(screen.getByTestId("stats-overview")).toBeInTheDocument();
    expect(screen.getByText(strings.workbench.stats.qualityUnavailable)).toBeInTheDocument();
    expect(
      await screen.findByText(strings.workbench.stats.healthUnavailable),
    ).toBeInTheDocument();
    expect(screen.getByText(strings.workbench.stats.activityUnavailable)).toBeInTheDocument();
    expect(screen.getByText(strings.workbench.stats.servicesUnavailable)).toBeInTheDocument();
  });

  it("redirects legacy measurement routes to their Stats views through the shell", async () => {
    renderWithProviders(<App />, { routerEntries: ["/coverage"] });
    await waitFor(() => expect(screen.getByTestId("stats-page")).toBeInTheDocument());
    expect(screen.getByTestId("coverage-page")).toBeInTheDocument();
    cleanup();

    renderWithProviders(<App />, { routerEntries: ["/activity"] });
    await waitFor(() => expect(screen.getByTestId("stats-page")).toBeInTheDocument());
    expect(screen.getByTestId("activity-page")).toBeInTheDocument();
    cleanup();

    renderWithProviders(<App />, { routerEntries: ["/capabilities"] });
    await waitFor(() => expect(screen.getByTestId("stats-page")).toBeInTheDocument());
    expect(screen.getByTestId("capabilities-page")).toBeInTheDocument();
  });
});
