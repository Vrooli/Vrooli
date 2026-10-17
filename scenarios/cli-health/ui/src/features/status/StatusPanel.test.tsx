import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";

import { renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";

vi.mock("../../api/clients", () => ({
  searchClient: { status: vi.fn(), search: vi.fn() },
  validationClient: { validateScenario: vi.fn() },
}));

import { searchClient } from "../../api/clients";
import { StatusPanel } from "./StatusPanel";

describe("StatusPanel", () => {
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("renders availability + counts from search.status()", async () => {
    vi.mocked(searchClient.status).mockResolvedValue({
      available: true,
      ollama: true,
      qdrant: true,
      indexedCount: 283,
      lastReconcileAt: "2026-05-19T09:00:00Z",
      lastReconcileOutcome: "ok",
    } as never);

    renderWithProviders(<StatusPanel />);

    await waitFor(() =>
      expect(screen.getByTestId(selectors.status.available)).toHaveTextContent(
        "status.availableYes",
      ),
    );
    expect(screen.getByTestId(selectors.status.indexed)).toHaveTextContent("283");
    expect(screen.getByTestId(selectors.status.ollama)).toHaveTextContent("status.yes");
    expect(screen.getByTestId(selectors.status.qdrant)).toHaveTextContent("status.yes");
  });

  it("does not expose the removed unauthenticated reindex action", async () => {
    vi.mocked(searchClient.status).mockResolvedValue({
      available: false,
      ollama: false,
      qdrant: false,
      indexedCount: 0,
      lastReconcileAt: "",
      lastReconcileOutcome: "",
    } as never);

    renderWithProviders(<StatusPanel />);

    await waitFor(() =>
      expect(screen.getByTestId(selectors.status.available)).toHaveTextContent(
        "status.availableNo",
      ),
    );
    expect(screen.queryByTestId(selectors.status.reindex)).not.toBeInTheDocument();
    expect(screen.queryByTestId(selectors.status.reindexed)).not.toBeInTheDocument();
  });

  it("renders an error when status() rejects", async () => {
    vi.mocked(searchClient.status).mockRejectedValue(new Error("status boom"));

    renderWithProviders(<StatusPanel />);
    expect(await screen.findByTestId(selectors.status.error)).toBeInTheDocument();
  });
});
