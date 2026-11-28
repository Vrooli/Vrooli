import { describe, expect, it } from "vitest";
import { buildTypedInvestigationRequest } from "./typedInvestigation";

describe("buildTypedInvestigationRequest", () => {
  it("creates a bounded read-only request for the finite lifecycle", () => {
    const request = buildTypedInvestigationRequest(
      ["run-b", "run-a", "run-b"],
      "  Explain the supported failure evidence.  ",
      "deep",
      "agent-manager-ui/request-1",
    );

    expect(request.subject.runIds).toEqual(["run-a", "run-b"]);
    expect(request.subject.ref).toBe("run-set:run-a,run-b");
    expect(request.question).toBe("Explain the supported failure evidence.");
    expect(request.budget.maxTurns).toBe(16);
    expect(request.evidencePolicy.requiredPlanes).toEqual(["run_state", "events", "invocations"]);
    expect(request.recommendationPolicy.allowSubjectMutation).toBe(false);
  });

  it("rejects an empty subject instead of creating an unbounded diagnosis", () => {
    expect(() => buildTypedInvestigationRequest([], "diagnose", "standard", "request-1")).toThrow(
      "At least one subject run is required",
    );
  });
});
