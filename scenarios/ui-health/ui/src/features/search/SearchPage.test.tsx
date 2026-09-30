import { describe, it, expect, vi, beforeEach } from "vitest";
import { screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import type { SearchHit, SearchResults } from "../../api/search";

const makeHit = (overrides: Partial<SearchHit> = {}): SearchHit => ({
  scenario: "ui-health",
  slot: "DashboardCard",
  kind: "component",
  displayName: "DashboardCard",
  description: "A card on the dashboard.",
  filePath: "ui/src/components/DashboardCard.tsx",
  score: 0.87,
  provenance: "custom",
  library: "",
  componentName: "",
  ...overrides,
});

vi.mock("../../api/search", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/search")>();
  const searchSurfaces = vi.fn((query: string): Promise<SearchResults> => {
    if (query === "empty") return Promise.resolve({ hits: [], modeUsed: "text" });
    return Promise.resolve({
      hits: [
        makeHit(),
        makeHit({ slot: "SettingsPage", displayName: "SettingsPage", kind: "page" }),
      ],
      modeUsed: "ai",
    });
  });
  return { ...actual, searchSurfaces };
});

import { SearchPage } from "./SearchPage";
import { searchSurfaces } from "../../api/search";

beforeEach(() => {
  vi.mocked(searchSurfaces).mockClear();
});

describe("SearchPage", () => {
  it("renders the empty initial state before typing", () => {
    renderWithProviders(<SearchPage />);
    expect(screen.getByTestId(selectors.search.input)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.search.empty)).toBeInTheDocument();
    expect(searchSurfaces).not.toHaveBeenCalled();
  });

  it("shows the short-query hint for a single character", async () => {
    const user = userEvent.setup();
    renderWithProviders(<SearchPage />);
    await user.type(screen.getByTestId(selectors.search.input), "a");
    expect(await screen.findByTestId(selectors.search.shortQuery)).toBeInTheDocument();
    // Should not call API for short queries
    expect(searchSurfaces).not.toHaveBeenCalled();
  });

  it("renders results after a debounced query", async () => {
    const user = userEvent.setup();
    renderWithProviders(<SearchPage />);
    await user.type(screen.getByTestId(selectors.search.input), "card");
    const list = await screen.findByTestId(selectors.search.resultsList);
    expect(list).toBeInTheDocument();
    const rows = within(list).getAllByRole("article");
    expect(rows).toHaveLength(2);
    await waitFor(() => expect(searchSurfaces).toHaveBeenCalledWith("card"));
  });

  it("filters results by kind", async () => {
    const user = userEvent.setup();
    renderWithProviders(<SearchPage />);
    await user.type(screen.getByTestId(selectors.search.input), "card");
    await screen.findByTestId(selectors.search.resultsList);

    await user.click(screen.getByTestId(selectors.search.kindFilter({ kind: "page" })));
    await waitFor(() => {
      const list = screen.getByTestId(selectors.search.resultsList);
      expect(within(list).getAllByRole("article")).toHaveLength(1);
    });
  });

  it("shows the no-results state when the API returns zero hits", async () => {
    const user = userEvent.setup();
    renderWithProviders(<SearchPage />);
    await user.type(screen.getByTestId(selectors.search.input), "empty");
    expect(await screen.findByTestId(selectors.search.noResults)).toBeInTheDocument();
  });

  it("shows the error state when the API throws", async () => {
    vi.mocked(searchSurfaces).mockRejectedValueOnce(new Error("boom"));
    const user = userEvent.setup();
    renderWithProviders(<SearchPage />);
    await user.type(screen.getByTestId(selectors.search.input), "broken");
    expect(await screen.findByTestId(selectors.search.error)).toBeInTheDocument();
  });
});
