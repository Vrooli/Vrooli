import { describe, expect, it, vi } from "vitest";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { InlineQuestionStepper } from "./inline-question-stepper";

vi.mock("../../services/backlog-service", () => ({
  backlogService: { batchReview: vi.fn().mockResolvedValue(undefined) },
}));

const question = {
  id: "OT-P0-001", source: "review" as const, item_kind: "idea" as const,
  item_name: "test-item", title: "Core functionality", description: "Must work",
  criticality: "P0" as const, review_status: "unreviewed" as const, review_type: "target" as const,
};

describe("InlineQuestionStepper", () => {
  it("records an independent review decision and completes", async () => {
    const onAllAnswered = vi.fn();
    render(<InlineQuestionStepper questions={[question]} backlogKind="idea" backlogName="test-item" onAllAnswered={onAllAnswered} />);
    fireEvent.click(screen.getByTestId("question-stepper-review-approve"));
    fireEvent.click(screen.getByTestId("question-stepper-next"));
    await waitFor(() => expect(onAllAnswered).toHaveBeenCalledWith({}));
  });
});
