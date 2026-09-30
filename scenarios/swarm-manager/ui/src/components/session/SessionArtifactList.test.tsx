import { describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { SessionArtifactList } from "./SessionArtifactList";
import type { AgentSessionProposal } from "../../types";

const proposal: AgentSessionProposal = {
  id: "prop-transition",
  kind: "start_transition",
  status: "ready",
  summary: "Start the declared transition",
  payloadJson: JSON.stringify({
    transition_key: "goal.close_out",
    subject_ref: "goal:release",
    projection_action: "keep_open",
    projection_agrees: false,
    reason: "The acceptance evidence is complete even though the projection is stale.",
  }),
  createdAt: "2026-08-11T00:00:00Z",
  updatedAt: "2026-08-11T00:00:00Z",
};

describe("SessionArtifactList start_transition proposals", () => {
  it("renders both verdicts and a distinct approval control when they disagree", async () => {
    const onApplyProposal = vi.fn();
    render(
      <SessionArtifactList
        artifacts={[]}
        proposals={[proposal]}
        onOpenArtifact={() => undefined}
        onApplyProposal={onApplyProposal}
      />,
    );

    expect(screen.getByTestId("agent-session-transition-verdicts")).toHaveTextContent("Projection: keep_open · Session: disagrees");
    expect(screen.getByTestId("agent-session-transition-disagreement")).toHaveTextContent("acceptance evidence is complete");
    await screen.getByTestId("agent-session-approve-transition").click();
    expect(onApplyProposal).toHaveBeenCalledWith(proposal);
  });
});
