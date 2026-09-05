import { renderWithProviders as render } from "../test-utils";
import { describe, it, expect, vi } from "vitest";
import { screen, fireEvent } from "@testing-library/react";

import HandoffSuggestionChip from "../components/handoff/HandoffSuggestionChip";
import type { HandoffSuggestion } from "../lib/captureRules";

// [REQ:P0-014h] Handoff Capture Rules

const suggestion: HandoffSuggestion = {
  ruleId: "r1",
  ruleName: "Plan file",
  eventId: "e1",
  payload: "/home/me/.vrooli/plans/a-plan.md",
};

describe("HandoffSuggestionChip", () => {
  // A wrong suggestion has to be traceable to the rule that caused it, or the
  // operator has no way to fix it.
  it("names the rule that fired and the payload it found", () => {
    render(<HandoffSuggestionChip suggestion={suggestion} onOpen={vi.fn()} onDismiss={vi.fn()} />);
    const chip = screen.getByTestId("handoff-suggestion");
    expect(chip).toHaveAttribute("data-rule-id", "r1");
    expect(chip).toHaveTextContent("/home/me/.vrooli/plans/a-plan.md");
  });

  // Pressing the chip opens the composer; it does not send.
  it("opens rather than sending", () => {
    const onOpen = vi.fn();
    render(<HandoffSuggestionChip suggestion={suggestion} onOpen={onOpen} onDismiss={vi.fn()} />);
    fireEvent.click(screen.getByText("handoff.suggestionOpen"));
    expect(onOpen).toHaveBeenCalledWith(suggestion);
  });

  it("can be dismissed", () => {
    const onDismiss = vi.fn();
    render(<HandoffSuggestionChip suggestion={suggestion} onOpen={vi.fn()} onDismiss={onDismiss} />);
    fireEvent.click(screen.getByTestId("handoff-suggestion-dismiss"));
    expect(onDismiss).toHaveBeenCalledWith(suggestion);
  });

  // It is inline, not a modal: nothing about it takes over the screen.
  it("renders inline, with no dialog role", () => {
    render(<HandoffSuggestionChip suggestion={suggestion} onOpen={vi.fn()} onDismiss={vi.fn()} />);
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  });
});
