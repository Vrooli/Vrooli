import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { InlineQuestionStepper } from "./inline-question-stepper";
import type { PendingQuestion, DecisionOption } from "../../types/domain";

// Mock the backlog service
vi.mock("../../services/backlog-service", () => ({
  backlogService: {
    getFileContent: vi.fn().mockResolvedValue(JSON.stringify({
      round: 1,
      generated_at: "2026-03-20T00:00:00Z",
      readiness: { problem_clarity: 2, scope_defined: 2, approach_solid: 2, testable: 2, risk_awareness: 2 },
      items: [
        { id: "d1", type: "decision", topic: "Scope", options: [{ key: "A", label: "Small" }, { key: "B", label: "Large" }], selected: null },
        { id: "d2", type: "decision", topic: "Stack", options: [{ key: "A", label: "Go" }], selected: null },
      ],
    })),
    saveFileContent: vi.fn().mockResolvedValue({}),
    batchReview: vi.fn().mockResolvedValue(undefined),
    workshopSave: vi.fn().mockResolvedValue({
      file: { name: "round-001.json", path: "workshop/round-001.json", type: "file", size: 100 },
      autoAdvance: { triggered: false, reason: "not_ready" },
    }),
  },
}));

const makeOptions = (): DecisionOption[] => [
  { key: "A", label: "Option A", rationale: "First option" },
  { key: "B", label: "Option B", rationale: "Second option" },
];

const makeWorkshopQuestion = (overrides?: Partial<PendingQuestion>): PendingQuestion => ({
  id: "d1",
  source: "workshop",
  item_kind: "idea",
  item_name: "test-item",
  topic: "Architecture",
  text: "Choose approach",
  options: makeOptions(),
  selected: null,
  round_number: 1,
  ...overrides,
});

const makeReviewQuestion = (overrides?: Partial<PendingQuestion>): PendingQuestion => ({
  id: "OT-P0-001",
  source: "review",
  item_kind: "idea",
  item_name: "test-item",
  title: "Core functionality",
  description: "Must support X",
  criticality: "P0",
  review_status: "unreviewed",
  review_type: "target",
  ...overrides,
});

const defaultProps = {
  backlogKind: "idea" as const,
  backlogName: "test-item",
  onAllAnswered: vi.fn(),
};

beforeEach(() => {
  vi.clearAllMocks();
});

