import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { ScenarioNavigatorPopover } from "./ScenarioNavigatorPopover";
import { selectors } from "../../consts/selectors";
import type { CrossItemQuestion } from "../../lib/command-post-utils";
import type { BacklogKind, PendingQuestion, DecisionOption } from "../../types";
import type { QuestionAnswer } from "../backlog/question-renderers";

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

const makeOptions = (): DecisionOption[] => [
  { key: "A", label: "Option Alpha", rationale: "First approach" },
  { key: "B", label: "Option Beta", rationale: "Second approach", recommended: true },
];

const makeWorkshopQuestion = (overrides?: Partial<PendingQuestion>): PendingQuestion => ({
  id: "q1",
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

function buildParentGroups(...groups: Array<{ kind: BacklogKind; name: string; title: string; questions: Array<Partial<PendingQuestion>> }>): Map<string, CrossItemQuestion[]> {
  const map = new Map<string, CrossItemQuestion[]>();
  for (const g of groups) {
    const key = `${g.kind}/${g.name}`;
    map.set(
      key,
      g.questions.map((qOverrides, i) =>
        makeCrossItemQuestion({
          parentKind: g.kind,
          parentName: g.name,
          parentTitle: g.title,
          question: makeWorkshopQuestion({ id: `${g.name}-q${i}`, ...qOverrides }),
        }),
      ),
    );
  }
  return map;
}

beforeEach(() => {
  vi.clearAllMocks();
});

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe("ScenarioNavigatorPopover", () => {
  const onClose = vi.fn();
  const onJumpTo = vi.fn();
  const onSnoozeParent = vi.fn();

  const twoGroupMap = buildParentGroups(
    { kind: "idea", name: "dashboard", title: "Dashboard feature", questions: [{ id: "dash-q0" }, { id: "dash-q1" }] },
    { kind: "fix", name: "login-bug", title: "Fix login bug", questions: [{ id: "login-q0" }] },
  );

  const defaultProps = {
    isOpen: true,
    onClose,
    parentGroups: twoGroupMap,
    currentParentKey: "idea/dashboard",
    localAnswers: new Map<string, QuestionAnswer>(),
    skippedIds: new Set<string>(),
    onJumpTo,
    onSnoozeParent,
  };

  it("renders nothing when closed", () => {
    const { container } = render(<ScenarioNavigatorPopover {...defaultProps} isOpen={false} />);
    expect(container.innerHTML).toBe("");
  });

  it("renders parent items with correct counts", () => {
    render(<ScenarioNavigatorPopover {...defaultProps} />);

    const popover = screen.getByTestId(selectors.commandPost.decisionStream.navigatorPopover);
    expect(popover).toBeInTheDocument();

    const rows = screen.getAllByTestId(selectors.commandPost.decisionStream.navigatorRow);
    expect(rows).toHaveLength(2);

    // First row: Dashboard feature with 0/2 answered
    expect(rows[0]).toHaveTextContent("Dashboard feature");
    expect(rows[0]).toHaveTextContent("0/2");

    // Second row: Fix login bug with 0/1 answered
    expect(rows[1]).toHaveTextContent("Fix login bug");
    expect(rows[1]).toHaveTextContent("0/1");
  });

  it("shows correct answered count when answers exist", () => {
    const answers = new Map<string, QuestionAnswer>();
    answers.set("dash-q0", { selected: "A" });

    render(
      <ScenarioNavigatorPopover
        {...defaultProps}
        localAnswers={answers}
      />,
    );

    const rows = screen.getAllByTestId(selectors.commandPost.decisionStream.navigatorRow);
    expect(rows[0]).toHaveTextContent("1/2");
  });

  it("highlights current parent", () => {
    render(<ScenarioNavigatorPopover {...defaultProps} />);

    const rows = screen.getAllByTestId(selectors.commandPost.decisionStream.navigatorRow);
    // Current parent should have cyan accent
    expect(rows[0]?.className).toContain("bg-cyan-500/10");
    // Other parents should not
    expect(rows[1]?.className).not.toContain("bg-cyan-500/10");
  });

  it("click row calls onJumpTo with correct key and closes", () => {
    render(<ScenarioNavigatorPopover {...defaultProps} />);

    const rows = screen.getAllByTestId(selectors.commandPost.decisionStream.navigatorRow);
    const secondRow = rows[1];
    expect(secondRow).toBeDefined();
    if (!secondRow) {
      throw new Error("expected second navigator row");
    }
    fireEvent.click(secondRow);

    expect(onJumpTo).toHaveBeenCalledWith("fix/login-bug");
    expect(onClose).toHaveBeenCalledOnce();
  });

  it("snooze button calls onSnoozeParent", () => {
    render(<ScenarioNavigatorPopover {...defaultProps} />);

    const snoozeButtons = screen.getAllByTestId(selectors.commandPost.decisionStream.navigatorSnooze);
    const secondSnoozeButton = snoozeButtons[1];
    expect(secondSnoozeButton).toBeDefined();
    if (!secondSnoozeButton) {
      throw new Error("expected second snooze button");
    }
    fireEvent.click(secondSnoozeButton);

    expect(onSnoozeParent).toHaveBeenCalledWith("fix", "login-bug");
    // Should not trigger row click / onJumpTo
    expect(onJumpTo).not.toHaveBeenCalled();
  });
});
