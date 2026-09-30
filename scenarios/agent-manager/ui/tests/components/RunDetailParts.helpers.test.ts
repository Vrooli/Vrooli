import assert from "node:assert/strict";
import { test } from "vitest";
import { ApprovalState, ExecutionMode, RunMode, RunPhase, RunStatus, TaskStatus } from "../../src/types.js";
import { executionModeLabel, formatCurrency, getCostTotals, isInteractiveRun, runModeLabel, runPhaseLabel, runStatusLabel, taskStatusLabel } from "../../src/components/RunDetailParts.js";

test("run detail helpers normalize every status, mode, phase, and task status", () => {
  assert.equal(runStatusLabel(RunStatus.NEEDS_REVIEW), "needs_review");
  assert.equal(runStatusLabel(RunStatus.COMPLETE, ApprovalState.REJECTED), "rejected");
  assert.equal(runStatusLabel(999 as RunStatus), "pending");
  assert.equal(runModeLabel(RunMode.SANDBOXED), "sandboxed");
  assert.equal(runModeLabel(RunMode.IN_PLACE), "in_place");
  assert.equal(runModeLabel(999 as RunMode), "unspecified");
  assert.equal(executionModeLabel(ExecutionMode.INTERACTIVE), "interactive");
  assert.equal(executionModeLabel(ExecutionMode.CODEC_PIPE), "codec_pipe");
  assert.equal(executionModeLabel(999 as ExecutionMode), "codec_pipe");
  assert.equal(isInteractiveRun(ExecutionMode.INTERACTIVE), true);
  assert.equal(isInteractiveRun(ExecutionMode.CODEC_PIPE), false);
  for (const [phase, label] of [[RunPhase.QUEUED, "queued"], [RunPhase.INITIALIZING, "initializing"], [RunPhase.SANDBOX_CREATING, "sandbox_creating"], [RunPhase.RUNNER_ACQUIRING, "runner_acquiring"], [RunPhase.EXECUTING, "executing"], [RunPhase.COLLECTING_RESULTS, "collecting_results"], [RunPhase.AWAITING_REVIEW, "awaiting_review"], [RunPhase.APPLYING, "applying"], [RunPhase.CLEANING_UP, "cleaning_up"], [RunPhase.COMPLETED, "completed"]] as const) assert.equal(runPhaseLabel(phase), label);
  for (const [status, label] of [[TaskStatus.QUEUED, "queued"], [TaskStatus.RUNNING, "running"], [TaskStatus.NEEDS_REVIEW, "needs_review"], [TaskStatus.APPROVED, "approved"], [TaskStatus.REJECTED, "rejected"], [TaskStatus.FAILED, "failed"], [TaskStatus.CANCELLED, "cancelled"]] as const) assert.equal(taskStatusLabel(status), label);
});

test("getCostTotals only folds valid cost events and preserves model/tier provenance", () => {
  const totals = getCostTotals([
    { data: { case: "log", value: {} } },
    { data: { case: "cost", value: undefined } },
    { data: { case: "cost", value: { inputTokens: 10, outputTokens: 2, cacheCreationTokens: 1, cacheReadTokens: 3, totalCostUsd: 0.25, webSearchRequests: 1, serverToolUseRequests: 2, model: "gpt-5", serviceTier: "priority" } } },
    { data: { case: "cost", value: { inputTokens: 5, model: "gpt-5", serviceTier: "priority" } } },
  ] as any);
  assert.deepEqual(totals, { inputTokens: 15, outputTokens: 2, cacheCreationTokens: 1, cacheReadTokens: 3, totalCostUsd: 0.25, webSearchRequests: 1, serverToolUseRequests: 2, models: ["gpt-5"], serviceTiers: ["priority"], events: 2 });
  assert.equal(formatCurrency(1.2), "$1.2000");
});