describe("InlineQuestionStepper", () => {
  it("renders the first question by default", () => {
    const q1 = makeWorkshopQuestion({ id: "d1", topic: "First Question" });
    const q2 = makeWorkshopQuestion({ id: "d2", topic: "Second Question" });
    render(<InlineQuestionStepper {...defaultProps} questions={[q1, q2]} />);

    expect(screen.getByText("First Question")).toBeInTheDocument();
    expect(screen.getByText("1 of 2")).toBeInTheDocument();
  });

  it("shows progress indicator with correct count", () => {
    const questions = [
      makeWorkshopQuestion({ id: "d1" }),
      makeWorkshopQuestion({ id: "d2" }),
      makeReviewQuestion({ id: "r1" }),
    ];
    render(<InlineQuestionStepper {...defaultProps} questions={questions} />);

    expect(screen.getByText("1 of 3")).toBeInTheDocument();
  });

  it("back arrow is disabled on first question", () => {
    render(<InlineQuestionStepper {...defaultProps} questions={[makeWorkshopQuestion()]} />);

    const backBtn = screen.getByTestId("question-stepper-prev");
    expect(backBtn).toBeDisabled();
  });

  it("forward arrow advances to next question", async () => {
    const q1 = makeWorkshopQuestion({ id: "d1", topic: "First" });
    const q2 = makeWorkshopQuestion({ id: "d2", topic: "Second" });
    render(<InlineQuestionStepper {...defaultProps} questions={[q1, q2]} />);

    fireEvent.click(screen.getByTestId("question-stepper-next"));

    await waitFor(() => {
      expect(screen.getByText("Second")).toBeInTheDocument();
      expect(screen.getByText("2 of 2")).toBeInTheDocument();
    });
  });

  it("back arrow navigates to previous question", async () => {
    const q1 = makeWorkshopQuestion({ id: "d1", topic: "First" });
    const q2 = makeWorkshopQuestion({ id: "d2", topic: "Second" });
    render(<InlineQuestionStepper {...defaultProps} questions={[q1, q2]} />);

    // Advance to second
    fireEvent.click(screen.getByTestId("question-stepper-next"));
    await waitFor(() => expect(screen.getByText("Second")).toBeInTheDocument());

    // Go back
    fireEvent.click(screen.getByTestId("question-stepper-prev"));
    expect(screen.getByText("First")).toBeInTheDocument();
  });

  it("workshop: clicking an option selects it", () => {
    render(<InlineQuestionStepper {...defaultProps} questions={[makeWorkshopQuestion()]} />);

    const options = screen.getAllByTestId("question-stepper-workshop-option");
    fireEvent.click(options[0]!); // Click option A

    // Option A button should have the selected styling (emerald border)
    expect(options[0]!.className).toContain("emerald");
  });

  it("review: clicking Approve sets status", () => {
    render(<InlineQuestionStepper {...defaultProps} questions={[makeReviewQuestion()]} />);

    const approveBtn = screen.getByTestId("question-stepper-review-approve");
    fireEvent.click(approveBtn);

    expect(approveBtn.className).toContain("emerald");
  });

  it("review: clicking Flag shows comment field", () => {
    render(<InlineQuestionStepper {...defaultProps} questions={[makeReviewQuestion()]} />);

    const flagBtn = screen.getByTestId("question-stepper-review-flag");
    fireEvent.click(flagBtn);

    expect(screen.getByPlaceholderText("Comment (optional)...")).toBeInTheDocument();
  });

  it("skip advances without requiring an answer", async () => {
    const q1 = makeWorkshopQuestion({ id: "d1", topic: "First" });
    const q2 = makeWorkshopQuestion({ id: "d2", topic: "Second" });
    render(<InlineQuestionStepper {...defaultProps} questions={[q1, q2]} />);

    fireEvent.click(screen.getByTestId("question-stepper-skip"));

    expect(screen.getByText("Second")).toBeInTheDocument();
  });

  it("calls onAllAnswered when all questions are skipped", async () => {
    const onAllAnswered = vi.fn();
    render(
      <InlineQuestionStepper
        {...defaultProps}
        onAllAnswered={onAllAnswered}
        questions={[makeWorkshopQuestion()]}
      />,
    );

    fireEvent.click(screen.getByTestId("question-stepper-skip"));

    await waitFor(() => expect(onAllAnswered).toHaveBeenCalledTimes(1));
  });

  it("calls onAllAnswered when all questions are answered and advanced", async () => {
    const onAllAnswered = vi.fn();
    const q1 = makeWorkshopQuestion({ id: "d1" });
    render(
      <InlineQuestionStepper
        {...defaultProps}
        onAllAnswered={onAllAnswered}
        questions={[q1]}
      />,
    );

    // Select option A
    const options = screen.getAllByTestId("question-stepper-workshop-option");
    fireEvent.click(options[0]!);

    // Click Done (last question)
    fireEvent.click(screen.getByTestId("question-stepper-next"));

    await waitFor(() => {
      expect(onAllAnswered).toHaveBeenCalledTimes(1);
    });
  });

  it("renders mixed workshop and review questions in sequence", async () => {
    const q1 = makeWorkshopQuestion({ id: "d1", topic: "Workshop Q" });
    const q2 = makeReviewQuestion({ id: "r1", title: "Review Q" });
    render(<InlineQuestionStepper {...defaultProps} questions={[q1, q2]} />);

    // First question is workshop
    expect(screen.getByText("Workshop Q")).toBeInTheDocument();

    // Advance
    fireEvent.click(screen.getByTestId("question-stepper-next"));
    await waitFor(() => {
      expect(screen.getByText("Review Q")).toBeInTheDocument();
    });
  });

  it("prevents card link navigation via stopPropagation", () => {
    render(<InlineQuestionStepper {...defaultProps} questions={[makeWorkshopQuestion()]} />);

    const container = screen.getByTestId("question-stepper");
    const clickEvent = new MouseEvent("click", { bubbles: true, cancelable: true });
    const preventDefaultSpy = vi.spyOn(clickEvent, "preventDefault");
    container.dispatchEvent(clickEvent);

    expect(preventDefaultSpy).toHaveBeenCalled();
  });
});
