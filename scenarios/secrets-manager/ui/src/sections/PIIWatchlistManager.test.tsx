import { cleanup, fireEvent, screen, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../test-utils";
import { PIIWatchlistManager } from "./PIIWatchlistManager";

const apiMocks = vi.hoisted(() => ({
  fetchWatchlist: vi.fn(),
  createWatchlistEntry: vi.fn(),
  deleteWatchlistEntry: vi.fn()
}));

vi.mock("../lib/api", async (importOriginal) => ({
  ...(await importOriginal<typeof import("../lib/api")>()),
  fetchWatchlist: apiMocks.fetchWatchlist,
  createWatchlistEntry: apiMocks.createWatchlistEntry,
  deleteWatchlistEntry: apiMocks.deleteWatchlistEntry
}));

function renderWatchlist() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false }
    }
  });

  return renderWithProviders(
    <QueryClientProvider client={queryClient}>
      <PIIWatchlistManager />
    </QueryClientProvider>
  );
}

describe("PIIWatchlistManager", () => {
  beforeEach(() => {
    apiMocks.fetchWatchlist.mockResolvedValue({ entries: [], count: 0 });
    apiMocks.createWatchlistEntry.mockResolvedValue({ id: "entry-2" });
    apiMocks.deleteWatchlistEntry.mockResolvedValue(undefined);
  });

  afterEach(cleanup);

  it("validates, creates, and deletes encrypted watchlist entries", async () => {
    apiMocks.fetchWatchlist.mockResolvedValue({ entries: [{ id: "entry-1", label: "Work email", value_type: "email", created_at: "2026-07-23T00:00:00Z" }], count: 1 });
    renderWatchlist();
    expect(await screen.findByText("Work email")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Add entry" }));
    expect(await screen.findByText("Label and value are required")).toBeInTheDocument();
    fireEvent.change(screen.getByPlaceholderText("e.g. My work email"), { target: { value: "  Personal email " } });
    fireEvent.change(screen.getByPlaceholderText("Encrypted on save"), { target: { value: "private@example.test" } });
    fireEvent.change(screen.getByRole("combobox"), { target: { value: "custom" } });
    fireEvent.click(screen.getByRole("button", { name: "Add entry" }));
    await waitFor(() => expect(apiMocks.createWatchlistEntry).toHaveBeenCalledWith({ label: "Personal email", value: "private@example.test", value_type: "custom" }));
    fireEvent.click(screen.getByRole("button", { name: "Delete watchlist entry Work email" }));
    expect(screen.getByText("Delete watchlist entry?")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Delete" }));
    await waitFor(() => expect(apiMocks.deleteWatchlistEntry).toHaveBeenCalledWith("entry-1"));
  });

  it("fails closed when the watchlist encryption key is unavailable", async () => {
    apiMocks.fetchWatchlist.mockRejectedValue(new Error("watchlist unavailable (503)"));
    renderWatchlist();
    expect(await screen.findByText("Watchlist encryption key is not configured.")).toBeInTheDocument();
    expect(screen.getByText("Encryption key not configured")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Add entry" })).not.toBeInTheDocument();
  });
});
