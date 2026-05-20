import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, screen, waitFor } from "@testing-library/react";

import { renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";

vi.mock("../../api/clients", () => ({
  searchClient: { status: vi.fn(), search: vi.fn() },
  validationClient: { validateScenario: vi.fn() },
  reindexClient: { reindex: vi.fn() },
}));

import { reindexClient, searchClient } from "../../api/clients";
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

  it("clicking reindex triggers a non-dry-run reconcile", async () => {
    vi.mocked(searchClient.status).mockResolvedValue({
      available: false,
      ollama: false,
      qdrant: false,
      indexedCount: 0,
      lastReconcileAt: "",
      lastReconcileOutcome: "",
    } as never);
    vi.mocked(reindexClient.reindex).mockResolvedValue({
      jobId: "job-1",
      plannedUpserts: 5,
      plannedDeletes: 1,
      dryRun: false,
    } as never);

    renderWithProviders(<StatusPanel />);

    await waitFor(() =>
      expect(screen.getByTestId(selectors.status.available)).toHaveTextContent(
        "status.availableNo",
      ),
    );
    fireEvent.click(screen.getByTestId(selectors.status.reindex));

    await waitFor(() => {
      expect(reindexClient.reindex).toHaveBeenCalledWith({
        scenario: "",
        dryRun: false,
      });
    });
    expect(await screen.findByTestId(selectors.status.reindexed)).toBeInTheDocument();
  });

  it("renders an error when status() rejects", async () => {
    vi.mocked(searchClient.status).mockRejectedValue(new Error("status boom"));

    renderWithProviders(<StatusPanel />);
    expect(await screen.findByTestId(selectors.status.error)).toBeInTheDocument();
  });
});
