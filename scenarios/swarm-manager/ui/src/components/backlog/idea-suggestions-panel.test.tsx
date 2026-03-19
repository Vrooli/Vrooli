import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { IdeaSuggestionsPanel } from "./idea-suggestions-panel";
import type { IdeaSuggestion } from "../../types";

const baseSuggestions: IdeaSuggestion[] = [
  { id: "s1", suggestion: "Add validation", status: "pending" },
  { id: "s2", suggestion: "Use caching", status: "accepted", notes: "High priority" },
  { id: "s3", suggestion: "Remove feature", status: "rejected" },
];

describe("IdeaSuggestionsPanel", () => {
  it("shows notes textarea for accepted suggestions", () => {
    render(
      <IdeaSuggestionsPanel
        suggestions={baseSuggestions}
        filePath="suggest/suggestions.json"
        isSubmitting={false}
        onSubmit={vi.fn()}
      />
    );

    // The accepted suggestion (s2) should have a notes textarea with existing value
    const notesTextareas = screen.getAllByPlaceholderText("Add a note about this decision...");
    // Both accepted (s2) and rejected (s3) should have notes textareas
    expect(notesTextareas).toHaveLength(2);
  });

  it("does not show notes textarea for pending suggestions", () => {
    const suggestions: IdeaSuggestion[] = [
      { id: "s1", suggestion: "Pending item", status: "pending" },
    ];

    render(
      <IdeaSuggestionsPanel
        suggestions={suggestions}
        filePath="suggest/suggestions.json"
        isSubmitting={false}
        onSubmit={vi.fn()}
      />
    );

    expect(screen.queryByPlaceholderText("Add a note about this decision...")).not.toBeInTheDocument();
  });

  it("shows notes textarea when status changes from pending to accepted", () => {
    const suggestions: IdeaSuggestion[] = [
      { id: "s1", suggestion: "Test item", status: "pending" },
    ];

    render(
      <IdeaSuggestionsPanel
        suggestions={suggestions}
        filePath="suggest/suggestions.json"
        isSubmitting={false}
        onSubmit={vi.fn()}
      />
    );

    // Initially no notes textarea
    expect(screen.queryByPlaceholderText("Add a note about this decision...")).not.toBeInTheDocument();

    // Change status to "accepted"
    const select = screen.getByDisplayValue("Pending");
    fireEvent.change(select, { target: { value: "accepted" } });

    // Now notes textarea should appear
    expect(screen.getByPlaceholderText("Add a note about this decision...")).toBeInTheDocument();
  });

  it("preserves existing notes value", () => {
    const suggestions: IdeaSuggestion[] = [
      { id: "s1", suggestion: "Test", status: "accepted", notes: "My note" },
    ];

    render(
      <IdeaSuggestionsPanel
        suggestions={suggestions}
        filePath="suggest/suggestions.json"
        isSubmitting={false}
        onSubmit={vi.fn()}
      />
    );

    const textarea = screen.getByPlaceholderText("Add a note about this decision...");
    expect(textarea).toHaveValue("My note");
  });

  it("includes notes in the submitted suggestions", () => {
    const onSubmit = vi.fn();
    const suggestions: IdeaSuggestion[] = [
      { id: "s1", suggestion: "Test", status: "accepted" },
    ];

    render(
      <IdeaSuggestionsPanel
        suggestions={suggestions}
        filePath="suggest/suggestions.json"
        isSubmitting={false}
        onSubmit={onSubmit}
      />
    );

    // Type a note
    const textarea = screen.getByPlaceholderText("Add a note about this decision...");
    fireEvent.change(textarea, { target: { value: "Important note" } });

    // Submit
    fireEvent.click(screen.getByText("Save Decisions & Run Enhance"));

    expect(onSubmit).toHaveBeenCalledWith(
      expect.arrayContaining([
        expect.objectContaining({ id: "s1", notes: "Important note" }),
      ])
    );
  });
});
