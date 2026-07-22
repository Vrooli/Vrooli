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
  suggestedSkills: [],
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
  describe("Decision header", () => {
    it("renders a clickable title and counter without the redundant back control", () => {
      render(<DecisionStreamView {...defaultProps} />);

      expect(screen.getByTestId(selectors.commandPost.decisionStream.header)).toBeInTheDocument();
      expect(screen.queryByTestId(selectors.commandPost.decisionStream.backButton)).not.toBeInTheDocument();
      expect(screen.getByTestId(selectors.commandPost.decisionStream.counter)).toHaveTextContent("1/1");
      expect(screen.getByTestId(selectors.commandPost.decisionStream.openItemLink)).toHaveTextContent("Dashboard feature");
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

    it("restores the selected decision by its stable question id", () => {
      const questions = [
        makeCrossItemQuestion({ question: makeWorkshopQuestion({ id: "d1", topic: "First" }) }),
        makeCrossItemQuestion({ question: makeWorkshopQuestion({ id: "d2", topic: "Restored" }) }),
      ];
      render(<DecisionStreamView {...defaultProps} questions={questions} currentQuestionId="d2" />);

      expect(screen.getByText("Restored")).toBeInTheDocument();
      expect(screen.getByTestId(selectors.commandPost.decisionStream.counter)).toHaveTextContent("2/2");
    });

    it("falls back to the first decision and repairs a stale question id", () => {
      const onCurrentQuestionChange = vi.fn();
      render(<DecisionStreamView {...defaultProps} currentQuestionId="removed-after-refetch" onCurrentQuestionChange={onCurrentQuestionChange} />);

      expect(screen.getByText("Architecture decision")).toBeInTheDocument();
      expect(onCurrentQuestionChange).toHaveBeenCalledWith("d1");
    });

    it("opens the full backlog item from the title link", () => {
      const onOpenItem = vi.fn();
      render(<DecisionStreamView {...defaultProps} onOpenItem={onOpenItem} />);

      fireEvent.click(screen.getByTestId(selectors.commandPost.decisionStream.openItemLink));
      expect(onOpenItem).toHaveBeenCalledWith("idea", "dashboard");
    });
  });

  describe("Removed context panel", () => {
    it("does not render the duplicate item context surface", () => {
      render(<DecisionStreamView {...defaultProps} />);

      expect(screen.queryByTestId(selectors.commandPost.decisionStream.contextPanel)).not.toBeInTheDocument();
      expect(screen.queryByTestId(selectors.commandPost.decisionStream.contextToggle)).not.toBeInTheDocument();
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

    it("empty state has no redundant back button", () => {
      render(<DecisionStreamView {...defaultProps} questions={[]} />);

      expect(screen.queryByText("Back to Command Post")).not.toBeInTheDocument();
    });
  });

  describe("Keyboard shortcuts", () => {
    it("Escape calls onBack when context panel is collapsed", () => {
      const onBack = vi.fn();
      render(<DecisionStreamView {...defaultProps} onBack={onBack} />);

      fireEvent.keyDown(window, { key: "Escape" });
      expect(onBack).toHaveBeenCalledOnce();
    });

    it("'i' key does not restore the removed context panel", () => {
      render(<DecisionStreamView {...defaultProps} />);

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
