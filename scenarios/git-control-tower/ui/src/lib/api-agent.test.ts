import { beforeEach, describe, expect, it, vi } from "vitest";

const { client } = vi.hoisted(() => ({
  client: {
    listProfiles: vi.fn(),
    listRuns: vi.fn(),
    getRun: vi.fn(),
    getRunEvents: vi.fn(),
    getRunDiff: vi.fn(),
  },
}));

vi.mock("./connect", () => ({ agentReadClient: client }));

import {
  fetchAgentProfiles,
  fetchAgentRuns,
  fetchAgentRun,
  fetchAgentRunEvents,
  fetchAgentRunDiff,
} from "./api-agent";

describe("typed Agent Manager read adapters", () => {
  beforeEach(() => vi.resetAllMocks());

  it("maps profile, run, event, and diff responses into current UI types", async () => {
    client.listProfiles.mockResolvedValue({ profiles: [{ id: "p1", key: "reviewer", name: "Reviewer", description: "review", roleRef: "code.review" }], total: 1 });
    const run = {
      id: "r1", taskId: "t1", sessionId: "s1", status: "complete", phase: "done", progressPercent: 100,
      errorMsg: "", approvalState: "approved", promptPreview: "inspect", sandboxId: "sb1", commitHash: "abc",
      createdAt: "created", startedAt: "started", endedAt: "ended",
      summary: { filesModified: ["main.go"], filesCreated: [], filesDeleted: [], tokensUsed: 123n, turnsUsed: 4, costEstimate: 0.25 },
      actions: { canInvestigate: false, canApplyInvestigation: false, canDelete: false, canStop: false, canRetry: false, canContinue: false, canApprove: false, canReject: false, canReview: true, canExtractRecommendations: false, canRegenerateRecommendations: false },
    };
    client.listRuns.mockResolvedValue({ runs: [run], total: 1 });
    client.getRun.mockResolvedValue(run);
    client.getRunEvents.mockResolvedValue({ events: [{ id: "e1", runId: "r1", sequence: 7n, eventType: "message", timestamp: "now", data: { role: "assistant", content: "hello" } }] });
    client.getRunDiff.mockResolvedValue({ runId: "r1", content: "diff --git", files: [{ path: "main.go", changeType: "modified", additions: 3, deletions: 1, isBinary: false, patch: "+x" }] });

    expect(await fetchAgentProfiles("42")).toMatchObject({ profiles: [{ id: "p1", key: "reviewer", roleRef: "code.review" }], total: 1 });
    expect(await fetchAgentRuns("demo", 10, "42")).toMatchObject({ runs: [{ id: "r1", status: "complete", summary: { tokensUsed: 123, filesModified: ["main.go"] }, actions: { canReview: true } }], total: 1 });
    expect(await fetchAgentRun("r1", "demo", "42")).toMatchObject({ id: "r1", taskId: "t1", createdAt: "created" });
    expect(await fetchAgentRunEvents("r1", "demo", 5, "42")).toMatchObject({ events: [{ id: "e1", sequence: 7, eventType: "message", data: { role: "assistant", content: "hello" } }] });
    expect(await fetchAgentRunDiff("r1", "demo", "42")).toMatchObject({ runId: "r1", files: [{ path: "main.go", additions: 3, patch: "+x" }] });
  });

  it("sends scenario, bounds, repository, and cursor in typed requests", async () => {
    client.listRuns.mockResolvedValue({ runs: [], total: 0 });
    await fetchAgentRuns("git-control-tower", 8, "42");
    expect(client.listRuns).toHaveBeenCalledWith(expect.objectContaining({ scenarioSlug: "git-control-tower", limit: 8, repositoryId: "42" }));
    client.getRunEvents.mockResolvedValue({ events: [] });
    await fetchAgentRunEvents("r1", "demo", 11, "42");
    expect(client.getRunEvents).toHaveBeenCalledWith(expect.objectContaining({ runId: "r1", scenarioSlug: "demo", afterSequence: 11n, limit: 100, repositoryId: "42" }));
  });
});
