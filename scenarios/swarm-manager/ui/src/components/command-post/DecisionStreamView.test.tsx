import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { DecisionStreamView } from "./DecisionStreamView";
import { selectors } from "../../consts/selectors";
import type { CrossItemQuestion } from "../../lib/command-post-utils";
import type { BacklogItem, BacklogKind, DecisionOption, PendingQuestion } from "../../types";

// ---------------------------------------------------------------------------
// Mocks
// ---------------------------------------------------------------------------

vi.mock("../../services/backlog-service", () => ({
  backlogService: {
    get: vi.fn().mockRejectedValue(new Error("not found")),
    getFileContent: vi.fn().mockResolvedValue("{}"),
    saveFileContent: vi.fn().mockResolvedValue({}),
    batchReview: vi.fn().mockResolvedValue(undefined),
    workshopSave: vi.fn().mockResolvedValue({
      file: { name: "round-001.json", path: "workshop/round-001.json", type: "file", size: 100 },
      autoAdvance: { triggered: false, reason: "not_ready" },
    }),
  },
}));

const mockBacklogItem: BacklogItem = {
  name: "dashboard",
  title: "Dashboard feature",
  description: "Build the dashboard page with widgets and charts.",
  kind: "idea",
  status: "ready",
  priority: 2,
  tags: ["ui", "frontend"],
  initiative: "v2-launch",
  effort: "M",
  created: "2026-04-01T00:00:00Z",
  updated: "2026-04-01T00:00:00Z",
};

vi.mock("../../stores/backlog-store", () => ({
  useBacklogStore: vi.fn((selector: (state: { items: BacklogItem[] }) => unknown) =>
    selector({ items: [mockBacklogItem] }),
  ),
}));

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

const makeOptions = (): DecisionOption[] => [
  { key: "A", label: "Option Alpha", rationale: "First approach" },
  { key: "B", label: "Option Beta", rationale: "Second approach", recommended: true },
];

const makeWorkshopQuestion = (overrides?: Partial<PendingQuestion>): PendingQuestion => ({
  id: "d1",
  source: "workshop",
  item_kind: "idea",
  item_name: "dashboard",
  topic: "Architecture decision",
  text: "Choose approach",
  options: makeOptions(),
  selected: null,
  round_number: 1,
  ...overrides,
});

const makeCrossItemQuestion = (overrides?: Partial<CrossItemQuestion>): CrossItemQuestion => ({
  question: makeWorkshopQuestion(),
  parentKind: "idea" as BacklogKind,
  parentName: "dashboard",
  parentTitle: "Dashboard feature",
  ...overrides,
});

const defaultProps = {
  questions: [makeCrossItemQuestion()],
  onComplete: vi.fn(),
  onBack: vi.fn(),
  onSnoozeItem: vi.fn(),
};

