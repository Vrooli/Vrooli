import { fetchDiff, fetchHealth, fetchBranches, createBranch, fetchGroupingRules, fetchRepoGroups, fetchRepoHistory, fetchRepoStatus } from "./api";
import { mockFetchJson, textResponse } from "../test-utils";

const connectMocks = vi.hoisted(() => ({
  branchClient: {
    listBranches: vi.fn(),
    createBranch: vi.fn(),
    switchBranch: vi.fn(),
    publishBranch: vi.fn(),
  },
  humanControlClient: {
    getAuthorityStatus: vi.fn(),
    prepareMutation: vi.fn(),
    confirmMutation: vi.fn(),
  },
  repoClient: {
    createCommit: vi.fn(), getRepoDiff: vi.fn(), getRepoGroups: vi.fn(), getRepoStatus: vi.fn(), getSyncStatus: vi.fn(),
    getGroupingRules: vi.fn(), getRepoHistory: vi.fn(),
  },
}));

vi.mock("./connect", () => connectMocks);

// [REQ:GCT-OT-P0-001] Health check endpoint

vi.mock("@vrooli/api-base", () => ({
  resolveApiBase: () => "http://localhost:18700/api/v1",
  buildApiUrl: (path: string, opts: { baseUrl: string }) => `${opts.baseUrl}${path}`
}));

test("fetchHealth returns parsed JSON on success", async () => {
  mockFetchJson({ status: "healthy", service: "x", timestamp: "t" });

  const result = await fetchHealth();
  expect(result.status).toBe("healthy");
});

test("fetchHealth throws when API returns non-200", async () => {
  globalThis.fetch = vi.fn(async () => {
    return textResponse("Service unavailable", { status: 503 });
  }) as unknown as typeof fetch;

  await expect(fetchHealth()).rejects.toThrow(/Service unavailable|Request failed: 503/);
});

test("fetchBranches returns parsed JSON on success", async () => {
  connectMocks.branchClient.listBranches.mockResolvedValue({ current: "main", locals: [], remotes: [], timestamp: undefined });

  const result = await fetchBranches();
  expect(result.current).toBe("main");
});

test("fetchRepoStatus uses the typed RepoService response", async () => {
  connectMocks.repoClient.getRepoStatus.mockResolvedValue({
    repoDir: "/repo",
    branch: "main",
    branchStatus: { head: "main", upstream: "origin/main", ahead: 1, behind: 2, oid: "abc" },
    files: { staged: ["staged.ts"], unstaged: ["unstaged.ts"], untracked: [], conflicts: [], binary: [], ignored: [], statuses: {}, renames: {} },
    fileHotspots: {},
    scopes: {},
    summary: { staged: 1, unstaged: 1, untracked: 0, conflicts: 0, ignored: 0 },
    author: { name: "Operator", email: "operator@example.test" },
    timestamp: "2026-09-06T00:00:00Z",
  });

  const result = await fetchRepoStatus("repo-1");
  expect(result.repo_dir).toBe("/repo");
  expect(result.branch.upstream).toBe("origin/main");
  expect(result.summary.staged).toBe(1);
  expect(connectMocks.repoClient.getRepoStatus).toHaveBeenCalledOnce();
});

test("fetchDiff uses the typed RepoService response", async () => {
  connectMocks.repoClient.getRepoDiff.mockResolvedValue({
    repoDir: "/repo",
    path: "api/main.go",
    staged: true,
    untracked: false,
    hasDiff: true,
    hunks: [{ oldStart: 1, oldCount: 1, newStart: 1, newCount: 2, header: "@@", lines: ["+line"] }],
    stats: { additions: 1, deletions: 0, files: 1, netLines: 1, hunkCount: 1, largestHunk: 1 },
    raw: "@@",
    annotatedLines: [{ number: 1, content: "line", change: "added", oldNumber: 0 }],
    mode: "diff",
    timestamp: "2026-09-06T00:00:00Z",
  });

  const result = await fetchDiff("api/main.go", true, false, undefined, "diff", false, "repo-1");
  expect(result.path).toBe("api/main.go");
  expect(result.staged).toBe(true);
  expect(result.stats.additions).toBe(1);
  expect(result.hunks?.[0]?.old_lines).toBe(1);
  expect(connectMocks.repoClient.getRepoDiff).toHaveBeenCalledOnce();
});

test("createBranch returns parsed JSON on success", async () => {
  connectMocks.humanControlClient.getAuthorityStatus.mockResolvedValue({ canMutate: true, reason: "" });
  connectMocks.humanControlClient.prepareMutation.mockResolvedValue({
    repositoryId: "repo-1", operation: "repo.branch.create", expectedRevision: "rev-1", subjectDigest: "digest-1",
  });
  connectMocks.humanControlClient.confirmMutation.mockResolvedValue({ intentId: "intent-1" });
  connectMocks.branchClient.createBranch.mockResolvedValue({ success: true, branch: { name: "feature/test" }, timestamp: undefined });

  const result = await createBranch({ name: "feature/test" });
  expect(result.success).toBe(true);
  expect(result.branch?.name).toBe("feature/test");
});

test("fetchGroupingRules uses the typed RepoService response", async () => {
  connectMocks.repoClient.getGroupingRules.mockResolvedValue({ enabled: true, rules: [] });
  await fetchGroupingRules("repo-1");
  expect(connectMocks.repoClient.getGroupingRules).toHaveBeenCalledOnce();
});

test("fetchRepoGroups uses the typed RepoService response", async () => {
  connectMocks.repoClient.getRepoGroups.mockResolvedValue({ groups: [{ key: "scenario:x", label: "x", source: "contract", files: ["x/a.go"] }] });

  const result = await fetchRepoGroups("repo-1");
  expect(result.groups[0]?.key).toBe("scenario:x");
  expect(connectMocks.repoClient.getRepoGroups).toHaveBeenCalledOnce();
});

test("fetchRepoHistory requests files and checks includes together", async () => {
  connectMocks.repoClient.getRepoHistory.mockResolvedValue({
    repoDir: "/repo", lines: [], entries: [], limit: 30, timestamp: "t", grepPattern: "",
  });

  const result = await fetchRepoHistory(30, true, "repo-1", undefined, true);
  expect(result.repo_dir).toBe("/repo");
  expect(connectMocks.repoClient.getRepoHistory).toHaveBeenCalledOnce();
  expect(connectMocks.repoClient.getRepoHistory.mock.calls[0]?.[0]).toMatchObject({
    repositoryId: "repo-1", includeFiles: true, includeChecks: true,
  });
});
