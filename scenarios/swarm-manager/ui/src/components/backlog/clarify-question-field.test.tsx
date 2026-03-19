import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { ClarifyQuestionField } from "./clarify-question-field";
import type { IdeaClarificationQuestion } from "../../types";

const baseQuestion: IdeaClarificationQuestion = {
  id: "q1",
  question: "Which approach?",
  options: ["Option A", "Option B", "Option C"],
  answer: "",
};

describe("ClarifyQuestionField", () => {
  it("shows no selection when answer is empty", () => {
    render(
      <ClarifyQuestionField
        question={baseQuestion}
        index={0}
        onChange={vi.fn()}
      />
    );

    const radios = screen.getAllByRole("radio");
    radios.forEach((radio) => {
      expect(radio).not.toBeChecked();
    });
  });

  it("selects the matching option when answer matches a predefined option", () => {
    const question = { ...baseQuestion, answer: "Option B" };

    render(
      <ClarifyQuestionField
        question={question}
        index={0}
        onChange={vi.fn()}
      />
    );

    const optionB = screen.getByLabelText("Option B");
    expect(optionB).toBeChecked();

    // "Other" should not be checked
    const other = screen.getByLabelText("Other");
    expect(other).not.toBeChecked();

    // No textarea for "other" answer should be visible
    expect(screen.queryByPlaceholderText("Specify your answer...")).not.toBeInTheDocument();
  });

  it("selects Other and shows textarea when answer does not match any option", () => {
    const question = { ...baseQuestion, answer: "A custom answer that doesn't match" };

    render(
      <ClarifyQuestionField
        question={question}
        index={0}
        onChange={vi.fn()}
      />
    );

    const other = screen.getByLabelText("Other");
    expect(other).toBeChecked();

    // The textarea should be visible with the custom answer
    const textarea = screen.getByPlaceholderText("Specify your answer...");
    expect(textarea).toBeInTheDocument();
    expect(textarea).toHaveValue("A custom answer that doesn't match");
  });

  it("matches options with trimmed comparison", () => {
    const question = { ...baseQuestion, answer: "  Option A  " };

    render(
      <ClarifyQuestionField
        question={question}
        index={0}
        onChange={vi.fn()}
      />
    );

    // Should match "Option A" via trimmed comparison
    const optionA = screen.getByLabelText("Option A");
    expect(optionA).toBeChecked();
  });

  it("clicking Other clears the answer and shows textarea", () => {
    const onChange = vi.fn();
    const question = { ...baseQuestion, answer: "Option A" };

    render(
      <ClarifyQuestionField
        question={question}
        index={0}
        onChange={onChange}
      />
    );

    fireEvent.click(screen.getByLabelText("Other"));
    expect(onChange).toHaveBeenCalledWith(
      expect.objectContaining({ answer: "" })
    );
  });

  it("clicking a predefined option sets the answer", () => {
    const onChange = vi.fn();
    const question = { ...baseQuestion, answer: "" };

    render(
      <ClarifyQuestionField
        question={question}
        index={0}
        onChange={onChange}
      />
    );

    fireEvent.click(screen.getByLabelText("Option C"));
    expect(onChange).toHaveBeenCalledWith(
      expect.objectContaining({ answer: "Option C" })
    );
  });

  it("renders a plain textarea when no options are provided", () => {
    const question: IdeaClarificationQuestion = {
      id: "q2",
      question: "Describe your approach",
      answer: "My approach",
    };

    render(
      <ClarifyQuestionField
        question={question}
        index={0}
        onChange={vi.fn()}
      />
    );

    expect(screen.queryAllByRole("radio")).toHaveLength(0);
    const textarea = screen.getByPlaceholderText("Your answer...");
    expect(textarea).toHaveValue("My approach");
  });
});
