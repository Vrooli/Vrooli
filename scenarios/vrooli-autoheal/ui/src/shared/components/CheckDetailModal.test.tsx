// CheckDetailModal component tests
// [REQ:UI-EVENTS-001] [REQ:PERSIST-HISTORY-001]
import { describe, it, expect, vi, beforeEach } from "vitest";
import { screen, fireEvent, waitFor } from "@testing-library/react";
import { CheckDetailModal } from "./CheckDetailModal";
import * as api from "../../lib/api";
import * as exportUtils from "../../lib/export";
import { renderWithProviders } from "../../test-utils";
import {
  createActionResult,
  createCheckActionsResponse,
  createCheckHistoryResponse,
  createConfig,
  createDefaultsResponse,
  createHistoryEntry,
} from "../../test-utils";
import {
  resetMockCheckMetadata,
  setMockCheckMetadata,
} from "../../test-utils/mocks/checkMetadataContext";

vi.mock("../contexts/CheckMetadataContext", async () => {
  const { useMockCheckMetadata } = await import(
    "../../test-utils/mocks/checkMetadataContext"
  );
  return {
    useCheckMetadata: useMockCheckMetadata,
  };
});

// Mock the API module
vi.mock("../../lib/api", async () => {
  const actual = await vi.importActual("../../lib/api");
  return {
    ...actual,
    fetchCheckHistory: vi.fn(),
    fetchConfig: vi.fn(),
    fetchDefaults: vi.fn(),
    fetchCheckActions: vi.fn(),
    setCheckAutoHeal: vi.fn(),
    executeAction: vi.fn(),
  };
});

// Mock the export module
vi.mock("../../lib/export", () => ({
  exportCheckHistoryToCSV: vi.fn(),
}));

