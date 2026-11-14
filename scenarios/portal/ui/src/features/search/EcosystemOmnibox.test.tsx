import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { selectors } from "../../consts/selectors";
import { renderWithProviders } from "../../test-utils";
import { EcosystemOmnibox } from "./EcosystemOmnibox";

const searchApi = vi.hoisted(() => ({
  suggest: vi.fn(),
}));

vi.mock("../../api/search", () => searchApi);

describe("EcosystemOmnibox", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    cleanup();
  });

  it("waits for enough input before suggesting", async () => {
    const user = userEvent.setup();
    renderWithProviders(<EcosystemOmnibox />);

    await user.type(screen.getByTestId(selectors.search.input), "a");
    await new Promise((resolve) => window.setTimeout(resolve, 300));

    expect(searchApi.suggest).not.toHaveBeenCalled();
    expect(screen.queryByTestId(selectors.search.status)).not.toBeInTheDocument();
  });

  it("renders degraded search results and inserts selected references", async () => {
    const user = userEvent.setup();
    searchApi.suggest.mockResolvedValueOnce({
      degraded: true,
      reason: "search budget exhausted",
      hits: [{
        providerId: "docs",
        type: "doc",
        title: "Portal Guide",
        snippet: "Chat-first front door",
        path: "scenarios/portal/README.md",
      }],
    });
    renderWithProviders(<EcosystemOmnibox />);

    await user.type(screen.getByTestId(selectors.search.input), "portal");

    expect(await screen.findByTestId(selectors.search.result({ index: 0 }))).toHaveTextContent("Portal Guide");
    expect(screen.getByTestId(selectors.search.status)).toHaveTextContent("search budget exhausted");
    await user.click(screen.getByTestId(selectors.search.result({ index: 0 })));

    await waitFor(() => {
      expect(screen.getByTestId(selectors.search.input)).toHaveValue("portal scenarios/portal/README.md");
    });
  });

  it("shows an error status when suggest fails", async () => {
    const user = userEvent.setup();
    searchApi.suggest.mockRejectedValueOnce(new Error("search unavailable"));
    renderWithProviders(<EcosystemOmnibox />);

    await user.type(screen.getByTestId(selectors.search.input), "portal");

    await waitFor(() => {
      expect(screen.getByTestId(selectors.search.status)).toHaveTextContent("search unavailable");
    });
  });
});
