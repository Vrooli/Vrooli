import { fireEvent, screen } from "@testing-library/react";
import { renderWithProviders as render } from "../../test-utils";
import { describe, expect, it, vi } from "vitest";
import { ProposalDecisionStreamView } from "./ProposalDecisionStreamView";

vi.mock("../../services/proposal-session-service", () => ({ proposalSessionService: { decide: vi.fn() } }));

const proposal = (id: string, summary: string) => ({
  sessionId: "session-1", sessionTitle: "Lifecycle triage",
  proposal: { id, kind: "mutation_list", status: "ready", summary, payload_json: JSON.stringify({ mutations: [{ id: `${id}-mutation` }] }), created_at: "", updated_at: "" },
});

describe("ProposalDecisionStreamView", () => {
  it("navigates proposals with the same back, skip, snooze, and next controls as the decision flow", () => {
    const onSnooze = vi.fn();
    render(<ProposalDecisionStreamView proposals={[proposal("p1", "First proposal"), proposal("p2", "Second proposal")]} onComplete={vi.fn()} onBack={vi.fn()} onSnooze={onSnooze} />);
    expect(screen.getByText("First proposal")).toBeInTheDocument();
    expect(screen.getByText("1/2")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /next/i }));
    expect(screen.getByText("Second proposal")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /snooze/i }));
    expect(onSnooze).toHaveBeenCalledWith("p2");
  });

  it("renders capture proposals as first-class decision-stream work", () => {
    render(<ProposalDecisionStreamView
      proposals={[{ ...proposal("capture-p1", "Capture proposal"), target: { type: "capture", ref: "cap-1", name: "A captured idea" } }]}
      onComplete={vi.fn()} onBack={vi.fn()} onSnooze={vi.fn()}
    />);
    expect(screen.getByText("A captured idea")).toBeInTheDocument();
    expect(screen.getByText("Capture proposal")).toBeInTheDocument();
  });
});
