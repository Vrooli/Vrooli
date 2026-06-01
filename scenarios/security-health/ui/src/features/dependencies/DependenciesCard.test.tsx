import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { create } from "@bufbuild/protobuf";
import {
  DependencyRecordSchema,
  SearchResponseSchema,
  SearchResultSchema,
  StatusResponseSchema,
} from "@vrooli/proto-types/security-health/v1/dependencies/dependencies_pb";

import { renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";

vi.mock("../../api/dependencies", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/dependencies")>();
  return { ...actual, dependencyClient: { status: vi.fn(), search: vi.fn() } };
});

import { DependenciesCard } from "./DependenciesCard";
import { dependencyClient, Ecosystem, Mode } from "../../api/dependencies";

const mockStatus = vi.mocked(dependencyClient.status);
const mockSearch = vi.mocked(dependencyClient.search);

describe("DependenciesCard", () => {
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("renders the index status strip", async () => {
    mockStatus.mockResolvedValue(
      create(StatusResponseSchema, { available: true, indexedCount: 658, vulnerableCount: 4 }),
    );
    renderWithProviders(<DependenciesCard />);
    // Counts are interpolated and not rendered under i18n cimode; assert presence.
    await waitFor(() => expect(screen.getByTestId(selectors.dependencies.status)).toBeInTheDocument());
  });

  it("renders the index coverage line in both building and ready states", async () => {
    // Building: ready=false → the coverage span carries the amber building class.
    mockStatus.mockResolvedValue(
      create(StatusResponseSchema, {
        available: true,
        indexedVectors: 4123,
        expectedVectors: 4390,
        indexReady: false,
      }),
    );
    const { unmount } = renderWithProviders(<DependenciesCard />);
    const building = await screen.findByTestId(selectors.dependencies.coverage);
    expect(building).toBeInTheDocument();
    expect(building.className).toContain("amber");
    unmount();
    cleanup();

    // Ready: ready=true → no building class.
    mockStatus.mockResolvedValue(
      create(StatusResponseSchema, {
        available: true,
        indexedVectors: 4390,
        expectedVectors: 4390,
        indexReady: true,
      }),
    );
    renderWithProviders(<DependenciesCard />);
    const ready = await screen.findByTestId(selectors.dependencies.coverage);
    expect(ready).toBeInTheDocument();
    expect(ready.className).not.toContain("amber");
  });

  it("runs a search and renders vulnerable results plus the text-mode hint", async () => {
    mockStatus.mockResolvedValue(create(StatusResponseSchema, { available: true, indexedCount: 1, vulnerableCount: 1 }));
    mockSearch.mockResolvedValue(
      create(SearchResponseSchema, {
        modeUsed: Mode.TEXT,
        results: [
          create(SearchResultSchema, {
            score: 0.9,
            record: create(DependencyRecordSchema, {
              scenario: "web-console",
              ecosystem: Ecosystem.NPM,
              name: "esbuild",
              version: "0.19.0",
              vulnIds: ["GHSA-xxxx"],
            }),
          }),
        ],
      }),
    );

    const user = userEvent.setup();
    renderWithProviders(<DependenciesCard />);
    await user.click(screen.getByTestId(selectors.dependencies.searchButton));

    await waitFor(() => expect(screen.getByTestId(selectors.dependencies.results)).toBeInTheDocument());
    const table = screen.getByTestId(selectors.dependencies.results);
    expect(within(table).getByText("esbuild")).toBeInTheDocument();
    expect(within(table).getByText("GHSA-xxxx")).toBeInTheDocument();
    expect(screen.getByTestId(selectors.dependencies.modeHint)).toBeInTheDocument();
  });

  it("shows the empty state when a search returns nothing", async () => {
    mockStatus.mockResolvedValue(create(StatusResponseSchema, { available: true }));
    mockSearch.mockResolvedValue(create(SearchResponseSchema, { modeUsed: Mode.AI, results: [] }));

    const user = userEvent.setup();
    renderWithProviders(<DependenciesCard />);
    await user.click(screen.getByTestId(selectors.dependencies.searchButton));

    await waitFor(() => expect(screen.getByTestId(selectors.dependencies.empty)).toBeInTheDocument());
  });
});
