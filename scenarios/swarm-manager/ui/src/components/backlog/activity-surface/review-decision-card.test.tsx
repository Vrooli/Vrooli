import { describe, expect, it, vi } from "vitest";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { renderWithProviders } from "../../../test-utils";

const post = vi.fn();

vi.mock("../../../lib/api-client", () => ({
  defaultApiClient: { post: (...args: unknown[]) => post(...args) },
}));

import { ReviewDecisionCard } from "./review-decision-card";

const baseRound = {
  round: 2,
  generated_at: "2026-07-22T12:00:00Z",
  execution_id: "execution-42",
  status: "complete" as const,
  classification: "inconclusive",
  agent_assessment: "The available evidence does not establish a safe verdict.",
  disposition: { kind: "attention", rationale: "An operator should decide with the current evidence.", confidence: "medium" as const },
  evidence: [{ id: "evidence-1", type: "api_test" as const, title: "API contract", description: "response", verified: true }],
};

describe("ReviewDecisionCard", () => {
  it("shows gathering instead of terminal controls while evidence is live", () => {
    renderWithProviders(<ReviewDecisionCard kind="execute" name="item-a" round={{ ...baseRound, status: "gathering" }} onDecided={vi.fn()} />);
    expect(screen.getByText("Review is gathering evidence")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Accept" })).not.toBeInTheDocument();
  });

  it("labels inconclusive review honestly and sends the explicit decision", async () => {
    post.mockResolvedValue({});
    const onDecided = vi.fn();
    renderWithProviders(<ReviewDecisionCard kind="execute" name="item-a" round={baseRound} onDecided={onDecided} />);
    expect(screen.getByText("inconclusive")).toBeInTheDocument();
    expect(screen.getByText(/Advisory: attention/)).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "View changed files" })).toHaveAttribute("href", "/executions/execution-42?tab=changes");

    fireEvent.click(screen.getByRole("button", { name: "Accept" }));
    await waitFor(() => expect(post).toHaveBeenCalledWith("/backlog/execute/item-a/review-decide", expect.objectContaining({ decision: "accept" })));
    expect(onDecided).toHaveBeenCalledOnce();
  });

  it("opens the prepared follow-up path only after saving Send back", async () => {
    post.mockResolvedValue({});
    const onSendBack = vi.fn();
    renderWithProviders(<ReviewDecisionCard kind="execute" name="item-a" round={baseRound} onDecided={vi.fn()} onSendBack={onSendBack} />);
    fireEvent.click(screen.getByRole("button", { name: "Send back" }));
    await waitFor(() => expect(post).toHaveBeenCalledWith("/backlog/execute/item-a/review-decide", expect.objectContaining({ decision: "followup" })));
    expect(onSendBack).toHaveBeenCalledOnce();
  });
});