beforeEach(() => {
  vi.clearAllMocks();
});

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe("DecisionStreamView", () => {
  describe("Unified header", () => {
    it("renders the header with back button, title, and counter", () => {
      render(<DecisionStreamView {...defaultProps} />);

      expect(screen.getByTestId(selectors.commandPost.decisionStream.header)).toBeInTheDocument();
      expect(screen.getByTestId(selectors.commandPost.decisionStream.backButton)).toBeInTheDocument();
      expect(screen.getByTestId(selectors.commandPost.decisionStream.counter)).toHaveTextContent("1/1");
      expect(screen.getByText("Dashboard feature")).toBeInTheDocument();
    });

    it("shows correct counter with multiple questions", () => {
      const questions = [
        makeCrossItemQuestion({ question: makeWorkshopQuestion({ id: "d1" }) }),
        makeCrossItemQuestion({ question: makeWorkshopQuestion({ id: "d2", topic: "Second" }) }),
        makeCrossItemQuestion({ question: makeWorkshopQuestion({ id: "d3", topic: "Third" }) }),
      ];
      render(<DecisionStreamView {...defaultProps} questions={questions} />);

      expect(screen.getByTestId(selectors.commandPost.decisionStream.counter)).toHaveTextContent("1/3");
    });

    it("calls onBack when back button is clicked", () => {
      const onBack = vi.fn();
      render(<DecisionStreamView {...defaultProps} onBack={onBack} />);

      fireEvent.click(screen.getByTestId(selectors.commandPost.decisionStream.backButton));
      expect(onBack).toHaveBeenCalledOnce();
    });
  });

  describe("Context panel", () => {
    it("is collapsed by default", () => {
      render(<DecisionStreamView {...defaultProps} />);

      expect(screen.queryByTestId(selectors.commandPost.decisionStream.contextPanel)).not.toBeInTheDocument();
    });

    it("expands when chevron toggle is clicked", () => {
      render(<DecisionStreamView {...defaultProps} />);

      fireEvent.click(screen.getByTestId(selectors.commandPost.decisionStream.contextToggle));
      expect(screen.getByTestId(selectors.commandPost.decisionStream.contextPanel)).toBeInTheDocument();
    });

    it("shows backlog item details when expanded", () => {
      render(<DecisionStreamView {...defaultProps} />);

      fireEvent.click(screen.getByTestId(selectors.commandPost.decisionStream.contextToggle));

      // Description
      expect(screen.getByText(/Build the dashboard page/)).toBeInTheDocument();
      // Initiative
      expect(screen.getByText("v2-launch")).toBeInTheDocument();
      // Effort
      expect(screen.getByText("M")).toBeInTheDocument();
      // Slug
      expect(screen.getByText("idea/dashboard")).toBeInTheDocument();
    });

    it("shows slug even when context panel has no store data", () => {
      render(<DecisionStreamView {...defaultProps} />);

      fireEvent.click(screen.getByTestId(selectors.commandPost.decisionStream.contextToggle));
      // Slug is always shown regardless of parentItem availability
      expect(screen.getByText("idea/dashboard")).toBeInTheDocument();
    });

    it("collapses when toggle is clicked again", () => {
      render(<DecisionStreamView {...defaultProps} />);

      const toggle = screen.getByTestId(selectors.commandPost.decisionStream.contextToggle);
      fireEvent.click(toggle);
      expect(screen.getByTestId(selectors.commandPost.decisionStream.contextPanel)).toBeInTheDocument();

      fireEvent.click(toggle);
      expect(screen.queryByTestId(selectors.commandPost.decisionStream.contextPanel)).not.toBeInTheDocument();
    });

    it("auto-collapses when navigating to a different parent item", () => {
      const q1 = makeCrossItemQuestion({
        question: makeWorkshopQuestion({ id: "d1" }),
        parentName: "dashboard",
        parentTitle: "Dashboard feature",
      });
      const q2 = makeCrossItemQuestion({
        question: makeWorkshopQuestion({ id: "d2", topic: "Other topic" }),
        parentKind: "fix" as BacklogKind,
        parentName: "other-item",
        parentTitle: "Other item",
      });
      render(<DecisionStreamView {...defaultProps} questions={[q1, q2]} />);

      // Expand context panel
      fireEvent.click(screen.getByTestId(selectors.commandPost.decisionStream.contextToggle));
      expect(screen.getByTestId(selectors.commandPost.decisionStream.contextPanel)).toBeInTheDocument();

      // Navigate to next (different parent)
      fireEvent.click(screen.getByTestId(selectors.commandPost.decisionStream.navNext));
      expect(screen.queryByTestId(selectors.commandPost.decisionStream.contextPanel)).not.toBeInTheDocument();
    });
  });

  describe("Navigation buttons", () => {
    it("all nav buttons have min-h-[44px] for touch targets", () => {
      const questions = [
        makeCrossItemQuestion({ question: makeWorkshopQuestion({ id: "d1" }) }),
        makeCrossItemQuestion({ question: makeWorkshopQuestion({ id: "d2", topic: "Second" }) }),
      ];
      render(<DecisionStreamView {...defaultProps} questions={questions} />);

      const back = screen.getByTestId(selectors.commandPost.decisionStream.navBack);
      const skip = screen.getByTestId(selectors.commandPost.decisionStream.navSkip);
      const snooze = screen.getByTestId(selectors.commandPost.decisionStream.navSnooze);
      const next = screen.getByTestId(selectors.commandPost.decisionStream.navNext);

      for (const btn of [back, skip, snooze, next]) {
        expect(btn.className).toContain("min-h-[44px]");
      }
    });

    it("back button is disabled on the first question", () => {
      render(<DecisionStreamView {...defaultProps} />);

      const back = screen.getByTestId(selectors.commandPost.decisionStream.navBack);
      expect(back).toBeDisabled();
      expect(back.className).toContain("opacity-30");
    });

    it("shows 'Done' instead of 'Next' on the last question", () => {
      render(<DecisionStreamView {...defaultProps} />);

      expect(screen.getByTestId(selectors.commandPost.decisionStream.navNext)).toHaveTextContent("Done");
    });

    it("shows 'Next' when there are more questions", () => {
      const questions = [
        makeCrossItemQuestion({ question: makeWorkshopQuestion({ id: "d1" }) }),
        makeCrossItemQuestion({ question: makeWorkshopQuestion({ id: "d2", topic: "Second" }) }),
      ];
      render(<DecisionStreamView {...defaultProps} questions={questions} />);

      expect(screen.getByTestId(selectors.commandPost.decisionStream.navNext)).toHaveTextContent("Next");
    });

    it("skip advances to the next question", () => {
      const questions = [
        makeCrossItemQuestion({ question: makeWorkshopQuestion({ id: "d1", topic: "First topic" }) }),
        makeCrossItemQuestion({ question: makeWorkshopQuestion({ id: "d2", topic: "Second topic" }) }),
      ];
      render(<DecisionStreamView {...defaultProps} questions={questions} />);

      expect(screen.getByText("First topic")).toBeInTheDocument();

      fireEvent.click(screen.getByTestId(selectors.commandPost.decisionStream.navSkip));

      expect(screen.getByText("Second topic")).toBeInTheDocument();
      expect(screen.getByTestId(selectors.commandPost.decisionStream.counter)).toHaveTextContent("2/2");
    });

    it("snooze calls onSnoozeItem with the correct key", () => {
      const onSnoozeItem = vi.fn();
      render(<DecisionStreamView {...defaultProps} onSnoozeItem={onSnoozeItem} />);

      fireEvent.click(screen.getByTestId(selectors.commandPost.decisionStream.navSnooze));
      expect(onSnoozeItem).toHaveBeenCalledWith("backlog:idea/dashboard");
    });
  });

  describe("Progress bar", () => {
    it("renders with correct width percentage", () => {
      const questions = [
        makeCrossItemQuestion({ question: makeWorkshopQuestion({ id: "d1" }) }),
        makeCrossItemQuestion({ question: makeWorkshopQuestion({ id: "d2" }) }),
        makeCrossItemQuestion({ question: makeWorkshopQuestion({ id: "d3" }) }),
        makeCrossItemQuestion({ question: makeWorkshopQuestion({ id: "d4" }) }),
      ];
      render(<DecisionStreamView {...defaultProps} questions={questions} />);

      const progressBar = screen.getByTestId(selectors.commandPost.decisionStream.progressBar);
      const fill = progressBar.firstChild as HTMLElement;
      // 1/4 = 25%
      expect(fill.style.width).toBe("25%");
    });
  });

  describe("Empty state", () => {
    it("renders 'No pending questions' when questions list is empty", () => {
      render(<DecisionStreamView {...defaultProps} questions={[]} />);

      expect(screen.getByText("No pending questions")).toBeInTheDocument();
    });

    it("empty state back button has proper touch target", () => {
      render(<DecisionStreamView {...defaultProps} questions={[]} />);

      const backBtn = screen.getByText("Back to Command Post").closest("button");
      expect(backBtn?.className).toContain("min-h-[44px]");
    });
  });

  describe("Keyboard shortcuts", () => {
    it("Escape calls onBack when context panel is collapsed", () => {
      const onBack = vi.fn();
      render(<DecisionStreamView {...defaultProps} onBack={onBack} />);

      fireEvent.keyDown(window, { key: "Escape" });
      expect(onBack).toHaveBeenCalledOnce();
    });

    it("Escape collapses context panel instead of calling onBack when panel is open", () => {
      const onBack = vi.fn();
      render(<DecisionStreamView {...defaultProps} onBack={onBack} />);

      // Open context panel
      fireEvent.click(screen.getByTestId(selectors.commandPost.decisionStream.contextToggle));
      expect(screen.getByTestId(selectors.commandPost.decisionStream.contextPanel)).toBeInTheDocument();

      // Escape should close panel, not go back
      fireEvent.keyDown(window, { key: "Escape" });
      expect(screen.queryByTestId(selectors.commandPost.decisionStream.contextPanel)).not.toBeInTheDocument();
      expect(onBack).not.toHaveBeenCalled();
    });

    it("'i' key toggles context panel", () => {
      render(<DecisionStreamView {...defaultProps} />);

      fireEvent.keyDown(window, { key: "i" });
      expect(screen.getByTestId(selectors.commandPost.decisionStream.contextPanel)).toBeInTheDocument();

      fireEvent.keyDown(window, { key: "i" });
      expect(screen.queryByTestId(selectors.commandPost.decisionStream.contextPanel)).not.toBeInTheDocument();
    });
  });

  describe("Question content", () => {
    it("renders workshop question with options", () => {
      render(<DecisionStreamView {...defaultProps} />);

      expect(screen.getByTestId(selectors.commandPost.decisionStream.questionArea)).toBeInTheDocument();
      expect(screen.getByText("Option Alpha")).toBeInTheDocument();
      expect(screen.getByText("Option Beta")).toBeInTheDocument();
    });
  });
});
