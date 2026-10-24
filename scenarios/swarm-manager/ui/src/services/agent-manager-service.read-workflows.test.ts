import { afterEach, describe, expect, it, vi } from "vitest";
import { createAgentManagerService } from "./agent-manager-service";
import type { IApiClient } from "../lib/api-client";

const clients: IApiClient[] = [];
function reader(data: unknown) {
  const client: IApiClient = {
    get: vi.fn().mockResolvedValue(data), post: vi.fn(), put: vi.fn(), patch: vi.fn(), delete: vi.fn(),
  };
  clients.push(client);
  return { client, service: createAgentManagerService(client) };
}
afterEach(() => {
  for (const client of clients.splice(0)) {
    for (const method of ["post", "put", "patch", "delete"] as const) expect(client[method]).not.toHaveBeenCalled();
  }
});

describe("Agent Manager read projections (isolated transport; no live control)", () => {
  it.each(["pending", "starting", "running", "needs_review", "complete", "failed", "cancelled", "unspecified"])(
    "retains supported %s while normalizing wire whitespace/case", async (status) => {
      const { client, service } = reader({ run_id: "returned-run", task_id: "fixture-task", status: ` ${status.toUpperCase()} `, active: true });
      const result = await service.getRunState("requested-run");
      expect(client.get).toHaveBeenCalledTimes(1); expect(client.get).toHaveBeenCalledWith("/agent-manager/runs/requested-run");
      expect(result).toMatchObject({ runId: "returned-run", taskId: "fixture-task", status, active: true });
    },
  );
  it.each([undefined, "unrecognized", ""])("does not invent an active status for %s", async (status) => {
    const { service } = reader({ status });
    expect(await service.getRunState("fixture-run")).toMatchObject({ runId: "fixture-run", status: "unspecified", active: false });
  });
  it("retains finite duration and positive usage without changing timestamps or refusal text", async () => {
    const { service } = reader({ duration_seconds: 0, tokens_used: 12, turns_used: 2, cost_estimate: 0.25, changed_files: 3, context_tokens: 9, started_at: "start", finished_at: "finish", error_message: "original refusal" });
    expect(await service.getRunState("fixture-run")).toEqual({ runId: "fixture-run", taskId: undefined, status: "unspecified", active: false, startedAt: "start", finishedAt: "finish", errorMessage: "original refusal", durationSeconds: 0, tokensUsed: 12, turnsUsed: 2, costEstimate: 0.25, changedFiles: 3, contextTokens: 9 });
  });
  it.each([0, -1, "12", undefined])("does not display unsupported usage values (%s)", async (value) => {
    const { service } = reader({ tokens_used: value, turns_used: value, cost_estimate: value, changed_files: value, context_tokens: value });
    expect(await service.getRunState("fixture-run")).toMatchObject({ tokensUsed: undefined, turnsUsed: undefined, costEstimate: undefined, changedFiles: undefined, contextTokens: undefined });
  });
  it.each([Infinity, NaN, "3", undefined])("refuses non-finite/non-numeric duration (%s)", async (duration_seconds) => {
    expect((await reader({ duration_seconds }).service.getRunState("fixture-run")).durationSeconds).toBeUndefined();
  });
  it.each([[{ run: { sandboxId: "fixture-sandbox" } }, "fixture-sandbox"], [{ run: {} }, undefined], [{}, undefined], [{ run: { sandboxId: 42 } }, undefined]])("reads only typed sandbox detail (%j)", async (data, expected) => {
    const { client, service } = reader(data);
    expect(await service.getRunDetails("fixture-run")).toEqual({ sandboxId: expected });
    expect(client.get).toHaveBeenCalledTimes(1); expect(client.get).toHaveBeenCalledWith("/agent-manager/runs/fixture-run");
  });
  it("propagates the exact refused read without retry, writes or alternate lookup", async () => {
    const { client, service } = reader({}); const refusal = new Error("read refused");
    vi.mocked(client.get).mockRejectedValue(refusal);
    await expect(service.getRunState("fixture-run")).rejects.toBe(refusal);
    expect(client.get).toHaveBeenCalledTimes(1); expect(client.get).toHaveBeenCalledWith("/agent-manager/runs/fixture-run");
  });
});
