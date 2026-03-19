import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { SuggestionFormDialog } from "./suggestion-form-dialog";

describe("SuggestionFormDialog", () => {
  it("shows notes field when status is accepted", () => {
    render(
      <SuggestionFormDialog
        isOpen={true}
        mode="edit"
        initialValues={{
          suggestion: "Test suggestion",
          status: "accepted",
          notes: "My note",
        }}
        onClose={vi.fn()}
        onSubmit={vi.fn()}
      />
    );

    const notesInput = screen.getByLabelText("Notes (optional)");
    expect(notesInput).toBeInTheDocument();
    expect(notesInput).toHaveValue("My note");
  });

  it("shows notes field when status is rejected", () => {
    render(
      <SuggestionFormDialog
        isOpen={true}
        mode="edit"
        initialValues={{
          suggestion: "Test suggestion",
          status: "rejected",
        }}
        onClose={vi.fn()}
        onSubmit={vi.fn()}
      />
    );

    expect(screen.getByLabelText("Notes (optional)")).toBeInTheDocument();
  });

  it("hides notes field when status is pending", () => {
    render(
      <SuggestionFormDialog
        isOpen={true}
        mode="create"
        initialValues={{
          suggestion: "Test suggestion",
          status: "pending",
        }}
        onClose={vi.fn()}
        onSubmit={vi.fn()}
      />
    );

    expect(screen.queryByLabelText("Notes (optional)")).not.toBeInTheDocument();
  });

  it("includes notes in submit payload", () => {
    const onSubmit = vi.fn();

    render(
      <SuggestionFormDialog
        isOpen={true}
        mode="edit"
        initialValues={{
          suggestion: "Test suggestion",
          status: "accepted",
          notes: "A note",
        }}
        onClose={vi.fn()}
        onSubmit={onSubmit}
      />
    );

    fireEvent.click(screen.getByText("Save Changes"));

    expect(onSubmit).toHaveBeenCalledWith(
      expect.objectContaining({
        suggestion: "Test suggestion",
        status: "accepted",
        notes: "A note",
      })
    );
  });

  it("trims empty notes to undefined in submit payload", () => {
    const onSubmit = vi.fn();

    render(
      <SuggestionFormDialog
        isOpen={true}
        mode="edit"
        initialValues={{
          suggestion: "Test suggestion",
          status: "accepted",
          notes: "   ",
        }}
        onClose={vi.fn()}
        onSubmit={onSubmit}
      />
    );

    fireEvent.click(screen.getByText("Save Changes"));

    expect(onSubmit).toHaveBeenCalledWith(
      expect.objectContaining({
        notes: undefined,
      })
    );
  });
});
