import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, screen, waitFor, within } from "@testing-library/react";
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

vi.mock("../workspaces/useWorkspaces", () => ({
  useWorkspaces: vi.fn(),
}));

vi.mock("../search/useWorkbenchSearch", () => ({
  useWorkbenchSearch: vi.fn(),
}));

vi.mock("../../api/review", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/review")>();
  return {
    ...actual,
    getReviewSummary: vi.fn(),
  };
});

import App from "../../App";
import type { CapabilityDescription, ListNextWorkResponse } from "../../api/catalog";
import { SearchAvailability, SearchKind, type SearchResult } from "../../api/search";
import {
  WorkspaceProvenance,
  type WorkspaceSummary,
} from "../../api/workspaces";
import { strings } from "../../consts/strings";
import * as data from "../data";
import * as reviewApi from "../../api/review";
import * as searchHook from "../search/useWorkbenchSearch";
import * as workspaceHook from "../workspaces/useWorkspaces";
import { StartPage } from "./StartPage";

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

const WORKSPACE_NAME = "Harbor";

const workspaces: WorkspaceSummary[] = [
  {
    $typeName: "vrooli.react_component_library.v1.workspace.WorkspaceSummary",
    id: "ws_harbor",
    name: WORKSPACE_NAME,
    summary: "Four-step onboarding",
    provenance: WorkspaceProvenance.AUTHORED,
    tags: [],
    bindingScenario: "harbor",
    bindingPage: "welcome",
    pageCount: 4,
    displayCount: 12,
    stateCount: 16,
    revision: 2,
    createdAt: "2026-11-18T10:00:00Z",
    updatedAt: "2026-11-18T12:00:00Z",
    originWorkspaceId: "",
  },
  {
    $typeName: "vrooli.react_component_library.v1.workspace.WorkspaceSummary",
    id: "ws_curated",
    name: "Curated onboarding",
    summary: "",
    provenance: WorkspaceProvenance.CURATED,
    tags: [],
    bindingScenario: "",
    bindingPage: "",
    pageCount: 2,
    displayCount: 6,
    stateCount: 8,
    revision: 1,
    createdAt: "2026-11-17T10:00:00Z",
    updatedAt: "2026-11-17T12:00:00Z",
    originWorkspaceId: "",
  },
];

const searchResult: SearchResult = {
  $typeName: "vrooli.react_component_library.v1.search.SearchResult",
  id: "asset-button",
  kind: SearchKind.ASSET,
  title: "Button",
  subtitle: "controls.button",
  snippet: "",
  href: "/library?q=button",
  score: 1,
  availability: SearchAvailability.AVAILABLE,
  badges: [],
  relations: [],
};

function LocationProbe() {
  const location = useLocation();
  return (
    <div data-testid="location-probe">
      {location.pathname}
      {location.search}
    </div>
  );
}

