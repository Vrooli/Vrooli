import { describe, expect, it, vi } from "vitest";
import { createProposalSessionService } from "./proposal-session-service";

describe("proposal session decisions", () => {
  it("uses DecideAttempt and reloads the durable session projection", async () => {
    const apiClient = { get: vi.fn().mockResolvedValue({ id: "session-1", proposals: [{ id: "proposal-1", status: "applied" }] }) };
    const decisions = { decide: vi.fn().mockResolvedValue({}) };
    const service = createProposalSessionService(apiClient as never, decisions);

    await expect(service.decide("session-1", "proposal-1", ["mutation-1"], "Approved after review.")).resolves.toMatchObject({ id: "session-1" });

    expect(decisions.decide).toHaveBeenCalledWith({
      subjectKind: "agent-session-proposal",
      subjectRef: "session-1/proposal-1",
      roundNum: 1,
      decision: "accept",
      actor: "operator-ui",
      rationale: "Approved after review.",
      acceptedProposalIds: ["mutation-1"],
    });
    expect(apiClient.get).toHaveBeenCalledWith("/agent-sessions/session-1");
  });
});
