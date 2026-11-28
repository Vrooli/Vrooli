import { describe, expect, it, vi } from "vitest";
import { createPlanWorkshopService } from "./plan-workshop-service";

describe("plan workshop transition application", () => {
  it("applies a review through the declared transition and reloads the session", async () => {
    const apiClient = {
      get: vi.fn()
        .mockResolvedValueOnce({ id: "pw-1", review: { workflow: { execution_id: "exec-review" } } })
        .mockResolvedValueOnce({ id: "pw-1", review: { state: "applied", workflow: { execution_id: "exec-review" } } }),
    };
    const transitions = { list: vi.fn(), start: vi.fn(), apply: vi.fn().mockResolvedValue(undefined) };

    const result = await createPlanWorkshopService(apiClient as never, transitions).applyReview("pw-1");

    expect(transitions.apply).toHaveBeenCalledWith("plan.workshop.review", "exec-review");
    expect(result.review.state).toBe("applied");
  });

  it("applies reconciliation through the declared transition and reloads its resolution", async () => {
    const apiClient = {
      get: vi.fn()
        .mockResolvedValueOnce({ id: "pw-1", resolutions: [{ response_id: "response-1", workflow: { execution_id: "exec-reconcile" } }] })
        .mockResolvedValueOnce({ id: "pw-1", resolutions: [{ response_id: "response-1", state: "candidate_ready", workflow: { execution_id: "exec-reconcile" } }] }),
    };
    const transitions = { list: vi.fn(), start: vi.fn(), apply: vi.fn().mockResolvedValue(undefined) };

    const result = await createPlanWorkshopService(apiClient as never, transitions).applyReconciliation("pw-1", "response-1");

    expect(transitions.apply).toHaveBeenCalledWith("plan.workshop.reconcile", "exec-reconcile");
    expect(result.resolution.state).toBe("candidate_ready");
  });

  it("decides a candidate through DecideAttempt and reloads the session", async () => {
    const apiClient = { get: vi.fn().mockResolvedValue({ id: "pw-1", resolutions: [{ response_id: "response-1", state: "candidate_applied" }] }) };
    const transitions = { list: vi.fn(), start: vi.fn(), apply: vi.fn() };
    const decisions = { decide: vi.fn().mockResolvedValue({}) };

    const result = await createPlanWorkshopService(apiClient as never, transitions, decisions).applyCandidate("pw-1", "response-1");

    expect(decisions.decide).toHaveBeenCalledWith({
      subjectKind: "plan-workshop-candidate",
      subjectRef: "pw-1/response-1",
      roundNum: 1,
      decision: "accept",
      actor: "operator-ui",
      rationale: "Candidate accepted after review.",
    });
    expect(result.resolution.state).toBe("candidate_applied");
  });
});