describe("StartPage", () => {
  beforeEach(() => {
    vi.mocked(data.useCatalog).mockReturnValue(queryResult(assets()));
    vi.mocked(data.useCoverage).mockReturnValue(queryResult<data.CoverageReport>(undefined));
    vi.mocked(data.useCapabilities).mockReturnValue(
      queryResult<CapabilityDescription>(undefined),
    );
    vi.mocked(data.useDesignScenarios).mockReturnValue(queryResult([]));
    vi.mocked(data.useNextWork).mockReturnValue(queryResult<ListNextWorkResponse>(undefined));
    vi.mocked(data.useDesignPages).mockReturnValue(queryResult([]));
    vi.mocked(workspaceHook.useWorkspaces).mockReturnValue(queryResult(workspaces));
    vi.mocked(reviewApi.getReviewSummary).mockResolvedValue(null);
    vi.mocked(searchHook.useWorkbenchSearch).mockReturnValue({
      response: null,
      results: [],
      loading: false,
      error: null,
    });
  });

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("derives category cards from catalog domains", async () => {
    renderWithProviders(<StartPage />);
    expect(screen.getByTestId("start-page")).toBeInTheDocument();

    const cards = await screen.findAllByTestId("start-category-card");
    expect(cards).toHaveLength(2);
    const controls = screen.getByRole("link", { name: /controls/i });
    expect(controls).toHaveAttribute("href", "/library?q=controls");
  });

  it("opens the first search result's destination", async () => {
    vi.mocked(searchHook.useWorkbenchSearch).mockReturnValue({
      response: null,
      results: [searchResult],
      loading: false,
      error: null,
    });
    renderWithProviders(
      <>
        <StartPage />
        <LocationProbe />
      </>,
    );
    const input = screen.getByRole("textbox", { name: strings.workbench.search.label });
    fireEvent.change(input, { target: { value: "button" } });
    fireEvent.keyDown(input, { key: "Enter" });

    await waitFor(() => {
      expect(screen.getByTestId("location-probe")).toHaveTextContent("/library?q=button");
    });
  });

  it("shows loading skeletons before the catalog and workspaces answer", () => {
    vi.mocked(data.useCatalog).mockReturnValue(
      queryResult<data.CatalogAsset[]>(undefined, { isLoading: true }),
    );
    vi.mocked(workspaceHook.useWorkspaces).mockReturnValue(
      queryResult<WorkspaceSummary[]>(undefined, { isLoading: true }),
    );
    renderWithProviders(<StartPage />);

    expect(screen.getByTestId("start-categories-loading")).toBeInTheDocument();
    expect(screen.getByTestId("start-designs-loading")).toBeInTheDocument();
  });

  it("shows honest empty states when no assets or workspaces exist", async () => {
    vi.mocked(data.useCatalog).mockReturnValue(queryResult([]));
    vi.mocked(workspaceHook.useWorkspaces).mockReturnValue(queryResult([]));
    renderWithProviders(<StartPage />);

    expect(
      await screen.findByText(strings.workbench.start.emptyCategoriesTitle),
    ).toBeInTheDocument();
    expect(screen.getByText(strings.workbench.start.noDesigns)).toBeInTheDocument();
  });

  it("lists existing workspaces with a continuation link", async () => {
    renderWithProviders(<StartPage />);

    // The continuation rail and the recently-opened strip both name the
    // workspace; either is a correct place to see it.
    expect((await screen.findAllByText(WORKSPACE_NAME)).length).toBeGreaterThan(0);
    const openLinks = screen.getAllByRole("link", { name: /workbench.start.openWorkspace/i });
    expect(openLinks[0]).toHaveAttribute("href", "/design?workspace=ws_harbor");
  });

  it("honours the shortcuts the Start rail advertises", async () => {
    renderWithProviders(
      <>
        <App />
        <LocationProbe />
      </>,
      { routerEntries: ["/"] },
    );
    await screen.findByTestId("start-page");

    fireEvent.keyDown(window, { key: "g" });
    fireEvent.keyDown(window, { key: "d" });
    await waitFor(() =>
      expect(screen.getByTestId("location-probe")).toHaveTextContent("/design"),
    );

    fireEvent.keyDown(window, { key: "n" });
    await waitFor(() =>
      expect(screen.getByTestId("location-probe")).toHaveTextContent("/design?new=1"),
    );
  });

  it("does not steal G sequences while the person is typing", () => {
    renderWithProviders(
      <>
        <StartPage />
        <LocationProbe />
      </>,
    );
    const input = screen.getByRole("textbox", { name: strings.workbench.search.label });
    input.focus();
    fireEvent.keyDown(input, { key: "g" });
    fireEvent.keyDown(input, { key: "d" });
    expect(screen.getByTestId("location-probe").textContent).toBe("/");
  });

  it("exposes exactly five primary destinations on desktop and mobile", async () => {
    renderWithProviders(<App />, { routerEntries: ["/"] });
    await screen.findByTestId("start-page");

    const rail = screen.getByRole("navigation", { name: strings.workbench.nav.label });
    const railLinks = within(rail).getAllByRole("link");
    expect(railLinks).toHaveLength(5);
    expect(railLinks[0]).toHaveAttribute("aria-current", "page");
    expect(railLinks.map((link) => link.getAttribute("aria-label") ?? link.textContent)).toEqual([
      expect.stringContaining(strings.workbench.nav.start),
      expect.stringContaining(strings.workbench.nav.library),
      expect.stringContaining(strings.workbench.nav.designShort),
      expect.stringContaining(strings.workbench.nav.review),
      expect.stringContaining(strings.workbench.nav.stats),
    ]);
    expect(within(rail).queryByText(strings.workbench.nav.settings)).not.toBeInTheDocument();

    const mobile = screen.getByRole("navigation", { name: strings.workbench.nav.mobile });
    expect(within(mobile).getAllByRole("link")).toHaveLength(5);
  });
});

function assets(): data.CatalogAsset[] {
  return [
    {
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
    },
    {
      id: "asset-chart",
      libraryId: "react-component-library:Chart",
      displayName: "Chart",
      description: "A measured signal.",
      version: "1.4.0",
      latestVersion: "1.4.0",
      slug: "Chart",
      slot: "data",
      catalogId: "data.chart",
      catalogKind: "component",
      catalogDomain: "data",
      tags: ["data"],
      metrics: { directAdoptionCount: 3, effectiveAdoptionCount: 12, versionCount: 1 },
      dependencies: [],
      designStyles: [],
      updatedAt: "2026-11-17T12:00:00Z",
      indexedAt: "2026-11-17T11:00:00Z",
    },
  ] as unknown as data.CatalogAsset[];
}
