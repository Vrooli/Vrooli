import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import React from "react";
import { useClarificationStore } from "../../stores/clarification-store";
import type { ClarificationThread } from "../../types/domain";
import { ApiError } from "../../lib/api-client";

// Stub FloatingPanel to render children when open.
vi.mock("../ui/floating-panel", () => ({
  FloatingPanel: ({ children, isOpen }: { children: React.ReactNode; isOpen: boolean }) =>
    isOpen ? <div data-testid="floating-panel">{children}</div> : null,
}));

// Stub ClarificationMessages.
vi.mock("./clarification-messages", () => ({
  ClarificationMessages: ({ messages }: { messages: unknown[] }) => (
    <div data-testid="clarification-messages">{messages.length} messages</div>
  ),
}));

// Stub ClarificationActionButtons.
vi.mock("./clarification-action-buttons", () => ({
  ClarificationActionButtons: () => <div data-testid="action-buttons" />,
}));

// Stub CaptureAttachmentPreview.
vi.mock("../capture/capture-attachment-preview", () => ({
  CaptureAttachmentPreview: () => null,
}));

// Stub useAttachments.
vi.mock("../../hooks/useAttachments", () => ({
  useAttachments: () => ({
    attachments: [],
    addFile: vi.fn(),
    removeFile: vi.fn(),
    clearAll: vi.fn(),
    getFiles: () => [],
  }),
}));

// Mock backlog service.
const mockGetClarification = vi.fn();
const mockCreateClarification = vi.fn();
const mockContinueClarification = vi.fn();
const mockClarificationAction = vi.fn();

vi.mock("../../services/backlog-service", () => ({
  backlogService: {
    getClarification: (...args: unknown[]) => mockGetClarification(...args) as unknown,
    createClarification: (...args: unknown[]) => mockCreateClarification(...args) as unknown,
    continueClarification: (...args: unknown[]) => mockContinueClarification(...args) as unknown,
    clarificationAction: (...args: unknown[]) => mockClarificationAction(...args) as unknown,
  },
}));

// Import after mocks are set up.
import { ClarificationPanel } from "./clarification-panel";

const MOCK_TARGET = {
  backlogKind: "idea" as const,
  backlogName: "test-item",
  roundNumber: 1,
  itemId: "d1",
  itemTopic: "Test decision",
};

const MOCK_THREAD: ClarificationThread = {
  id: "thread-1",
  round_number: 1,
  item_id: "d1",
  run_id: "run-abc",
  messages: [
    { role: "user", content: "Why this approach?", created_at: "2026-04-01T00:00:00Z" },
    { role: "assistant", content: "Because it is simpler.", created_at: "2026-04-01T00:01:00Z" },
  ],
  status: "active",
  created_at: "2026-04-01T00:00:00Z",
  updated_at: "2026-04-01T00:01:00Z",
};

function resetStore() {
  useClarificationStore.setState({
    isOpen: false,
    target: null,
    thread: null,
    isCreating: false,
    isLoading: false,
  });
}

beforeEach(() => {
  resetStore();
  vi.clearAllMocks();
  localStorage.clear();
});

afterEach(() => {
  vi.useRealTimers();
});

