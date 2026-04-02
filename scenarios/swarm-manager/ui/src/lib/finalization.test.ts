import { describe, it, expect } from "vitest";
import {
  resolvePostRunExecution,
  canRunPostRunChecks,
  hasActionableFinalizationIssues,
  getExecutionReviewResults,
  buildFinalizationContext,
} from "./finalization";
import type { ExecutionRecord, Finalization, ReviewResult } from "../types";

const makeExecution = (overrides?: Partial<ExecutionRecord>): ExecutionRecord => ({
  executionId: "exec-1",
  backlogKind: "idea",
  backlogName: "test-feature",
  status: "completed",
  mode: "manual",
  createdAt: "2026-03-20T00:00:00Z",
  updatedAt: "2026-03-20T01:00:00Z",
  ...overrides,
});

const makeFinalization = (overrides?: Partial<Finalization>): Finalization => ({
  eligible: true,
  status: "completed",
  phase: "completed",
  scopeSource: "sandbox_diff",
  warnings: [],
  affectedScenarios: ["swarm-manager"],
  aggregateClassification: "ready",
  scenarios: [],
  ...overrides,
});

const makeReviewResult = (overrides?: Partial<ReviewResult>): ReviewResult => ({
  jobId: "job-1",
  classification: "needs_work",
  dimensions: [],
  summary: "Fix the tests",
  reviewedAt: "2026-03-20T03:00:00Z",
  ...overrides,
});

describe("resolvePostRunExecution", () => {
  it("returns execution as-is when finalization exists", () => {
    const finalization = makeFinalization();
    const exec = makeExecution({ finalization });
    const result = resolvePostRunExecution(exec);
    expect(result).toBe(exec);
  });

  it("returns execution with synthetic finalization when status is validating", () => {
    const exec = makeExecution({ status: "validating" });
    const result = resolvePostRunExecution(exec);
    expect(result).not.toBeNull();
    expect(result!.finalization).toBeDefined();
    expect(result!.finalization!.status).toBe("running");
    expect(result!.finalization!.phase).toBe("scope_detection");
    expect(result!.executionId).toBe("exec-1");
  });

  it("returns null for completed execution without finalization", () => {
    const exec = makeExecution({ status: "completed" });
    expect(resolvePostRunExecution(exec)).toBeNull();
  });

  it("returns null for failed execution without finalization", () => {
    const exec = makeExecution({ status: "failed" });
    expect(resolvePostRunExecution(exec)).toBeNull();
  });

  it("returns null for running execution", () => {
    const exec = makeExecution({ status: "running" });
    expect(resolvePostRunExecution(exec)).toBeNull();
  });

  it("returns null for pending execution", () => {
    const exec = makeExecution({ status: "pending" });
    expect(resolvePostRunExecution(exec)).toBeNull();
  });

  it("prefers real finalization over validating status", () => {
    const finalization = makeFinalization({ status: "failed" });
    const exec = makeExecution({ status: "validating", finalization });
    const result = resolvePostRunExecution(exec);
    expect(result).toBe(exec);
    expect(result!.finalization!.status).toBe("failed");
  });

  it("does not mutate the original execution", () => {
    const exec = makeExecution({ status: "validating" });
    const result = resolvePostRunExecution(exec);
    expect(exec.finalization).toBeUndefined();
    expect(result).not.toBe(exec);
  });
});

describe("canRunPostRunChecks", () => {
  it("returns true for completed", () => {
    expect(canRunPostRunChecks(makeExecution({ status: "completed" }))).toBe(true);
  });

  it("returns true for failed", () => {
    expect(canRunPostRunChecks(makeExecution({ status: "failed" }))).toBe(true);
  });

  it("returns true for needs_fixup", () => {
    expect(canRunPostRunChecks(makeExecution({ status: "needs_fixup" }))).toBe(true);
  });

  it("returns false for validating", () => {
    expect(canRunPostRunChecks(makeExecution({ status: "validating" }))).toBe(false);
  });

  it("returns false for running", () => {
    expect(canRunPostRunChecks(makeExecution({ status: "running" }))).toBe(false);
  });
});

describe("hasActionableFinalizationIssues", () => {
  it("returns false when no finalization", () => {
    expect(hasActionableFinalizationIssues(makeExecution())).toBe(false);
  });

  it("returns true when finalization failed", () => {
    const exec = makeExecution({ finalization: makeFinalization({ status: "failed" }) });
    expect(hasActionableFinalizationIssues(exec)).toBe(true);
  });

  it("returns true when aggregate classification is needs_work", () => {
    const exec = makeExecution({
      finalization: makeFinalization({ aggregateClassification: "needs_work" }),
    });
    expect(hasActionableFinalizationIssues(exec)).toBe(true);
  });

  it("returns false when aggregate classification is ready", () => {
    const exec = makeExecution({
      finalization: makeFinalization({ aggregateClassification: "ready" }),
    });
    expect(hasActionableFinalizationIssues(exec)).toBe(false);
  });
});

describe("getExecutionReviewResults", () => {
  it("returns empty array when no finalization", () => {
    expect(getExecutionReviewResults(makeExecution())).toEqual([]);
  });

  it("extracts review results from scenarios", () => {
    const review = makeReviewResult();
    const exec = makeExecution({
      finalization: makeFinalization({
        scenarios: [{
          scenarioName: "test",
          changedPaths: [],
          restart: { status: "completed", attempts: 1 },
          health: { status: "completed", scenarioStatus: "running", healthStatus: "healthy", schemaValid: true },
          review: { status: "completed", result: review },
        }],
      }),
    });
    const results = getExecutionReviewResults(exec);
    expect(results).toHaveLength(1);
    expect(results[0]!.classification).toBe("needs_work");
  });

  it("skips scenarios without review results", () => {
    const exec = makeExecution({
      finalization: makeFinalization({
        scenarios: [{
          scenarioName: "test",
          changedPaths: [],
          restart: { status: "completed", attempts: 1 },
          health: { status: "completed", scenarioStatus: "running", healthStatus: "healthy", schemaValid: true },
          review: { status: "skipped", skipReason: "no changes" },
        }],
      }),
    });
    expect(getExecutionReviewResults(exec)).toEqual([]);
  });
});

describe("buildFinalizationContext", () => {
  it("returns empty string when no finalization", () => {
    expect(buildFinalizationContext(undefined)).toBe("");
  });

  it("includes aggregate summary", () => {
    const result = buildFinalizationContext(makeFinalization({ aggregateSummary: "All good" }));
    expect(result).toContain("All good");
  });

  it("includes warnings", () => {
    const result = buildFinalizationContext(
      makeFinalization({
        warnings: [{ code: "W001", message: "Something happened", retryable: false, createdAt: "2026-03-20T00:00:00Z" }],
      }),
    );
    expect(result).toContain("warning [W001]");
    expect(result).toContain("Something happened");
  });
});
