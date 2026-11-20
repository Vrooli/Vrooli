import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import type { UseQueryResult } from "@tanstack/react-query";

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

vi.mock("../designs", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../designs")>();
  return {
    ...actual,
    listDesigns: vi.fn(),
  };
});

import App from "../../App";
import type { CapabilityDescription, ListNextWorkResponse } from "../../api/catalog";
import * as data from "../data";
import * as designs from "../designs";
import { LibraryPage } from "./LibraryPage";

function queryResult<T>(value: T | undefined): UseQueryResult<T> {
  return {
    data: value,
    error: null,
    isError: false,
    isLoading: false,
    isFetching: false,
    isSuccess: value !== undefined,
    refetch: vi.fn(),
  } as unknown as UseQueryResult<T>;
}

const asset = {
  id: "asset-button",
  libraryId: "react-component-library:Button",
  displayName: "Button",
  description: "The primary action control.",
  version: "2.2.14",
  latestVersion: "2.2.14",
  slug: "Button",
  slot: "controls",
  catalogId: "controls.button",
  catalogKind: "component",
  catalogDomain: "controls",
  tags: ["control"],
  metrics: { directAdoptionCount: 8, effectiveAdoptionCount: 80, versionCount: 2 },
  dependencies: [],
  designStyles: [],
  updatedAt: "2026-11-18T12:00:00Z",
  indexedAt: "2026-11-18T11:00:00Z",
} as unknown as data.CatalogAsset;

describe("LibraryPage", () => {
  beforeEach(() => {
    vi.mocked(data.useCatalog).mockReturnValue(queryResult([asset]));
    vi.mocked(data.useCoverage).mockReturnValue(queryResult<data.CoverageReport>(undefined));
    vi.mocked(data.useCapabilities).mockReturnValue(
      queryResult<CapabilityDescription>(undefined),
    );
    vi.mocked(data.useDesignScenarios).mockReturnValue(queryResult([]));
    vi.mocked(data.useNextWork).mockReturnValue(queryResult<ListNextWorkResponse>(undefined));
    vi.mocked(data.useDesignPages).mockReturnValue(queryResult([]));
    vi.mocked(designs.listDesigns).mockResolvedValue([]);
  });

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("renders the existing asset index at /library", async () => {
    renderWithProviders(<LibraryPage />);
    expect(screen.getByTestId("library-page")).toBeInTheDocument();
    expect(screen.getByTestId("components-page")).toBeInTheDocument();
    expect(await screen.findByText(asset.displayName)).toBeInTheDocument();
  });

  it("routes /library through the shell", async () => {
    renderWithProviders(<App />, { routerEntries: ["/library"] });
    await waitFor(() => expect(screen.getByTestId("library-page")).toBeInTheDocument());
    expect(screen.getByTestId("components-page")).toBeInTheDocument();
  });

  it("redirects legacy /components and /catalog to the library and keeps the query", async () => {
    renderWithProviders(<App />, { routerEntries: ["/components?q=Button"] });
    await waitFor(() => expect(screen.getByTestId("library-page")).toBeInTheDocument());
    expect(screen.getByTestId("components-page")).toBeInTheDocument();
    expect(screen.getByRole("searchbox")).toHaveValue("Button");
    cleanup();

    renderWithProviders(<App />, { routerEntries: ["/catalog"] });
    await waitFor(() => expect(screen.getByTestId("library-page")).toBeInTheDocument());
  });
});