describe("ClarificationPanel", () => {
  describe("thread resume", () => {
    it("renders loading spinner when isLoading is true", () => {
      useClarificationStore.setState({
        isOpen: true,
        target: { ...MOCK_TARGET, clarificationId: "thread-1" },
        isLoading: true,
      });
      mockGetClarification.mockReturnValue(new Promise(() => {})); // never resolves

      render(<ClarificationPanel />);
      expect(screen.getByTestId("clarification-loading")).toBeInTheDocument();
    });

    it("fetches thread on open when clarificationId is set", async () => {
      mockGetClarification.mockResolvedValue({ thread: MOCK_THREAD });

      useClarificationStore.setState({
        isOpen: true,
        target: { ...MOCK_TARGET, clarificationId: "thread-1" },
        isLoading: true,
      });

      render(<ClarificationPanel />);

      await waitFor(() => {
        expect(mockGetClarification).toHaveBeenCalledWith("idea", "test-item", "thread-1");
      });

      await waitFor(() => {
        expect(useClarificationStore.getState().thread).toEqual(MOCK_THREAD);
        expect(useClarificationStore.getState().isLoading).toBe(false);
      });
    });

    it("renders thread messages after successful fetch", async () => {
      mockGetClarification.mockResolvedValue({ thread: MOCK_THREAD });

      useClarificationStore.setState({
        isOpen: true,
        target: { ...MOCK_TARGET, clarificationId: "thread-1" },
        isLoading: true,
      });

      render(<ClarificationPanel />);

      await waitFor(() => {
        expect(screen.getByTestId("clarification-messages")).toBeInTheDocument();
      });
    });

    it("shows empty state on 404", async () => {
      mockGetClarification.mockRejectedValue(new ApiError("http", "Not found", { status: 404 }));

      useClarificationStore.setState({
        isOpen: true,
        target: { ...MOCK_TARGET, clarificationId: "thread-1" },
        isLoading: true,
      });

      render(<ClarificationPanel />);

      await waitFor(() => {
        expect(useClarificationStore.getState().isLoading).toBe(false);
        expect(useClarificationStore.getState().thread).toBeNull();
      });

      // Should show the empty prompt, not an error banner
      expect(screen.getByText(/Ask a question about this decision/)).toBeInTheDocument();
      expect(screen.queryByText(/Failed to load/)).toBeNull();
    });

    it("shows error banner on network failure", async () => {
      mockGetClarification.mockRejectedValue(new ApiError("network", "Failed to fetch"));

      useClarificationStore.setState({
        isOpen: true,
        target: { ...MOCK_TARGET, clarificationId: "thread-1" },
        isLoading: true,
      });

      render(<ClarificationPanel />);

      await waitFor(() => {
        expect(screen.getByText(/Unable to connect to the server/)).toBeInTheDocument();
      });
    });

    it("does not fetch when no clarificationId", () => {
      useClarificationStore.setState({
        isOpen: true,
        target: MOCK_TARGET,
        isLoading: false,
      });

      render(<ClarificationPanel />);
      expect(mockGetClarification).not.toHaveBeenCalled();
    });
  });

  describe("staleness", () => {
    it("does not show staleness banner before threshold", () => {
      vi.useFakeTimers();

      useClarificationStore.setState({
        isOpen: true,
        target: MOCK_TARGET,
        thread: {
          ...MOCK_THREAD,
          messages: [{ role: "user", content: "Hello", created_at: "2026-04-01T00:00:00Z" }],
        },
        isLoading: false,
      });

      render(<ClarificationPanel />);

      vi.advanceTimersByTime(80_000); // 80s < 90s threshold
      expect(screen.queryByTestId("staleness-warning")).toBeNull();
    });

    it("shows staleness banner after threshold", async () => {
      vi.useFakeTimers();
      // Polling calls getClarification repeatedly — return unchanged thread.
      const userOnlyThread: ClarificationThread = {
        ...MOCK_THREAD,
        messages: [{ role: "user", content: "Hello", created_at: "2026-04-01T00:00:00Z" }],
      };
      mockGetClarification.mockResolvedValue({ thread: userOnlyThread });

      useClarificationStore.setState({
        isOpen: true,
        target: MOCK_TARGET,
        thread: userOnlyThread,
        isLoading: false,
      });

      render(<ClarificationPanel />);

      await vi.advanceTimersByTimeAsync(91_000);
      expect(screen.getByTestId("staleness-warning")).toBeInTheDocument();
    });

    it("clears staleness when agent responds", async () => {
      vi.useFakeTimers();

      const userOnlyThread: ClarificationThread = {
        ...MOCK_THREAD,
        messages: [{ role: "user", content: "Hello", created_at: "2026-04-01T00:00:00Z" }],
      };
      mockGetClarification.mockResolvedValue({ thread: userOnlyThread });

      useClarificationStore.setState({
        isOpen: true,
        target: MOCK_TARGET,
        thread: userOnlyThread,
        isLoading: false,
      });

      const { rerender } = render(<ClarificationPanel />);

      await vi.advanceTimersByTimeAsync(91_000);
      expect(screen.getByTestId("staleness-warning")).toBeInTheDocument();

      // Simulate agent response arriving via polling.
      useClarificationStore.setState({
        thread: MOCK_THREAD, // has assistant message — isWaitingForAgent becomes false
      });

      rerender(<ClarificationPanel />);
      expect(screen.queryByTestId("staleness-warning")).toBeNull();
    });
  });

  describe("input state", () => {
    it("textarea is disabled while isLoading is true", () => {
      useClarificationStore.setState({
        isOpen: true,
        target: { ...MOCK_TARGET, clarificationId: "thread-1" },
        isLoading: true,
      });
      mockGetClarification.mockReturnValue(new Promise(() => {}));

      render(<ClarificationPanel />);

      const textarea = screen.getByPlaceholderText(/What would you like to know/);
      expect(textarea).toBeDisabled();
    });
  });
});
