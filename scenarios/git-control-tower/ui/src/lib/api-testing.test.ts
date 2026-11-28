import { beforeEach, describe, expect, it, vi } from "vitest";

const { client } = vi.hoisted(() => ({
  client: {
    getScore: vi.fn(),
    listIssues: vi.fn(),
    getStaleness: vi.fn(),
    triggerLightScan: vi.fn(),
    getScenarioDetail: vi.fn(),
  },
}));

vi.mock("./connect", () => ({ tidinessClient: client }));

import {
  fetchTidinessIssues,
  fetchTidinessScenarioDetail,
  fetchTidinessScore,
  fetchTidinessStaleness,
  triggerTidinessLightScan,
} from "./api-testing";

describe("typed tidiness API adapters", () => {
  beforeEach(() => vi.resetAllMocks());

  it("maps typed score, issue, staleness, scan, and detail results to existing UI shapes", async () => {
    client.getScore.mockResolvedValue({
      scenario: "demo", score: 91, violations: 2, lastScan: "2026-11-21T10:00:00Z",
      breakdown: { lintIssues: 1, typeIssues: 0, longFiles: 1, complexFunctions: 0, techDebtMarkers: 0, duplicationIssues: 0 },
      metrics: { totalFiles: 5, totalLines: 80, avgFileLength: 16, maxComplexity: 4, avgComplexity: 2, duplicationPct: 0 },
    });
    client.listIssues.mockResolvedValue({ issues: [{
      id: 9n, scenario: "demo", filePath: "a.go", category: "length", severity: "warning",
      title: "long", description: "too long", lineNumber: 4, columnNumber: 2,
      agentNotes: "owner note", remediationSteps: "split", status: "open", createdAt: "now",
    }] });
    client.getStaleness.mockResolvedValue({ isStale: true, modifiedFiles: 2, staleReason: "changed" });
    client.triggerLightScan.mockResolvedValue({
      scenario: "demo", startedAt: "start", completedAt: "end", durationMs: 1200n,
      fileMetrics: [{ path: "a.go", lines: 10, extension: "go" }],
      longFiles: [{ path: "b.go", lines: 901, threshold: 900 }], totalFiles: 2, totalLines: 911,
      lintIssues: 1, typeIssues: 2, longFilesCount: 1,
    });
    client.getScenarioDetail.mockResolvedValue({
      scenario: "demo", lightIssues: 1, aiIssues: 2, longFiles: 3,
      files: [{ path: "a.go", lines: 10, totalIssues: 3, visitCount: 4 }],
    });

    expect((await fetchTidinessScore("demo", "7")).breakdown?.lint_issues).toBe(1);
    expect(await fetchTidinessIssues("demo", "a.go", "length", "warning", 20, "7")).toMatchObject([
      { id: 9, file_path: "a.go", line_number: 4, agent_notes: "owner note" },
    ]);
    expect(await fetchTidinessStaleness("demo", "7")).toMatchObject({ is_stale: true, modified_files: 2 });
    expect(await triggerTidinessLightScan({ scenario_name: "demo", timeout_sec: 20, incremental: true }, "7"))
      .toMatchObject({ duration_ms: 1200, long_files_count: 1, long_files: [{ path: "b.go", threshold: 900 }] });
    expect(await fetchTidinessScenarioDetail("demo", "7")).toMatchObject({
      aiIssues: 2, files: [{ path: "a.go", totalIssues: 3, visitCount: 4 }],
    });
  });

  it("sends repository and filter context on typed requests", async () => {
    client.listIssues.mockResolvedValue({ issues: [] });
    await fetchTidinessIssues("demo", "x.go", "lint", "error", 25, "42");
    expect(client.listIssues).toHaveBeenCalledWith(expect.objectContaining({
      scenarioName: "demo", repositoryId: "42", file: "x.go", category: "lint", severity: "error", limit: 25,
    }));
    client.triggerLightScan.mockResolvedValue({ fileMetrics: [], longFiles: [] });
    await triggerTidinessLightScan({ scenario_name: "demo" }, "42");
    expect(client.triggerLightScan).toHaveBeenCalledWith(expect.objectContaining({
      repositoryId: "42", scenarioName: "demo", incremental: false,
    }));
  });
});
