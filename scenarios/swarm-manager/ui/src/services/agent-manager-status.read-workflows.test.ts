import { afterEach, describe, expect, it, vi } from "vitest";
import type { IApiClient } from "../lib/api-client";
import { createAgentManagerService } from "./agent-manager-service";
const clients: IApiClient[] = [];
function fixture(data: unknown, error?: Error) { const api: IApiClient = { get: error ? vi.fn().mockRejectedValue(error) : vi.fn().mockResolvedValue(data), post: vi.fn(), put: vi.fn(), patch: vi.fn(), delete: vi.fn() }; clients.push(api); return { api, service: createAgentManagerService(api) }; }
afterEach(() => { try { for (const api of clients.splice(0)) { expect(api.get).toHaveBeenCalledTimes(1); expect(api.get).toHaveBeenCalledWith("/agent-manager/status"); for (const method of ["post", "put", "patch", "delete"] as const) expect(api[method]).not.toHaveBeenCalled(); } } finally { vi.restoreAllMocks(); } });
describe("actual manager status read compatibility without admission", () => {
 it.each([{ enabled: false, available: false }, { enabled: true, available: false }, { enabled: true, available: true }])("preserves exact availability flags %j", async data => { const f=fixture(data); expect(await f.service.getStatus()).toEqual({ $typeName: "vrooli.swarm_manager.v1.api.AgentManagerStatusResponse", ...data }); });
 it("preserves validated configured URL and profile correlation without treating them as identity", async () => { const f=fixture({ enabled: true, available: true, url: "https://fixture.example/status", profile_id: "00000000-0000-4000-8000-000000000001" }); expect(await f.service.getStatus()).toEqual({ $typeName: "vrooli.swarm_manager.v1.api.AgentManagerStatusResponse", enabled: true, available: true, url: "https://fixture.example/status", profileId: "00000000-0000-4000-8000-000000000001" }); });
 it.each([{ enabled: true, available: true, url: "not-a-uri" }, { enabled: true, available: true, profile_id: "anonymous" }])("refuses invalid configuration %j with no alternate route", async data => { const f=fixture(data); await expect(f.service.getStatus()).rejects.toThrow("Invalid agent-manager status response"); });
 it("retains transport refusal instead of converting it to available", async () => { const error=new Error("Fixture owner unavailable"); const f=fixture(undefined,error); await expect(f.service.getStatus()).rejects.toBe(error); });
});
