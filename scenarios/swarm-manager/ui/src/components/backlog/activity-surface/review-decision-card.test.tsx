import { describe, expect, it, vi } from "vitest";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { renderWithProviders } from "../../../test-utils";
import type { BacklogCriterion } from "@vrooli/proto-types/swarm-manager/v1/shared/backlog_pb";
import type { EvidenceItem } from "../../../services/review-service";

const decide = vi.fn();

vi.mock("../../../services/review-decision-service", () => ({ reviewDecisionService: { decide: (...args: unknown[]) => decide(...args) } }));

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
    expect(screen.queryByRole("button", { name: /Accept/ })).not.toBeInTheDocument();
  });

  it("labels inconclusive review honestly and sends the explicit decision", async () => {
    decide.mockResolvedValue({});
    const onDecided = vi.fn();
    renderWithProviders(<ReviewDecisionCard kind="execute" name="item-a" round={baseRound} onDecided={onDecided} />);
    expect(screen.getByText("inconclusive")).toBeInTheDocument();
    expect(screen.getByText(/Advisory: attention/)).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "View changed files" })).toHaveAttribute("href", "/executions/execution-42?tab=changes");

    fireEvent.change(screen.getByLabelText("Operator identity"), { target: { value: "operator@example.test" } });
    fireEvent.click(screen.getByLabelText("Agree with the review's assessment"));
    fireEvent.click(screen.getByRole("button", { name: /Accept \(0 unverified\)/ }));
    await waitFor(() => expect(decide).toHaveBeenCalledWith(expect.objectContaining({ decision: "accept", actor: "operator@example.test" })));
    expect(onDecided).toHaveBeenCalledOnce();
  });

  it("starts with an empty rationale and renders criterion-bound evidence", () => {
    const criterion = { $typeName: "vrooli.swarm_manager.v1.shared.BacklogCriterion", id: "criterion-1", gherkin: "Given proof When reviewed Then it is settled." } as BacklogCriterion;
    const evidence = { id: "evidence-1", type: "api_test", title: "API contract", description: "response", verified: true, criterion_id: "criterion-1", settlement: "settled" } as EvidenceItem;
    renderWithProviders(<ReviewDecisionCard kind="execute" name="item-a" round={{ ...baseRound, evidence: [evidence] }} criteria={[criterion]} onDecided={vi.fn()} />);
    expect(screen.getByLabelText("Decision rationale")).toHaveValue("");
    expect(screen.getByText("Given proof When reviewed Then it is settled.")).toBeInTheDocument();
    expect(screen.getByText("settled")).toBeInTheDocument();
  });

  it("warns about criteria that are actually unsettled, not about missing props", () => {
    // The old warning fired whenever `criteria` was empty, which in the
    // decision stream was always, because the call site never passed it.
    const criteria = [
      { $typeName: "vrooli.swarm_manager.v1.shared.BacklogCriterion", id: "c1", gherkin: "Proven by evidence." },
      { $typeName: "vrooli.swarm_manager.v1.shared.BacklogCriterion", id: "c2", gherkin: "Nothing proves this yet." },
    ] as BacklogCriterion[];
    const evidence = [{ id: "e1", type: "api_test", title: "t", description: "d", verified: true, criterion_id: "c1", settlement: "settled" }] as EvidenceItem[];
    renderWithProviders(<ReviewDecisionCard kind="execute" name="item-a" round={{ ...baseRound, evidence }} criteria={criteria} onDecided={vi.fn()} />);

    expect(screen.getByText("1 of 2 criteria unsettled. Accepting closes them as delivered.")).toBeInTheDocument();
    expect(screen.queryByText(/no typed criteria/)).not.toBeInTheDocument();
    expect(screen.getByText("unsettled")).toBeInTheDocument();
  });

  it("confirms when every criterion is settled", () => {
    const criteria = [{ $typeName: "vrooli.swarm_manager.v1.shared.BacklogCriterion", id: "c1", gherkin: "Proven." }] as BacklogCriterion[];
    const evidence = [{ id: "e1", type: "api_test", title: "t", description: "d", verified: true, criterion_id: "c1", settlement: "settled" }] as EvidenceItem[];
    renderWithProviders(<ReviewDecisionCard kind="execute" name="item-a" round={{ ...baseRound, evidence }} criteria={criteria} onDecided={vi.fn()} />);
    expect(screen.getByText("All 1 criteria settled by evidence.")).toBeInTheDocument();
  });

  it("still warns when the item genuinely has no typed criteria", () => {
    renderWithProviders(<ReviewDecisionCard kind="execute" name="item-a" round={baseRound} criteria={[]} onDecided={vi.fn()} />);
    expect(screen.getByText(/no typed criteria/)).toBeInTheDocument();
  });

  it("remembers the operator identity across decisions", async () => {
    decide.mockResolvedValue({});
    window.localStorage.clear();
    const first = renderWithProviders(<ReviewDecisionCard kind="execute" name="item-a" round={baseRound} onDecided={vi.fn()} />);
    fireEvent.change(screen.getByLabelText("Operator identity"), { target: { value: "matthalloran8" } });
    fireEvent.click(screen.getByLabelText("Agree with the review's assessment"));
    fireEvent.click(screen.getByRole("button", { name: /Accept/ }));
    await waitFor(() => expect(decide).toHaveBeenCalled());
    first.unmount();

    // A queue of hundreds of decisions must not demand the same keystrokes
    // each time; a retyped-and-typo'd actor splits one person's audit trail.
    renderWithProviders(<ReviewDecisionCard kind="execute" name="item-b" round={baseRound} onDecided={vi.fn()} />);
    expect(screen.getByLabelText("Operator identity")).toHaveValue("matthalloran8");
  });

  it("opens the prepared follow-up path only after saving Send back", async () => {
    decide.mockResolvedValue({});
    const onSendBack = vi.fn();
    renderWithProviders(<ReviewDecisionCard kind="execute" name="item-a" round={baseRound} onDecided={vi.fn()} onSendBack={onSendBack} />);
    fireEvent.change(screen.getByLabelText("Operator identity"), { target: { value: "operator@example.test" } });
    fireEvent.change(screen.getByLabelText("Decision rationale"), { target: { value: "Investigate the missing proof." } });
    fireEvent.change(screen.getByLabelText("Follow-up disposition"), { target: { value: "replan" } });
    fireEvent.click(screen.getByRole("button", { name: "Send back" }));
    await waitFor(() => expect(decide).toHaveBeenCalledWith(expect.objectContaining({ decision: "followup", followUp: { steering: "Investigate the missing proof.", disposition: "replan" } })));
    expect(onSendBack).toHaveBeenCalledOnce();
  });
});
