/**
 * SearchPage integration — the composed search surface: SearchPanel results
 * render in place (single-page, no reload) and every executed query is
 * recorded into the HistoryPanel sidebar, which can replay it.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, screen, waitFor } from "@testing-library/react";

import { renderWithProviders } from "../test-utils";
import { selectors } from "../consts/selectors";
import { clearHistory } from "../lib/searchHistory";

vi.mock("../api/clients", () => ({
  liveSearchClient: { search: vi.fn() },
  findingsClient: { searchFindings: vi.fn() },
}));

import { liveSearchClient } from "../api/clients";
import { SearchPage } from "./SearchPage";

const liveResponse = {
  results: [
    {
      url: "https://example.com",
      title: "Example",
      snippet: "An example result.",
      engine: "duckduckgo",
      score: 0.9,
      category: "general",
    },
  ],
  synthesis: undefined,
  cached: false,
  degraded: false,
  degradedReason: "",
  degradedEngines: [],
};

const runQuery = (query: string) => {
  fireEvent.change(screen.getByTestId(selectors.search.input), {
    target: { value: query },
  });
  fireEvent.click(screen.getByTestId(selectors.search.submit));
};

describe("SearchPage", () => {
  beforeEach(() => {
    window.localStorage.clear();
    clearHistory();
  });
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("shows results in place and adds a history entry per executed query", async () => {
    vi.mocked(liveSearchClient.search).mockResolvedValue(liveResponse as never);

    renderWithProviders(<SearchPage />);

    runQuery("first query");
    expect(await screen.findAllByTestId(selectors.search.result)).toHaveLength(1);
    await waitFor(() => {
      expect(screen.getAllByTestId(selectors.history.item)).toHaveLength(1);
    });

    runQuery("second query");
    await waitFor(() => {
      expect(screen.getAllByTestId(selectors.history.item)).toHaveLength(2);
    });
    // Still the same single-page mount: results remain rendered alongside the
    // updated history — no navigation or reload happened.
    expect(screen.getAllByTestId(selectors.search.result)).toHaveLength(1);
    expect(screen.getByTestId(selectors.pages.search)).toBeInTheDocument();
  });

  it("replays a history entry through the search panel", async () => {
    vi.mocked(liveSearchClient.search).mockResolvedValue(liveResponse as never);

    renderWithProviders(<SearchPage />);
    runQuery("replay me");
    await waitFor(() => {
      expect(screen.getAllByTestId(selectors.history.item)).toHaveLength(1);
    });
    vi.mocked(liveSearchClient.search).mockClear();

    fireEvent.click(screen.getAllByTestId(selectors.history.item)[0] as HTMLElement);

    await waitFor(() => {
      expect(liveSearchClient.search).toHaveBeenCalledWith({
        query: "replay me",
        limit: 20,
        synthesize: false,
      });
    });
  });
});