describe("[REQ:UI-EVENTS-001] CheckDetailModal", () => {
  const mockOnClose = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
    resetMockCheckMetadata();
    vi.mocked(api.fetchConfig).mockResolvedValue(createConfig());
    vi.mocked(api.fetchDefaults).mockResolvedValue(createDefaultsResponse());
    vi.mocked(api.fetchCheckActions).mockResolvedValue(createCheckActionsResponse());
  });

  it("renders with check ID in title", async () => {
    vi.mocked(api.fetchCheckHistory).mockResolvedValue(createCheckHistoryResponse());

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
    vi.mocked(api.fetchCheckHistory).mockResolvedValue(
      createCheckHistoryResponse({
        history: [
          createHistoryEntry({
            message: "All systems operational",
          }),
        ],
      })
    );

    renderWithProviders(
      <CheckDetailModal checkId="test-check" onClose={mockOnClose} />
    );

    await waitFor(() => {
      expect(screen.getByText("All systems operational")).toBeInTheDocument();
    });
  });

  it("displays stats summary", async () => {
    vi.mocked(api.fetchCheckHistory).mockResolvedValue(
      createCheckHistoryResponse({
        history: [
          createHistoryEntry({ message: "OK" }),
          createHistoryEntry({ message: "OK", timestamp: "2024-01-01T11:00:00Z" }),
          createHistoryEntry({
            status: "warning",
            message: "Warn",
            timestamp: "2024-01-01T10:00:00Z",
          }),
        ],
      })
    );

    renderWithProviders(
      <CheckDetailModal checkId="test-check" onClose={mockOnClose} />
    );

    await waitFor(() => {
      expect(screen.getByText("3")).toBeInTheDocument(); // Total
      expect(screen.getByText("2")).toBeInTheDocument(); // OK count
    });
  });

  it("calls onClose when close button clicked", async () => {
    vi.mocked(api.fetchCheckHistory).mockResolvedValue(createCheckHistoryResponse());

    renderWithProviders(
      <CheckDetailModal checkId="test-check" onClose={mockOnClose} />
    );

    fireEvent.click(screen.getByLabelText(/close modal/i));
    expect(mockOnClose).toHaveBeenCalled();
  });

  it("calls onClose when escape key pressed", async () => {
    vi.mocked(api.fetchCheckHistory).mockResolvedValue(createCheckHistoryResponse());

    renderWithProviders(
      <CheckDetailModal checkId="test-check" onClose={mockOnClose} />
    );

    fireEvent.keyDown(document, { key: "Escape" });
    expect(mockOnClose).toHaveBeenCalled();
  });

  it("calls onClose when backdrop clicked", async () => {
    vi.mocked(api.fetchCheckHistory).mockResolvedValue(createCheckHistoryResponse());

    renderWithProviders(
      <CheckDetailModal checkId="test-check" onClose={mockOnClose} />
    );

    const modal = screen.getByTestId("check-detail-modal");
    fireEvent.click(modal);
    expect(mockOnClose).toHaveBeenCalled();
  });

  it("triggers export when export button clicked", async () => {
    vi.mocked(api.fetchCheckHistory).mockResolvedValue(
      createCheckHistoryResponse({
        history: [createHistoryEntry({ message: "Systems operational" })],
      })
    );

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
    vi.mocked(api.fetchCheckHistory).mockResolvedValue(createCheckHistoryResponse());

    renderWithProviders(
      <CheckDetailModal checkId="test-check" onClose={mockOnClose} />
    );

    const historyTab = await screen.findByRole("button", { name: /^History/ });
    fireEvent.click(historyTab);

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

  it("renders importance notice when metadata includes importance", async () => {
    setMockCheckMetadata({
      "test-check": {
        importance: "Maintains service uptime during resource failures.",
        description: "metadata description",
        category: "resource",
      },
    });
    vi.mocked(api.fetchCheckHistory).mockResolvedValue(
      createCheckHistoryResponse({
        history: [
          createHistoryEntry({
            status: "warning",
            message: "Potential issue detected",
          }),
        ],
      })
    );

    renderWithProviders(<CheckDetailModal checkId="test-check" onClose={mockOnClose} />);

    await waitFor(() => {
      expect(screen.getByText("Why This Matters")).toBeInTheDocument();
      expect(screen.getByText(/maintains service uptime/i)).toBeInTheDocument();
    });
  });

  it("shows confirmation notice for dangerous heal actions", async () => {
    vi.mocked(api.fetchCheckHistory).mockResolvedValue(
      createCheckHistoryResponse({
        history: [
          createHistoryEntry({
            status: "critical",
            message: "Service is down",
          }),
        ],
      })
    );
    vi.mocked(api.fetchCheckActions).mockResolvedValue(
      createCheckActionsResponse({
        actions: [
          {
            id: "restart",
            name: "Restart Service",
            description: "Restarts the service process immediately.",
            available: true,
            dangerous: true,
          },
        ],
      })
    );

    renderWithProviders(<CheckDetailModal checkId="test-check" onClose={mockOnClose} />);

    const healButton = await screen.findByRole("button", { name: /heal now/i });
    fireEvent.click(healButton);

    await waitFor(() => {
      expect(screen.getByText("Confirm Action")).toBeInTheDocument();
      expect(screen.getByText(/restart service/i)).toBeInTheDocument();
    });
  });

  it("shows action result notice after heal execution", async () => {
    vi.mocked(api.fetchCheckHistory).mockResolvedValue(
      createCheckHistoryResponse({
        history: [
          createHistoryEntry({
            status: "warning",
            message: "Service may be degraded",
          }),
        ],
      })
    );
    vi.mocked(api.fetchCheckActions).mockResolvedValue(
      createCheckActionsResponse({
        actions: [
          {
            id: "heal",
            name: "Heal",
            description: "Attempts recovery",
            available: true,
            dangerous: false,
          },
        ],
      })
    );
    vi.mocked(api.executeAction).mockResolvedValue(
      createActionResult({
        actionId: "heal",
        success: false,
        message: "Recovery failed",
        error: "Service unreachable",
        duration: 123,
      })
    );

    renderWithProviders(<CheckDetailModal checkId="test-check" onClose={mockOnClose} />);

    const healButton = await screen.findByRole("button", { name: /heal now/i });
    fireEvent.click(healButton);

    await waitFor(() => {
      expect(screen.getByText("Recovery failed")).toBeInTheDocument();
      expect(screen.getByText("Service unreachable")).toBeInTheDocument();
      expect(screen.getByRole("button", { name: /dismiss/i })).toBeInTheDocument();
    });
  });
});
