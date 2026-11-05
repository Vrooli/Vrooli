import { afterEach, describe, expect, it, vi } from "vitest";
import type { IApiClient } from "../lib/api-client";
import { createExecutionService } from "./execution-service";

const clients: IApiClient[] = [];
function fixture(data: unknown) {
  const api: IApiClient = { get: vi.fn().mockResolvedValue(data), post: vi.fn(), put: vi.fn(), patch: vi.fn(), delete: vi.fn() };
  clients.push(api); return { api, service: createExecutionService(api) };
}
const execution = { execution_id: "historical-fixture", backlog_kind: "execute", backlog_name: "fixture-item", status: "failed", mode: "manual", created_at: "created", updated_at: "updated" };
afterEach(() => {
  for (const api of clients.splice(0)) for (const method of ["post", "put", "patch", "delete"] as const) expect(api[method]).not.toHaveBeenCalled();
  vi.restoreAllMocks();
});
describe("execution history read projection (fixture history only)", () => {
  it("preserves historical failure and correlated optional selections without creating work", async () => {
    const f = fixture({ execution: { ...execution, task_id: "historical-task", run_id: "historical-run", failure_reason: "original refusal", parent_execution_id: "historical-parent", fixup_attempt: 2, execution_preferences: { preferred_runner: "fixture-runner", model: "fixture-model", effort: "high" }, actual_runner: "fixture-runner", actual_model: "fixture-model", selection_reason: "original selection", continuation_of: "historical-parent", continuation_child_ids: ["historical-child"], scope_extensions: [{ paths: ["scenario/source.go"], reason: "original scope", recorded_at: "recorded", author: "declared actor" }], blocker_repair_policy: "investigate" } });
    const result = await f.service.get("historical-fixture");
    expect(result).toMatchObject({ executionId: "historical-fixture", status: "failed", mode: "manual", taskId: "historical-task", runId: "historical-run", failureReason: "original refusal", parentExecutionId: "historical-parent", fixupAttempt: 2, executionPreferences: { preferredRunner: "fixture-runner", model: "fixture-model", effort: "high" }, actualRunner: "fixture-runner", actualModel: "fixture-model", selectionReason: "original selection", continuationOf: "historical-parent", continuationChildIds: ["historical-child"], scopeExtensions: [{ paths: ["scenario/source.go"], reason: "original scope", recordedAt: "recorded", author: "declared actor" }], blockerRepairPolicy: "investigate" });
    expect(f.api.get).toHaveBeenCalledTimes(1); expect(f.api.get).toHaveBeenCalledWith("/execution/historical-fixture");
  });
  it("does not synthesize finalization, provider selection or continuation from a minimal record", async () => {
    const f = fixture({ execution }); const result = await f.service.get("historical-fixture");
    for (const field of ["finalization", "executionPreferences", "actualRunner", "actualModel", "selectionReason", "continuationOf", "continuationChildIds", "scopeExtensions", "blockerRepairPolicy"]) expect(result).not.toHaveProperty(field);
    expect(result.fixupAttempt).toBe(0); expect(f.api.get).toHaveBeenCalledTimes(1);
  });
  it("retains exact refused health/review results, warning details and dimensions", async () => {
    const f = fixture({ execution: { ...execution, finalization: {
      eligible: true, status: "failed", phase: "review", scope_source: "changed_files", skip_reason: "original skip", started_at: "started", completed_at: "finished", affected_scenarios: ["fixture-scenario"], aggregate_classification: "needs_work", aggregate_summary: "original summary",
      warnings: [{ code: "fixture-warning", scenario_name: "fixture-scenario", message: "original warning", retryable: true, created_at: "warning-created" }],
      scenarios: [{ scenario_name: "fixture-scenario", changed_paths: ["scenario/source.go"], restart: { status: "failed", attempts: 2, last_error: "original restart refusal", started_at: "restart-start", finished_at: "restart-finish" }, health: { status: "failed", scenario_status: "stopped", health_status: "unhealthy", schema_valid: false, details: "original health refusal", checked_at: "checked" }, review: { status: "failed", job_id: "historical-job", skip_reason: "original review skip", result: { job_id: "historical-job", classification: "needs_work", dimensions: [{ name: "correctness", status: "red", details: "original finding" }], summary: "original review summary", reviewed_at: "reviewed" } } }],
    } } });
    const result = await f.service.get("historical-fixture");
    expect(result.finalization).toEqual({ eligible: true, status: "failed", phase: "review", scopeSource: "changed_files", skipReason: "original skip", startedAt: "started", completedAt: "finished", affectedScenarios: ["fixture-scenario"], aggregateClassification: "needs_work", aggregateSummary: "original summary", warnings: [{ code: "fixture-warning", scenarioName: "fixture-scenario", message: "original warning", retryable: true, createdAt: "warning-created" }], scenarios: [{ scenarioName: "fixture-scenario", changedPaths: ["scenario/source.go"], restart: { status: "failed", attempts: 2, lastError: "original restart refusal", startedAt: "restart-start", finishedAt: "restart-finish" }, health: { status: "failed", scenarioStatus: "stopped", healthStatus: "unhealthy", schemaValid: false, details: "original health refusal", checkedAt: "checked" }, review: { status: "failed", jobId: "historical-job", skipReason: "original review skip", result: { jobId: "historical-job", classification: "needs_work", dimensions: [{ name: "correctness", status: "red", details: "original finding" }], summary: "original review summary", reviewedAt: "reviewed" } } }] });
    expect(f.api.get).toHaveBeenCalledTimes(1);
  });
  it("retains absent nested results without claiming successful restart, health or review", async () => {
    const f = fixture({ execution: { ...execution, finalization: { scenarios: [{ scenario_name: "fixture-scenario" }], warnings: [{}] } } });
    const finalization = (await f.service.get("historical-fixture")).finalization;
    expect(finalization).toMatchObject({ eligible: false, warnings: [{ code: "", message: "", retryable: false, createdAt: "" }], affectedScenarios: [], scenarios: [{ scenarioName: "fixture-scenario", changedPaths: [], restart: { status: "pending", attempts: 0 }, health: { status: "pending", schemaValid: false }, review: { status: "pending", result: undefined } }] });
    expect(f.api.get).toHaveBeenCalledTimes(1);
  });
  it("reads only the selected failed/manual history with exact query correlation", async () => {
    const f = fixture({ items: [execution] }); expect(await f.service.list({ status: "failed", mode: "manual", backlogKind: "execute", backlogName: "fixture item" })).toHaveLength(1);
    expect(f.api.get).toHaveBeenCalledTimes(1); expect(f.api.get).toHaveBeenCalledWith("/execution?status=failed&mode=manual&backlog_kind=execute&backlog_name=fixture+item");
  });
  it.each([{ execution: { ...execution, status: "unknown" } }, {}, { execution: { ...execution, mode: "unknown" } }])("refuses invalid/missing history without fallback or effects (%j)", async (data) => {
    vi.spyOn(console, "error").mockImplementation(() => {}); const f = fixture(data);
    await expect(f.service.get("historical-fixture")).rejects.toThrow("Invalid execution response"); expect(f.api.get).toHaveBeenCalledTimes(1);
  });
});
