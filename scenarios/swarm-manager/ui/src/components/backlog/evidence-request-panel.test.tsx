import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { EvidenceRequestPanel } from "./evidence-request-panel";
import { useReviewStore } from "../../stores/review-store";
import type { ReviewRound } from "../../services/review-service";

// Mock reviewService
const mockRequestMoreEvidence = vi.fn<() => Promise<{ thread_id: string }>>();
const mockContinueRequest = vi.fn<() => Promise<void>>();
const mockDismissRequest = vi.fn<() => Promise<void>>();

vi.mock("../../services/review-service", () => ({
  reviewService: {
    requestMoreEvidence: (...args: unknown[]) => mockRequestMoreEvidence(...(args as [])),
    continueRequest: (...args: unknown[]) => mockContinueRequest(...(args as [])),
    dismissRequest: (...args: unknown[]) => mockDismissRequest(...(args as [])),
  },
}));

// Mock Drawer to render children inline for testing
vi.mock("../ui/drawer", () => ({
  Drawer: ({
    isOpen,
    children,
    footer,
    testId,
  }: {
    isOpen: boolean;
    children: React.ReactNode;
    footer?: React.ReactNode;
    testId?: string;
  }) =>
    isOpen ? (
      <div data-testid={testId}>
        {children}
        {footer}
      </div>
    ) : null,
}));

// Mock EvidenceRequestMessages to simplify testing
vi.mock("./evidence-request-messages", () => ({
  EvidenceRequestMessages: ({ messages, isWaitingForAgent }: { messages: unknown[]; isWaitingForAgent: boolean }) => (
    <div data-testid="evidence-request-messages">
      {messages.length} messages
      {isWaitingForAgent && <span>waiting</span>}
    </div>
  ),
}));

const mockRound: ReviewRound = {
  round: 1,
  generated_at: "2026-04-01T10:00:00Z",
  execution_id: "exec-1",
  status: "complete",
  evidence: [],
  request_threads: [
    {
      id: "rt-1",
      status: "pending",
      messages: [
        { role: "user", content: "Need screenshots", timestamp: "2026-04-01T10:00:00Z" },
        { role: "assistant", content: "Here they are", timestamp: "2026-04-01T10:01:00Z" },
      ],
      created_at: "2026-04-01T10:00:00Z",
    },
  ],
};

beforeEach(() => {
  vi.clearAllMocks();
  // Reset store.
  useReviewStore.setState({
    requestPanelOpen: false,
    requestTarget: null,
    activeThreadId: null,
    activeThread: null,
    isCreating: false,
    isSending: false,
  });
  // Clear localStorage.
  try {
    localStorage.removeItem("swarm-evidence-request-draft");
  } catch { /* ignore */ }
});

function renderPanel(rounds: ReviewRound[] = [mockRound]) {
  const onAction = vi.fn();
  const result = render(
    <EvidenceRequestPanel
      backlogKind="fix"
      backlogName="my-item"
      reviewRounds={rounds}
      onAction={onAction}
    />,
  );
  return { ...result, onAction };
}

describe("EvidenceRequestPanel", () => {
  it("renders nothing when panel is closed", () => {
    renderPanel();
    expect(screen.queryByTestId("evidence-request-panel")).toBeNull();
  });

  it("renders panel when store is open", () => {
    useReviewStore.getState().openRequestPanel(1);
    renderPanel();
    expect(screen.getByTestId("evidence-request-panel")).toBeTruthy();
  });

  it("shows empty state when no messages", () => {
    useReviewStore.getState().openRequestPanel(1);
    renderPanel();
    expect(screen.getByText(/Describe what additional evidence/)).toBeTruthy();
  });

  it("creates new thread on first submit", async () => {
    mockRequestMoreEvidence.mockResolvedValue({ thread_id: "rt-new" });
    useReviewStore.getState().openRequestPanel(1, "ev-5");

    const { onAction } = renderPanel();

    const input = screen.getByTestId("evidence-request-input");
    fireEvent.change(input, { target: { value: "Need more screenshots" } });
    fireEvent.click(screen.getByTestId("evidence-request-send"));

    await waitFor(() => expect(mockRequestMoreEvidence).toHaveBeenCalledTimes(1));
    expect(mockRequestMoreEvidence).toHaveBeenCalledWith(
      "fix", "my-item", 1, "Need more screenshots", "ev-5",
    );
    expect(onAction).toHaveBeenCalled();
  });

  it("continues existing thread on follow-up submit", async () => {
    mockContinueRequest.mockResolvedValue(undefined);

    // Open panel with an existing thread.
    useReviewStore.getState().openRequestPanel(1);
    useReviewStore.getState().setActiveThread(mockRound.request_threads?.[0] ?? null);

    const { onAction } = renderPanel();

    const input = screen.getByTestId("evidence-request-input");
    fireEvent.change(input, { target: { value: "Also check the config" } });
    fireEvent.click(screen.getByTestId("evidence-request-send"));

    await waitFor(() => expect(mockContinueRequest).toHaveBeenCalledTimes(1));
    expect(mockContinueRequest).toHaveBeenCalledWith(
      "fix", "my-item", 1, "rt-1", "Also check the config",
    );
    expect(onAction).toHaveBeenCalled();
  });

  it("dismisses thread and closes panel", async () => {
    mockDismissRequest.mockResolvedValue(undefined);

    useReviewStore.getState().openRequestPanel(1);
    useReviewStore.getState().setActiveThread(mockRound.request_threads?.[0] ?? null);

    const { onAction } = renderPanel();

    fireEvent.click(screen.getByTestId("evidence-request-dismiss"));

    await waitFor(() => expect(mockDismissRequest).toHaveBeenCalledTimes(1));
    expect(onAction).toHaveBeenCalled();
    // Panel should close.
    expect(useReviewStore.getState().requestPanelOpen).toBe(false);
  });

  it("shows error on API failure", async () => {
    mockRequestMoreEvidence.mockRejectedValue(new Error("Network error"));
    useReviewStore.getState().openRequestPanel(1);

    renderPanel();

    const input = screen.getByTestId("evidence-request-input");
    fireEvent.change(input, { target: { value: "test" } });
    fireEvent.click(screen.getByTestId("evidence-request-send"));

    await waitFor(() => {
      expect(screen.getByTestId("evidence-request-error")).toBeTruthy();
    });
    expect(screen.getByText("Network error")).toBeTruthy();
  });

  it("submits via Ctrl+Enter", async () => {
    mockRequestMoreEvidence.mockResolvedValue({ thread_id: "rt-new" });
    useReviewStore.getState().openRequestPanel(1);

    renderPanel();

    const input = screen.getByTestId("evidence-request-input");
    fireEvent.change(input, { target: { value: "test" } });
    fireEvent.keyDown(input, { key: "Enter", ctrlKey: true });

    await waitFor(() => expect(mockRequestMoreEvidence).toHaveBeenCalledTimes(1));
  });

  it("disables send button when input is empty", () => {
    useReviewStore.getState().openRequestPanel(1);
    renderPanel();

    const sendBtn = screen.getByTestId("evidence-request-send");
    expect(sendBtn.hasAttribute("disabled")).toBe(true);
  });

  it("shows target context when evidenceId is set", () => {
    useReviewStore.getState().openRequestPanel(1, "ev-5");
    renderPanel();

    expect(screen.getByTestId("evidence-request-target-context")).toBeTruthy();
    expect(screen.getByText("ev-5")).toBeTruthy();
  });
});
