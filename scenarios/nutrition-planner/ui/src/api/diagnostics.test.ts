import { afterEach, describe, expect, it, vi } from "vitest";

import { fetchDiagnostics } from "./diagnostics";

describe("diagnostics API", () => {
  afterEach(() => vi.unstubAllGlobals());

  it("fetches a workspace-scoped report", async () => {
    const report = { workspaceId: "w1", generatedAt: "2026-09-18T00:00:00Z", schemaVersion: 0, databaseOk: true, providers: [], findings: [] };
    const fetchMock = vi.fn().mockResolvedValue(new Response(JSON.stringify(report), { status: 200 }));
    vi.stubGlobal("fetch", fetchMock);
    await expect(fetchDiagnostics("workspace/1")).resolves.toEqual(report);
    expect(fetchMock).toHaveBeenCalledWith(expect.stringContaining("workspace_id=workspace%2F1"), expect.objectContaining({ method: "GET", cache: "no-store" }));
  });

  it("decodes a failed diagnostics response", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(JSON.stringify({ code: "permission_denied", message: "not yours" }), { status: 403 })));
    await expect(fetchDiagnostics("w1")).rejects.toThrow("permission_denied: not yours");
  });
});
