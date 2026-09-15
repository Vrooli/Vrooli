import { describe, expect, it } from "vitest";
import { ApplyRunState, ApplyStepState, type ApplyStep } from "@vrooli/proto-types/vrooli-onboarding/v1/apply/apply_pb";
import { isApplyRunActive, stepFailureReason, summarizeApplyRun } from "./applyRunSummary";

function step(id: string, state: ApplyStepState, error = ""): ApplyStep {
  return { id, name: id.replace(/^scenario:/, ""), state, error, remediation: "", errorCode: "" } as ApplyStep;
}

// The shape a failed scenario start really returns: progress lines, then the
// lifecycle's structured error, then a JSON result line with no "message".
const POSTGRES_FAILURE = [
  "starting secrets-manager...",
  '{"level":"INFO","msg":"Scenario start requested"}',
  '{"level":"ERROR","msg":"Scenario start failed","error":{"message":"start resource dependency postgres: start postgres: verify managed-service process ownership: platform: process environment inspection is not supported on this platform","type":"*fmt.wrapError"}}',
  '{"success":false,"error":"start postgres: …","code":"operation_failed"}',
].join("\n");

describe("stepFailureReason", () => {
  it("takes the lifecycle's last structured message, not the progress chatter", () => {
    expect(stepFailureReason(POSTGRES_FAILURE)).toBe(
      "start resource dependency postgres: start postgres: verify managed-service process ownership: platform: process environment inspection is not supported on this platform",
    );
  });

  it("falls back to the last line of plain output", () => {
    expect(stepFailureReason("fetching…\n\ninstall failed: checksum mismatch\n")).toBe("install failed: checksum mismatch");
  });

  it("bounds a long reason", () => {
    expect(stepFailureReason("x".repeat(1000)).length).toBeLessThanOrEqual(240);
  });
});

describe("summarizeApplyRun", () => {
  it("counts finished steps and names each failure with its cause", () => {
    const summary = summarizeApplyRun(ApplyRunState.PARTIALLY_APPLIED, [
      step("tool:ast-grep", ApplyStepState.APPLIED),
      step("scenario:secrets-manager", ApplyStepState.FAILED, POSTGRES_FAILURE),
      step("scenario:vrooli-onboarding", ApplyStepState.SKIPPED_SELF),
    ]);
    expect(summary.outcome).toBe("partial");
    expect(summary.active).toBe(false);
    expect(summary.total).toBe(3);
    expect(summary.finished).toBe(3);
    expect(summary.failed).toEqual([
      expect.objectContaining({ id: "scenario:secrets-manager", name: "secrets-manager", reason: expect.stringContaining("not supported on this platform") }),
    ]);
  });

  it("treats pending and applying runs as still active", () => {
    expect(isApplyRunActive(ApplyRunState.PENDING)).toBe(true);
    expect(isApplyRunActive(ApplyRunState.APPLYING)).toBe(true);
    expect(isApplyRunActive(ApplyRunState.PARTIALLY_APPLIED)).toBe(false);
    expect(summarizeApplyRun(ApplyRunState.APPLYING, [step("tool:buf", ApplyStepState.APPLYING)]).finished).toBe(0);
  });
});
