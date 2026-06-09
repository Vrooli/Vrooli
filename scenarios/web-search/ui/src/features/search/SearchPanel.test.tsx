import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, screen, waitFor } from "@testing-library/react";

import { renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";

vi.mock("../../api/clients", () => ({
  liveSearchClient: { search: vi.fn() },
  findingsClient: { searchFindings: vi.fn() },
}));

import { findingsClient, liveSearchClient } from "../../api/clients";
import { SearchPanel } from "./SearchPanel";

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
};

describe("SearchPanel — live web", () => {
  beforeEach(() => {
    window.localStorage.clear();
  });
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("runs a live web search and renders snippet results", async () => {
    vi.mocked(liveSearchClient.search).mockResolvedValue(liveResponse as never);

    renderWithProviders(<SearchPanel />);
    fireEvent.change(screen.getByTestId(selectors.search.input), {
      target: { value: "vrooli" },
    });
    fireEvent.click(screen.getByTestId(selectors.search.submit));

    await waitFor(() => {
      expect(liveSearchClient.search).toHaveBeenCalledWith({
        query: "vrooli",
        limit: 20,
        synthesize: false,
      });
    });
    expect(await screen.findAllByTestId(selectors.search.result)).toHaveLength(1);
    expect(screen.getByTestId(selectors.search.cached)).toBeInTheDocument();
  });

  it("passes the synthesize flag and renders the synthesis block", async () => {
    vi.mocked(liveSearchClient.search).mockResolvedValue({
      ...liveResponse,
      synthesis: { text: "A synthesized answer.", citations: [], abstained: false },
    } as never);

    renderWithProviders(<SearchPanel />);
    fireEvent.click(screen.getByTestId(selectors.search.synthesizeToggle));
    fireEvent.change(screen.getByTestId(selectors.search.input), {
      target: { value: "vrooli" },
    });
    fireEvent.click(screen.getByTestId(selectors.search.submit));

    await waitFor(() => {
      expect(liveSearchClient.search).toHaveBeenCalledWith({
        query: "vrooli",
        limit: 20,
        synthesize: true,
      });
    });
    expect(await screen.findByTestId(selectors.search.synthesis)).toBeInTheDocument();
  });

  it("surfaces degraded_reason when the response is degraded", async () => {
    vi.mocked(liveSearchClient.search).mockResolvedValue({
      ...liveResponse,
      results: [],
      degraded: true,
      degradedReason: "budget exhausted",
    } as never);

    renderWithProviders(<SearchPanel />);
    fireEvent.change(screen.getByTestId(selectors.search.input), {
      target: { value: "vrooli" },
    });
    fireEvent.click(screen.getByTestId(selectors.search.submit));

    expect(await screen.findByTestId(selectors.search.degraded)).toHaveTextContent(
      strings.search.degraded,
    );
  });

  it("does not call the client for a blank query", () => {
    renderWithProviders(<SearchPanel />);
    fireEvent.click(screen.getByTestId(selectors.search.submit));
    expect(liveSearchClient.search).not.toHaveBeenCalled();
  });

  it("renders an error state when the live search fails", async () => {
    vi.mocked(liveSearchClient.search).mockRejectedValue(new Error("boom"));

    renderWithProviders(<SearchPanel />);
    fireEvent.change(screen.getByTestId(selectors.search.input), {
      target: { value: "vrooli" },
    });
    fireEvent.click(screen.getByTestId(selectors.search.submit));

    expect(await screen.findByTestId(selectors.search.error)).toBeInTheDocument();
  });
});

describe("SearchPanel — learnings", () => {
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("searches the findings corpus and shows method + hits", async () => {
    vi.mocked(findingsClient.searchFindings).mockResolvedValue({
      hits: [{ finding: { id: "f1", claim: "A learned claim", status: 1 }, score: 0.8, weak: false }],
      method: "hybrid",
    } as never);

    renderWithProviders(<SearchPanel />);
    fireEvent.click(screen.getByTestId(selectors.search.modeLearnings));
    fireEvent.change(screen.getByTestId(selectors.search.input), {
      target: { value: "claim" },
    });
    fireEvent.click(screen.getByTestId(selectors.search.submit));

    await waitFor(() => {
      expect(findingsClient.searchFindings).toHaveBeenCalledWith({
        query: "claim",
        limit: 20,
        includeArchived: false,
      });
    });
    expect(await screen.findByTestId(selectors.search.method)).toBeInTheDocument();
    expect(screen.getAllByTestId(selectors.search.findingHit)).toHaveLength(1);
  });

  it("shows the learnings empty state when there are no hits", async () => {
    vi.mocked(findingsClient.searchFindings).mockResolvedValue({
      hits: [],
      method: "dense",
    } as never);

    renderWithProviders(<SearchPanel />);
    fireEvent.click(screen.getByTestId(selectors.search.modeLearnings));
    fireEvent.change(screen.getByTestId(selectors.search.input), {
      target: { value: "nothing" },
    });
    fireEvent.click(screen.getByTestId(selectors.search.submit));

    expect(await screen.findByTestId(selectors.search.empty)).toHaveTextContent(
      strings.search.emptyLearnings,
    );
  });
});
