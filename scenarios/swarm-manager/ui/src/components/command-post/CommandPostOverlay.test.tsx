import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { CommandPostOverlay } from "./CommandPostOverlay";
import { selectors } from "../../consts/selectors";
import type { BacklogItem } from "../../types";

// ---------------------------------------------------------------------------
// Mocks
// ---------------------------------------------------------------------------

vi.mock("@tanstack/react-query", () => ({
  useQuery: vi.fn().mockReturnValue({
    data: { pending_questions: { items: [] } },
    isLoading: false,
  }),
}));

vi.mock("../../services", () => ({
  backlogService: {
    getBacklogSummary: vi.fn().mockResolvedValue({ pending_questions: { items: [] } }),
  },
}));

vi.mock("../../stores/snooze-store", () => ({
  useSnoozeStore: vi.fn((selector: (state: { snooze: () => void }) => unknown) =>
    selector({ snooze: vi.fn() }),
  ),
  useSnoozedKeys: vi.fn().mockReturnValue(new Set()),
}));

// SummaryView renders a simple stub — we're testing overlay chrome, not summary content
vi.mock("./SummaryView", () => ({
  SummaryView: ({ onEnterDecisionStream }: { onEnterDecisionStream: () => void }) => (
    <div data-testid="mock-summary">
      <button type="button" onClick={onEnterDecisionStream} data-testid="enter-decision-stream">
        Enter Decision Stream
      </button>
    </div>
  ),
}));

vi.mock("./DecisionStreamView", () => ({
  DecisionStreamView: ({ onBack }: { onBack: () => void }) => (
    <div data-testid="mock-decision-stream">
      <button type="button" onClick={onBack} data-testid="ds-go-back">
        Back
      </button>
    </div>
  ),
}));

vi.mock("../backlog/clarification-panel", () => ({
  ClarificationPanel: () => <div data-testid="mock-clarification-panel" />,
}));

vi.mock("../../stores/backlog-store", () => ({
  useBacklogStore: vi.fn((selector: (state: { items: BacklogItem[] }) => unknown) =>
    selector({ items: [] }),
  ),
  buildActiveBacklogKeys: vi.fn(() => new Set<string>()),
}));

const defaultProps = {
  isOpen: true,
  onClose: vi.fn(),
  onNavigateToDetail: vi.fn(),
  onSwitchLens: vi.fn(),
};

beforeEach(() => {
  vi.clearAllMocks();
});

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe("CommandPostOverlay", () => {
  it("renders nothing when isOpen is false", () => {
    render(<CommandPostOverlay {...defaultProps} isOpen={false} />);

    expect(screen.queryByTestId(selectors.commandPost.overlay)).not.toBeInTheDocument();
  });

  it("renders overlay with header in summary mode", () => {
    render(<CommandPostOverlay {...defaultProps} />);

    expect(screen.getByTestId(selectors.commandPost.overlay)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.commandPost.overlayHeader)).toBeInTheDocument();
    expect(screen.getByText("Command Post")).toBeInTheDocument();
  });

  it("hides overlay header when entering decision-stream mode", () => {
    render(<CommandPostOverlay {...defaultProps} />);

    // Enter decision stream
    fireEvent.click(screen.getByTestId("enter-decision-stream"));

    // Header should be gone
    expect(screen.queryByTestId(selectors.commandPost.overlayHeader)).not.toBeInTheDocument();
    // Decision stream should be visible
    expect(screen.getByTestId("mock-decision-stream")).toBeInTheDocument();
  });

  it("decision-stream container has h-full class for maximum space", () => {
    render(<CommandPostOverlay {...defaultProps} />);

    fireEvent.click(screen.getByTestId("enter-decision-stream"));

    // The decision-stream wrapper should have h-full
    const dsContainer = screen.getByTestId("mock-decision-stream").parentElement;
    expect(dsContainer?.className).toContain("h-full");
  });

  it("returns to summary when decision-stream back is clicked", () => {
    render(<CommandPostOverlay {...defaultProps} />);

    // Enter decision stream
    fireEvent.click(screen.getByTestId("enter-decision-stream"));
    expect(screen.getByTestId("mock-decision-stream")).toBeInTheDocument();

    // Go back
    fireEvent.click(screen.getByTestId("ds-go-back"));
    expect(screen.getByTestId("mock-summary")).toBeInTheDocument();
    expect(screen.getByTestId(selectors.commandPost.overlayHeader)).toBeInTheDocument();
  });

  it("close button calls onClose", () => {
    const onClose = vi.fn();
    render(<CommandPostOverlay {...defaultProps} onClose={onClose} />);

    fireEvent.click(screen.getByTestId(selectors.commandPost.close));
    expect(onClose).toHaveBeenCalledOnce();
  });

  it("resets to summary view when reopened", () => {
    const { rerender } = render(<CommandPostOverlay {...defaultProps} />);

    // Enter decision stream
    fireEvent.click(screen.getByTestId("enter-decision-stream"));
    expect(screen.getByTestId("mock-decision-stream")).toBeInTheDocument();

    // Close and reopen
    rerender(<CommandPostOverlay {...defaultProps} isOpen={false} />);
    rerender(<CommandPostOverlay {...defaultProps} isOpen={true} />);

    // Should be back at summary
    expect(screen.getByTestId("mock-summary")).toBeInTheDocument();
  });
});
