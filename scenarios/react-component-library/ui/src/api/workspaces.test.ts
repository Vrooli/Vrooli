import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import {
  captureWorkspaceState,
  compareWorkspaceCaptureReceipts,
  listWorkspaceCaptureReceipts,
  listWorkspaceVisualComparisons,
} from "./workspaces";

describe("api/workspaces Connect operations", () => {
  const fetchSpy = vi.fn();

  beforeEach(() => vi.stubGlobal("fetch", fetchSpy));
  afterEach(() => {
    vi.unstubAllGlobals();
    fetchSpy.mockReset();
  });

  it("captures state through WorkspaceService while preserving attempt metadata", async () => {
    fetchSpy.mockResolvedValueOnce(
      new Response(
        JSON.stringify({
          workspaceRevision: 7,
          captureId: "capture-1",
          evidence: {
            schemaVersion: "rcl-workspace-state-capture/v1",
            requestToken: "token-1",
            attemptId: "attempt-1",
            sourceRevision: 6,
            pageId: "page-1",
            displayId: "display-1",
            stateId: "state-1",
            executionId: "execution-1",
            capturedAt: "2026-11-21T12:00:00Z",
            screenshot: { reference: "shot-1", width: 1280, height: 720, scope: "full" },
            accessibility: { reference: "a11y-1", scope: "full" },
          },
        }),
        { status: 200, headers: { "content-type": "application/json" } },
      ),
    );

    await expect(
      captureWorkspaceState("ws-1", {
        expectedRevision: 6,
        pageId: "page-1",
        displayId: "display-1",
        stateId: "state-1",
        attemptId: "attempt-1",
      }),
    ).resolves.toMatchObject({ captureId: "capture-1", workspaceRevision: 7 });
    expect(fetchSpy.mock.calls[0]?.[0]).toContain("CaptureWorkspaceState");

    fetchSpy.mockResolvedValueOnce(
      new Response(JSON.stringify({ code: "unavailable", message: "capture pending" }), {
        status: 503,
        headers: { "content-type": "application/json", "attempt-state": "in_progress" },
      }),
    );
    await expect(
      captureWorkspaceState("ws-1", {
        expectedRevision: 6,
        pageId: "page-1",
        displayId: "display-1",
        stateId: "state-1",
        attemptId: "attempt-1",
      }),
    ).rejects.toMatchObject({ attemptState: "in_progress" });
  });

  it("lists receipts and maps comparison result JSON from typed Connect clients", async () => {
    fetchSpy.mockResolvedValueOnce(
      new Response(JSON.stringify({ receipts: [] }), {
        status: 200,
        headers: { "content-type": "application/json" },
      }),
    );
    await expect(listWorkspaceCaptureReceipts("ws-1", "state-1")).resolves.toEqual([]);

    const result = {
      jobId: "job-1",
      verdict: "pass",
      dimensionsMatch: true,
      width: 10,
      height: 10,
      changedPixels: "0",
      totalPixels: "100",
      changedFraction: 0,
      mae: 0,
      rmse: 0,
      psnr: 100,
      phashDistance: 0,
      phashSimilarity: 1,
      ssim: 1,
      heatmapRef: "heatmap-1",
      warnings: [],
    };
    const comparison = {
      comparisonId: "comparison-1",
      workspaceId: "ws-1",
      stateId: "state-1",
      baselineReceiptId: "baseline-1",
      currentReceiptId: "current-1",
      createdAt: "2026-11-21T12:00:00Z",
      resultJson: JSON.stringify(result),
    };
    fetchSpy.mockResolvedValueOnce(
      new Response(JSON.stringify(comparison), {
        status: 200,
        headers: { "content-type": "application/json" },
      }),
    );
    await expect(
      compareWorkspaceCaptureReceipts("ws-1", "state-1", "baseline-1", "current-1"),
    ).resolves.toMatchObject({ comparisonId: "comparison-1", result });
    expect(fetchSpy.mock.calls[1]?.[0]).toContain("CompareWorkspaceCaptureReceipts");

    fetchSpy.mockResolvedValueOnce(
      new Response(JSON.stringify({ comparisons: [comparison] }), {
        status: 200,
        headers: { "content-type": "application/json" },
      }),
    );
    await expect(listWorkspaceVisualComparisons("ws-1", "state-1")).resolves.toMatchObject([
      { comparisonId: "comparison-1", result },
    ]);
  });
});
