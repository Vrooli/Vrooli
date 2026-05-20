import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, screen, waitFor } from "@testing-library/react";

import { renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { Mode } from "@vrooli/proto-types/cli-health/v1/search/search_pb";

vi.mock("../../api/clients", () => ({
  searchClient: {
    search: vi.fn(),
  },
  validationClient: { validateScenario: vi.fn() },
  reindexClient: { reindex: vi.fn() },
}));

import { searchClient } from "../../api/clients";
import { SearchPanel } from "./SearchPanel";

describe("SearchPanel", () => {
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("submits a query and renders results", async () => {
    vi.mocked(searchClient.search).mockResolvedValue({
      results: [
        {
          scenario: "cli-health",
          group: "validate",
          name: "scenario",
          description: "Validate a scenario manifest.",
          score: 0.872,
          source: "manifest",
        },
      ],
      modeUsed: Mode.AI,
    } as never);

    renderWithProviders(<SearchPanel />);

    fireEvent.change(screen.getByTestId(selectors.search.input), {
      target: { value: "validate manifest" },
    });
    fireEvent.click(screen.getByTestId(selectors.search.submit));

    await waitFor(() => {
      expect(searchClient.search).toHaveBeenCalledWith({
        query: "validate manifest",
        limit: 20,
        mode: Mode.UNSPECIFIED,
      });
    });
    expect(await screen.findAllByTestId(selectors.search.result)).toHaveLength(1);
    expect(screen.getByTestId(selectors.search.modeUsed)).toBeInTheDocument();
  });

  it("switching to text mode propagates Mode.TEXT to the client", async () => {
    vi.mocked(searchClient.search).mockResolvedValue({
      results: [],
      modeUsed: Mode.TEXT,
    } as never);

    renderWithProviders(<SearchPanel />);

    fireEvent.click(screen.getByTestId(selectors.search.modeText));
    fireEvent.change(screen.getByTestId(selectors.search.input), {
      target: { value: "q" },
    });
    fireEvent.click(screen.getByTestId(selectors.search.submit));

    await waitFor(() => {
      expect(searchClient.search).toHaveBeenCalledWith(
        expect.objectContaining({ mode: Mode.TEXT }),
      );
    });
    expect(await screen.findByTestId(selectors.search.empty)).toBeInTheDocument();
  });

  it("renders an error when the client rejects", async () => {
    vi.mocked(searchClient.search).mockRejectedValue(new Error("backend down"));

    renderWithProviders(<SearchPanel />);
    fireEvent.change(screen.getByTestId(selectors.search.input), {
      target: { value: "q" },
    });
    fireEvent.click(screen.getByTestId(selectors.search.submit));

    expect(await screen.findByTestId(selectors.search.error)).toBeInTheDocument();
  });
});
