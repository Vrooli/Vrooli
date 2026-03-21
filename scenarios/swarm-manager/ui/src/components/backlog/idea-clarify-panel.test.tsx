import { describe, it, expect, vi } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import { IdeaClarifyPanel } from "./idea-clarify-panel";
import { parseClarifyQuestionsFile } from "../../lib/idea-agent-files";
import type { IdeaClarificationQuestion } from "../../types";

// Real data format from tunnel-manager/clarify/questions.json
const realQuestions: IdeaClarificationQuestion[] = [
  {
    id: "Q1",
    question: "Which approach for sudo access?",
    options: [
      "A: sudoers NOPASSWD rule for the specific systemctl restart cloudflared command",
      "B: Use polkit rule to allow the vrooli user to manage the cloudflared unit",
      "C: Run tunnel-manager as a systemd service with AmbientCapabilities or a drop-in override",
    ],
    // Answer doesn't match any option → should select "Other"
    answer: "Try to find a way around having to use sudo.",
  },
  {
    id: "Q2",
    question: "Remote-managed tunnel migration approach?",
    options: [
      "A: Support both modes from day one; auto-detect current mode on startup",
      "B: Require remote mode as a prerequisite; provide a one-time migration guide",
      "C: Start with local mode only matching current state; add remote mode as P1",
    ],
    // Answer exactly matches option A
    answer: "A: Support both modes from day one; auto-detect current mode on startup",
  },
  {
    id: "Q3",
    question: "Route manifest seeding approach?",
    options: [
      "A: Interactive CLI prompt showing discovered routes, ask to confirm each",
      "B: Generate a draft routes.json file that the user reviews and manually approves",
      "C: Fully automatic seeding from current config with no confirmation needed",
    ],
    // Answer exactly matches option A
    answer: "A: Interactive CLI prompt showing discovered routes, ask to confirm each",
  },
];

describe("IdeaClarifyPanel — radio selection on load", () => {
  it("shows correct radio selection for pre-filled answers matching options", () => {
    render(
      <IdeaClarifyPanel
        questions={realQuestions}
        filePath="clarify/questions.json"
        isSubmitting={false}
        onSubmit={vi.fn()}
      />
    );

    // Q2: answer matches option A exactly → option A radio should be checked
    const q2OptionA = screen.getByLabelText(
      "A: Support both modes from day one; auto-detect current mode on startup",
    );
    expect(q2OptionA).toBeChecked();

    // Q3: answer matches option A exactly → option A radio should be checked
    const q3OptionA = screen.getByLabelText(
      "A: Interactive CLI prompt showing discovered routes, ask to confirm each",
    );
    expect(q3OptionA).toBeChecked();
  });

  it("selects Other radio when answer does not match any predefined option", () => {
    render(
      <IdeaClarifyPanel
        questions={realQuestions}
        filePath="clarify/questions.json"
        isSubmitting={false}
        onSubmit={vi.fn()}
      />
    );

    // Q1: answer is custom text → "Other" should be checked
    // There are multiple "Other" radios (one per question), so find the one
    // in Q1's context by using testIdPrefix
    const otherRadios = screen.getAllByLabelText("Other");
    // Q1 is index 0, so otherRadios[0] should be the Q1 Other radio
    expect(otherRadios[0]).toBeChecked();

    // The textarea should show the custom answer
    const textareas = screen.getAllByPlaceholderText("Specify your answer...");
    expect(textareas[0]).toHaveValue("Try to find a way around having to use sudo.");
  });
});

describe("IdeaClarifyPanel — parsed from JSON string (API simulation)", () => {
  // Simulates the exact JSON the API returns
  const apiJsonResponse = JSON.stringify({
    version: "1.0.0",
    generated_at: "2026-02-18T12:00:00Z",
    questions: [
      {
        id: "Q1",
        question: "Which approach for sudo access?",
        options: [
          "A: sudoers NOPASSWD rule",
          "B: Use polkit rule",
          "C: Run as systemd service",
        ],
        answer: "Custom free-text answer that doesn't match any option",
      },
      {
        id: "Q2",
        question: "Remote tunnel migration?",
        options: [
          "A: Support both modes from day one",
          "B: Require remote mode as prerequisite",
          "C: Start with local mode only",
        ],
        answer: "A: Support both modes from day one",
      },
    ],
    generatedAt: "2026-02-19T04:52:19.336Z",
    updatedAt: "2026-02-19T04:52:19.336Z",
  });

  it("radio buttons reflect saved answers after parsing from API JSON", () => {
    const { questions } = parseClarifyQuestionsFile(apiJsonResponse);
    expect(questions).toHaveLength(2);

    render(
      <IdeaClarifyPanel
        questions={questions}
        filePath="clarify/questions.json"
        isSubmitting={false}
        onSubmit={vi.fn()}
      />
    );

    // Q1: custom answer → Other should be checked
    const otherRadios = screen.getAllByLabelText("Other");
    expect(otherRadios[0]).toBeChecked();

    // Q2: answer matches option A → option A should be checked
    const q2OptionA = screen.getByLabelText("A: Support both modes from day one");
    expect(q2OptionA).toBeChecked();

    // Q2 Other should NOT be checked
    expect(otherRadios[1]).not.toBeChecked();
  });

  it("radio buttons correct after prop change (simulates query refetch)", async () => {
    // Start with empty questions (simulates initial load before API returns)
    const { rerender } = render(
      <IdeaClarifyPanel
        questions={[]}
        filePath="clarify/questions.json"
        isSubmitting={false}
        onSubmit={vi.fn()}
      />
    );

    // No radio buttons initially
    expect(screen.queryAllByRole("radio")).toHaveLength(0);

    // Simulate API data arriving (rerender with parsed questions)
    const { questions } = parseClarifyQuestionsFile(apiJsonResponse);
    rerender(
      <IdeaClarifyPanel
        questions={questions}
        filePath="clarify/questions.json"
        isSubmitting={false}
        onSubmit={vi.fn()}
      />
    );

    // After rerender, radio buttons should appear with correct selections
    await waitFor(() => {
      const q2OptionA = screen.getByLabelText("A: Support both modes from day one");
      expect(q2OptionA).toBeChecked();
    });

    // Q1 Other should be checked
    const otherRadios = screen.getAllByLabelText("Other");
    expect(otherRadios[0]).toBeChecked();
  });
});
