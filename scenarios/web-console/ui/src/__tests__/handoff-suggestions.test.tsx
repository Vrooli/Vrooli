import { renderWithProviders as render } from "../test-utils";
import { describe, it, expect, vi } from "vitest";
import { screen, fireEvent, within } from "@testing-library/react";

import HandoffSuggestionChip from "../components/handoff/HandoffSuggestionChip";
import type { HandoffSuggestion } from "../lib/captureRules";

// [REQ:P0-014h] Handoff Capture Rules

const suggestion: HandoffSuggestion = {
  ruleId: "r1",
  ruleName: "Plan file",
  eventId: "e1",
  payload: "/home/me/.vrooli/plans/a-plan.md",
};

const three: HandoffSuggestion[] = [
  suggestion,
  { ...suggestion, payload: "/home/me/.vrooli/plans/b-plan.md" },
  { ...suggestion, payload: "/home/me/.vrooli/plans/c-plan.md" },
];

describe("HandoffSuggestionChip", () => {
  // A wrong suggestion has to be traceable to the rule that caused it, or the
  // operator has no way to fix it.
  it("names the rule that fired and the payload it found", () => {
    render(<HandoffSuggestionChip suggestions={[suggestion]} onOpen={vi.fn()} onDismiss={vi.fn()} />);
    const chip = screen.getByTestId("handoff-suggestion");
    expect(chip).toHaveAttribute("data-rule-id", "r1");
    expect(chip).toHaveTextContent("/home/me/.vrooli/plans/a-plan.md");
  });

  // Pressing the chip opens the composer; it does not send.
  it("opens rather than sending", () => {
    const onOpen = vi.fn();
    render(<HandoffSuggestionChip suggestions={[suggestion]} onOpen={onOpen} onDismiss={vi.fn()} />);
    fireEvent.click(screen.getByText("handoff.suggestionOpen"));
    expect(onOpen).toHaveBeenCalledWith(suggestion.payload);
  });

  it("can be dismissed", () => {
    const onDismiss = vi.fn();
    render(<HandoffSuggestionChip suggestions={[suggestion]} onOpen={vi.fn()} onDismiss={onDismiss} />);
    fireEvent.click(screen.getByTestId("handoff-suggestion-dismiss"));
    expect(onDismiss).toHaveBeenCalledWith([suggestion]);
  });

  // It is inline, not a modal: nothing about it takes over the screen.
  it("renders inline, with no dialog role", () => {
    render(<HandoffSuggestionChip suggestions={[suggestion]} onOpen={vi.fn()} onDismiss={vi.fn()} />);
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  });

  // Several matches of one rule in one message are one offer, not a stack of
  // identical chips that pushes the transcript down.
  it("folds several matches of one rule into one chip that counts them, hands them off together, and dismisses them together", () => {
    const onOpen = vi.fn();
    const onDismiss = vi.fn();
    render(<HandoffSuggestionChip suggestions={three} onOpen={onOpen} onDismiss={onDismiss} />);
    expect(screen.getAllByTestId("handoff-suggestion")).toHaveLength(1);
    const chip = screen.getByTestId("handoff-suggestion");
    expect(chip).toHaveAttribute("data-count", "3");
    expect(chip).toHaveTextContent("a-plan.md");
    expect(chip).toHaveTextContent("c-plan.md");
    expect(screen.queryAllByTestId("handoff-suggestion-item")).toHaveLength(0);

    fireEvent.click(screen.getByText("handoff.suggestionOpen"));
    expect(onOpen).toHaveBeenCalledWith(three.map((match) => match.payload).join("\n"));
    fireEvent.click(screen.getByTestId("handoff-suggestion-dismiss"));
    expect(onDismiss).toHaveBeenCalledWith(three);
  });

  it("lists each folded match on request, so one can be handed off or dismissed alone", () => {
    const onOpen = vi.fn();
    const onDismiss = vi.fn();
    render(<HandoffSuggestionChip suggestions={three} onOpen={onOpen} onDismiss={onDismiss} />);
    const toggle = screen.getByTestId("handoff-suggestion-expand");
    expect(toggle).toHaveAttribute("aria-expanded", "false");
    fireEvent.click(toggle);
    expect(toggle).toHaveAttribute("aria-expanded", "true");

    const items = screen.getAllByTestId("handoff-suggestion-item");
    expect(items).toHaveLength(3);
    fireEvent.click(within(items[1] as HTMLElement).getByTestId("handoff-suggestion-item-open"));
    expect(onOpen).toHaveBeenCalledWith("/home/me/.vrooli/plans/b-plan.md");
    fireEvent.click(within(items[2] as HTMLElement).getByTestId("handoff-suggestion-item-dismiss"));
    expect(onDismiss).toHaveBeenCalledWith([three[2]]);
  });
});
