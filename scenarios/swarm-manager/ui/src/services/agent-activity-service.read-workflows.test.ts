import { afterEach, describe, expect, it, vi } from "vitest";
import type { IApiClient } from "../lib/api-client";
import { createAgentActivityService } from "./agent-activity-service";

const clients: IApiClient[] = [];
function fixture(data: unknown) {
  const api: IApiClient = { get: vi.fn().mockResolvedValue(data), post: vi.fn(), put: vi.fn(), patch: vi.fn(), delete: vi.fn() };
  clients.push(api); return { api, service: createAgentActivityService(api) };
}
const activity = {
  activity_id: "fixture-activity", owner_type: "backlog", owner_kind: "execute", owner_name: "fixture-item", owner_title: "Original title",
  execution_id: "historical-execution", purpose: "review", interaction_type: "continue", task_id: "historical-task", run_id: "historical-run",
  status: "needs_review", requested_at: "requested", started_at: "started", finished_at: "finished", failure_reason: "original refusal",
  requested_by: "declared actor", metadata: { context: "preserved" }, updated_at: "updated",
};
afterEach(() => {
  for (const api of clients.splice(0)) {
    for (const method of ["post", "put", "patch", "delete"] as const) expect(api[method]).not.toHaveBeenCalled();
  }
  vi.restoreAllMocks();
});
describe("activity catalog reads (fictional history; no native activity)", () => {
  it("retains owner/execution correlation, refusal and annotation without treating them as caller proof", async () => {
    const f = fixture({ activity });
    expect(await f.service.get("fixture-activity")).toEqual({ activityId: "fixture-activity", ownerType: "backlog", ownerKind: "execute", ownerName: "fixture-item", ownerTitle: "Original title", executionId: "historical-execution", purpose: "review", interactionType: "continue", taskId: "historical-task", runId: "historical-run", status: "needs_review", requestedAt: "requested", startedAt: "started", finishedAt: "finished", failureReason: "original refusal", requestedBy: "declared actor", metadata: { context: "preserved" }, updatedAt: "updated" });
    expect(f.api.get).toHaveBeenCalledTimes(1); expect(f.api.get).toHaveBeenCalledWith("/agent-activities/fixture-activity");
  });
  it("preserves exact false-active filtering and selected owner in one read", async () => {
    const f = fixture({ items: [activity] });
    expect(await f.service.list({ ownerType: "backlog", ownerName: "item with space", executionId: "historical-execution", active: false })).toHaveLength(1);
    expect(f.api.get).toHaveBeenCalledTimes(1); expect(f.api.get).toHaveBeenCalledWith("/agent-activities?owner_type=backlog&owner_name=item+with+space&execution_id=historical-execution&active=false");
  });
  it("keeps omitted filters on the canonical catalog path and retains empty results", async () => {
    const f = fixture({ items: [] }); expect(await f.service.list()).toEqual([]);
    expect(f.api.get).toHaveBeenCalledTimes(1); expect(f.api.get).toHaveBeenCalledWith("/agent-activities");
  });
  it.each(["owner_type", "purpose", "interaction_type", "status"])("refuses unknown %s at the generated boundary without fallback reads", async (field) => {
    vi.spyOn(console, "error").mockImplementation(() => {});
    const f = fixture({ activity: { ...activity, [field]: "unknown" } });
    await expect(f.service.get("fixture-activity")).rejects.toThrow("Invalid agent activity response");
    expect(f.api.get).toHaveBeenCalledTimes(1); expect(f.api.get).toHaveBeenCalledWith("/agent-activities/fixture-activity");
  });
  it("refuses a missing activity rather than synthesizing historical owner data", async () => {
    const f = fixture({}); await expect(f.service.get("fixture-activity")).rejects.toThrow("Invalid agent activity response");
    expect(f.api.get).toHaveBeenCalledTimes(1);
  });
  it("propagates the exact catalog refusal without alternate reads or effects", async () => {
    const f = fixture({}); const refusal = new Error("catalog refused"); vi.mocked(f.api.get).mockRejectedValue(refusal);
    await expect(f.service.list({ active: true })).rejects.toBe(refusal);
    expect(f.api.get).toHaveBeenCalledTimes(1); expect(f.api.get).toHaveBeenCalledWith("/agent-activities?active=true");
  });
});
