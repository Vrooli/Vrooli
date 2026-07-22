import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { getComponentTestReport, listComponentTestReports, runComponentTest } from "./componentTests";

describe("api/componentTests", () => {
  const fetchSpy = vi.fn();

  beforeEach(() => vi.stubGlobal("fetch", fetchSpy));
  afterEach(() => {
    vi.unstubAllGlobals();
    fetchSpy.mockReset();
  });

  it("posts a test request and returns its report", async () => {
    fetchSpy.mockResolvedValueOnce(new Response(JSON.stringify({ report: { id: "report-1", verdict: "passed", results: [] } }), { status: 200 }));

    await expect(runComponentTest({ componentId: "asset-1", version: "1.0.0", includeClosure: true })).resolves.toMatchObject({ id: "report-1" });
    expect(fetchSpy).toHaveBeenCalledWith(expect.stringContaining("/RunComponentTest"), expect.objectContaining({ method: "POST" }));
  });

  it("returns a saved report and normalizes an empty report collection", async () => {
    fetchSpy.mockResolvedValueOnce(new Response(JSON.stringify({ report: { id: "report-2", verdict: "passed", results: [] } }), { status: 200 }));
    await expect(getComponentTestReport("report-2")).resolves.toMatchObject({ id: "report-2" });

    fetchSpy.mockResolvedValueOnce(new Response("{}", { status: 200 }));
    await expect(listComponentTestReports({ componentId: "asset-1" })).resolves.toEqual([]);
  });

  it("rejects responses that omit a required report", async () => {
    fetchSpy.mockResolvedValueOnce(new Response("{}", { status: 200 }));
    await expect(runComponentTest({ componentId: "asset-1", version: "1.0.0", includeClosure: false })).rejects.toThrow("no component test report");
  });
});
