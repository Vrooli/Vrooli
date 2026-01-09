// CheckDetailModal component tests
// [REQ:UI-EVENTS-001] [REQ:PERSIST-HISTORY-001]
import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { CheckDetailModal } from "./CheckDetailModal";
import * as api from "../lib/api";
import * as exportUtils from "../lib/export";

// Mock the API module
vi.mock("../lib/api", async () => {
  const actual = await vi.importActual("../lib/api");
  return {
    ...actual,
    fetchCheckHistory: vi.fn(),
  };
});

// Mock the export module
vi.mock("../lib/export", () => ({
  exportCheckHistoryToCSV: vi.fn(),
}));

const createTestQueryClient = () =>
  new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
        gcTime: 0,
      },
    },
  });

const renderWithProviders = (ui: React.ReactElement) => {
  const queryClient = createTestQueryClient();
  return render(
    <QueryClientProvider client={queryClient}>{ui}</QueryClientProvider>
  );
};

describe("[REQ:UI-EVENTS-001] CheckDetailModal", () => {
  const mockOnClose = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders with check ID in title", async () => {
    vi.mocked(api.fetchCheckHistory).mockResolvedValue({
      checkId: "test-check",
      history: [],
      count: 0,
    });

    renderWithProviders(
      <CheckDetailModal checkId="test-check" onClose={mockOnClose} />
    );

    expect(screen.getByText("test-check")).toBeInTheDocument();
  });

  it("shows loading state while fetching history", async () => {
    vi.mocked(api.fetchCheckHistory).mockImplementation(
      () => new Promise(() => {})
    );

    renderWithProviders(
      <CheckDetailModal checkId="test-check" onClose={mockOnClose} />
    );

    expect(screen.getByText(/loading history/i)).toBeInTheDocument();
  });

  it("displays history entries", async () => {
    vi.mocked(api.fetchCheckHistory).mockResolvedValue({
      checkId: "test-check",
      history: [
        {
          checkId: "test-check",
          status: "ok",
          message: "All systems operational",
          timestamp: "2024-01-01T12:00:00Z",
          duration: 100,
        },
      ],
      count: 1,
    });

    renderWithProviders(
      <CheckDetailModal checkId="test-check" onClose={mockOnClose} />
    );

    await waitFor(() => {
      expect(screen.getByText("All systems operational")).toBeInTheDocument();
    });
  });

  it("displays stats summary", async () => {
    vi.mocked(api.fetchCheckHistory).mockResolvedValue({
      checkId: "test-check",
      history: [
        { checkId: "test-check", status: "ok", message: "OK", timestamp: "2024-01-01T12:00:00Z", duration: 100 },
        { checkId: "test-check", status: "ok", message: "OK", timestamp: "2024-01-01T11:00:00Z", duration: 100 },
        { checkId: "test-check", status: "warning", message: "Warn", timestamp: "2024-01-01T10:00:00Z", duration: 100 },
      ],
      count: 3,
    });

    renderWithProviders(
      <CheckDetailModal checkId="test-check" onClose={mockOnClose} />
    );

    await waitFor(() => {
      expect(screen.getByText("3")).toBeInTheDocument(); // Total
      expect(screen.getByText("2")).toBeInTheDocument(); // OK count
    });
  });

  it("calls onClose when close button clicked", async () => {
    vi.mocked(api.fetchCheckHistory).mockResolvedValue({
      checkId: "test-check",
      history: [],
      count: 0,
    });

    renderWithProviders(
      <CheckDetailModal checkId="test-check" onClose={mockOnClose} />
    );

    fireEvent.click(screen.getByLabelText(/close modal/i));
    expect(mockOnClose).toHaveBeenCalled();
  });

  it("calls onClose when escape key pressed", async () => {
    vi.mocked(api.fetchCheckHistory).mockResolvedValue({
      checkId: "test-check",
      history: [],
      count: 0,
    });

    renderWithProviders(
      <CheckDetailModal checkId="test-check" onClose={mockOnClose} />
    );

    fireEvent.keyDown(document, { key: "Escape" });
    expect(mockOnClose).toHaveBeenCalled();
  });

  it("calls onClose when backdrop clicked", async () => {
    vi.mocked(api.fetchCheckHistory).mockResolvedValue({
      checkId: "test-check",
      history: [],
      count: 0,
    });

    renderWithProviders(
      <CheckDetailModal checkId="test-check" onClose={mockOnClose} />
    );

    const modal = screen.getByTestId("check-detail-modal");
    fireEvent.click(modal);
    expect(mockOnClose).toHaveBeenCalled();
  });

  it("triggers export when export button clicked", async () => {
    vi.mocked(api.fetchCheckHistory).mockResolvedValue({
      checkId: "test-check",
      history: [
        { checkId: "test-check", status: "ok", message: "Systems operational", timestamp: "2024-01-01T12:00:00Z", duration: 100 },
      ],
      count: 1,
    });

    renderWithProviders(
      <CheckDetailModal checkId="test-check" onClose={mockOnClose} />
    );

    // Wait for data to load - use the unique message
    await waitFor(() => {
      expect(screen.getByText("Systems operational")).toBeInTheDocument();
    });

    // Find and click the export button (it's the one with "Export" text in the header)
    const exportButton = screen.getByTitle(/export history/i);
    fireEvent.click(exportButton);
    expect(exportUtils.exportCheckHistoryToCSV).toHaveBeenCalled();
  });

  it("shows empty state when no history", async () => {
    vi.mocked(api.fetchCheckHistory).mockResolvedValue({
      checkId: "test-check",
      history: [],
      count: 0,
    });

    renderWithProviders(
      <CheckDetailModal checkId="test-check" onClose={mockOnClose} />
    );

    await waitFor(() => {
      expect(screen.getByText(/no history available/i)).toBeInTheDocument();
    });
  });

  it("shows error state when fetch fails", async () => {
    vi.mocked(api.fetchCheckHistory).mockRejectedValue(new Error("Network error"));

    renderWithProviders(
      <CheckDetailModal checkId="test-check" onClose={mockOnClose} />
    );

    await waitFor(() => {
      expect(screen.getByText(/network error/i)).toBeInTheDocument();
    });
  });
});
