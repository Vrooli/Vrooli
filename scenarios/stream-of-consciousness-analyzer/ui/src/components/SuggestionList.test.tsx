import { render, screen, fireEvent } from "@testing-library/react";
import { describe, it, expect, vi } from "vitest";
import "@testing-library/jest-dom/vitest";
import { SuggestionList } from "./SuggestionList";

const baseSuggestion = {
  source_id: "t1",
  target_id: "t2",
  label: "Related concept",
  confidence: 0.85,
  dismissed: false,
};

function makeSuggestion(overrides: Record<string, unknown> = {}) {
  return { ...baseSuggestion, id: crypto.randomUUID(), ...overrides };
}

// [REQ:P1-001] [REQ:P1-003] LLM suggestions as dismissible ghost nodes
describe("SuggestionList", () => {
  it("renders nothing when suggestions list is empty", () => {
    const { container } = render(
      <SuggestionList suggestions={[]} onDismiss={vi.fn()} onAccept={vi.fn()} />,
    );
    expect(container.firstChild).toBeNull();
  });

  it("renders visible suggestions with confidence", () => {
    const suggestions = [makeSuggestion({ label: "Link A", confidence: 0.72 })];
    render(
      <SuggestionList suggestions={suggestions} onDismiss={vi.fn()} onAccept={vi.fn()} />,
    );
    expect(screen.getByTestId("suggestion-list")).toBeInTheDocument();
    expect(screen.getByText(/Link A/)).toBeInTheDocument();
    expect(screen.getByText(/72%/)).toBeInTheDocument();
  });

  it("calls onAccept when a suggestion is clicked", () => {
    const onAccept = vi.fn();
    const s = makeSuggestion({ label: "Click me" });
    render(
      <SuggestionList suggestions={[s]} onDismiss={vi.fn()} onAccept={onAccept} />,
    );
    fireEvent.click(screen.getByText(/Click me/));
    expect(onAccept).toHaveBeenCalledWith(s);
  });

  it("calls onDismiss when dismiss button is clicked", () => {
    const onDismiss = vi.fn();
    const s = makeSuggestion();
    render(
      <SuggestionList suggestions={[s]} onDismiss={onDismiss} onAccept={vi.fn()} />,
    );
    // The X button is the second button in the suggestion item
    const items = screen.getAllByTestId("suggestion-item");
    const dismissBtn = items[0]?.querySelector("button:last-child");
    expect(dismissBtn).toBeTruthy();
    if (dismissBtn) fireEvent.click(dismissBtn);
    expect(onDismiss).toHaveBeenCalledWith(s.id);
  });

  it("hides dismissed suggestions by default", () => {
    const suggestions = [
      makeSuggestion({ label: "Visible" }),
      makeSuggestion({ label: "Gone", dismissed: true }),
    ];
    render(
      <SuggestionList suggestions={suggestions} onDismiss={vi.fn()} onAccept={vi.fn()} />,
    );
    expect(screen.getByText(/Visible/)).toBeInTheDocument();
    expect(screen.queryByText(/Gone/)).not.toBeInTheDocument();
  });

  it("shows dismissed suggestions when toggle is clicked", () => {
    const suggestions = [
      makeSuggestion({ label: "Active" }),
      makeSuggestion({ label: "Hidden", dismissed: true }),
    ];
    render(
      <SuggestionList suggestions={suggestions} onDismiss={vi.fn()} onAccept={vi.fn()} />,
    );
    fireEvent.click(screen.getByText("Show all"));
    expect(screen.getByText(/Hidden/)).toBeInTheDocument();
  });

  it("displays correct count of visible suggestions", () => {
    const suggestions = [
      makeSuggestion(),
      makeSuggestion(),
      makeSuggestion({ dismissed: true }),
    ];
    render(
      <SuggestionList suggestions={suggestions} onDismiss={vi.fn()} onAccept={vi.fn()} />,
    );
    expect(screen.getByText(/Suggestions \(2\)/)).toBeInTheDocument();
  });
});
