import { describe, expect, it } from "vitest";
import { listAgentSessionsResponseSchema } from "./agent-session-contracts";

/**
 * The client validates responses against the proto's buf.validate rules, and
 * protovalidate fails the whole message on a single violation. So a proposal
 * target type the server persists but the proto omits does not degrade one
 * row — it rejects the entire list response with "Invalid agent sessions
 * response" and blanks the Sessions view.
 *
 * That regression shipped: nine live sessions targeted captures while the
 * proto allowed only backlog_item and goal. The server-side twin of this test
 * is TestProposalTargetTypesMatchProtoContract, which pins the proto's allowed
 * list to agentsessions.ProposalTargetTypes.
 */
function sessionWithTarget(id: string, type: string): Record<string, unknown> {
  return {
    id,
    title: "Session",
    kind: "swarm_operations",
    status: "complete",
    skill_id: "swarm-manager-operations-session",
    created_at: "2026-08-10T14:30:49Z",
    updated_at: "2026-08-10T15:01:18Z",
    proposal_target: { type, ref: "ref-1", name: "Target" },
  };
}

describe("listAgentSessionsResponseSchema", () => {
  it.each(["backlog_item", "goal", "capture"])(
    "accepts a session whose proposal targets a %s",
    (type) => {
      const result = listAgentSessionsResponseSchema.safeParse({
        sessions: [sessionWithTarget("sess_1", type)],
      });
      expect(result.success).toBe(true);
    },
  );

  it("rejects the whole response for one bad target, which is why the sets must agree", () => {
    const result = listAgentSessionsResponseSchema.safeParse({
      sessions: [sessionWithTarget("sess_1", "goal"), sessionWithTarget("sess_2", "execution")],
    });

    // Documenting the blast radius, not endorsing it: the healthy first
    // session is lost along with the malformed second one.
    expect(result.success).toBe(false);
  });
});
