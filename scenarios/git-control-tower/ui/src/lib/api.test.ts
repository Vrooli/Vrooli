import { fetchBlame, fetchCapabilities, fetchDiff, fetchHealth, fetchBranches, createBranch, fetchGroupingRules, fetchRepoGroups, fetchRepoHistory, fetchRepoStatus, fetchPrecommitConfig, savePrecommitConfig, pushToRemote, pullFromRemote, RemoteOperationError } from "./api";
import { mockFetchJson, textResponse } from "../test-utils";

const connectMocks = vi.hoisted(() => ({
  integrationClient: { getCapabilities: vi.fn(), getCollaborationStatus: vi.fn() },
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
    getGroupingRules: vi.fn(), getRepoHistory: vi.fn(), pushToRemote: vi.fn(), pullFromRemote: vi.fn(),
    getPrecommitConfig: vi.fn(), savePrecommitConfig: vi.fn(),
    getBlame: vi.fn(),
  },
}));

vi.mock("./connect", () => connectMocks);

// [REQ:GCT-OT-P0-001] Health check endpoint

vi.mock("@vrooli/api-base", () => ({
  resolveApiBase: () => "http://localhost:18700/api/v1",
  buildApiUrl: (path: string, opts: { baseUrl: string }) => `${opts.baseUrl}${path}`
}));

test("fetchBlame uses RepoService and preserves enriched evidence", async () => {
  connectMocks.repoClient.getBlame.mockResolvedValue({
    revision: "abc", truncated: false, warnings: [],
    files: [{
      path: "src/main.go", status: "attributed", contentDigest: "sha256:abc", reason: "",
      standing: "verified", downgradeReasons: [],
      lines: [{ line: 1, content: "package main", commit: "abc", author: "A", authorTime: "today", subject: "initial" }],
      evidence: [{
        runId: "run-1", sandboxId: "sandbox-1", applicationReceipt: "receipt", contentDigest: "sha256:abc",
        commitId: "abc", visibility: "public", commitState: "present", runOutcome: "success",
        conversationId: "conversation-1", costUsd: 0.2, committedAt: "today", unavailable: [],
        workReferences: [{ kind: "task", id: "task-1", revision: "r1", relationship: "caused", verified: true, visibility: "public", state: "active", unavailableReason: "" }],
      }],
    }],
    changeBundles: [{ runId: "run-1", sandboxId: "sandbox-1", files: ["src/main.go"], runOutcome: "success", conversationId: "conversation-1", costUsd: 0.2, gaps: [], workReferences: [] }],
  });

  const result = await fetchBlame(["src/main.go"], "repo-1");
  expect(connectMocks.repoClient.getBlame).toHaveBeenCalledOnce();
  expect(connectMocks.repoClient.getBlame.mock.calls[0]?.[0]).toMatchObject({ repositoryId: "repo-1", paths: ["src/main.go"], maxPaths: 1, maxLines: 300, maxBytes: 1 << 20, enrich: true });
  expect(result.files[0]?.evidence[0]?.workReferences?.[0]?.id).toBe("task-1");
  expect(result.changeBundles[0]?.files).toEqual(["src/main.go"]);
});

test("fetchHealth returns parsed JSON on success", async () => {
  mockFetchJson({ status: "healthy", service: "x", timestamp: "t" });

  const result = await fetchHealth();
  expect(result.status).toBe("healthy");
});

test("fetchCapabilities uses the generated IntegrationService client", async () => {
  connectMocks.integrationClient.getCapabilities.mockResolvedValue({
    capabilities: [{ id: "test-genie", name: "Test Genie", status: "available", uses: [{ id: "tests", name: "Tests", description: "Execute", group: "quality" }] }],
    timestamp: "2026-05-01T00:00:00Z",
  });

  const result = await fetchCapabilities();
  expect(connectMocks.integrationClient.getCapabilities).toHaveBeenCalledOnce();
  expect(result.capabilities[0]?.id).toBe("test-genie");
  expect(result.capabilities[0]?.uses?.[0]?.name).toBe("Tests");
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

test("fetchRepoStatus aggregates every page from one snapshot", async () => {
  connectMocks.repoClient.getRepoStatus
    .mockResolvedValueOnce({
      repoDir:"/repo", branchStatus:{head:"main"},
      files:{staged:["a"],unstaged:[],untracked:[],conflicts:[],ignored:[],binary:["a"],statuses:{a:"M"},renames:{}},
      fileStats:{staged:{},unstaged:{},untracked:{}},fileHotspots:{},scopes:{},summary:{staged:2,unstaged:0,untracked:0,conflicts:0,ignored:0},
      page:{returnedEntries:1,totalEntries:2,snapshotId:"snap",nextPageToken:"cursor",complete:false},
    })
    .mockResolvedValueOnce({
      repoDir:"/repo",files:{staged:["b"],unstaged:[],untracked:[],conflicts:[],ignored:[],binary:["b"],statuses:{b:"A"},renames:{}},
      fileStats:{staged:{},unstaged:{},untracked:{}},fileHotspots:{},scopes:{},summary:{staged:2,unstaged:0,untracked:0,conflicts:0,ignored:0},
      page:{returnedEntries:1,totalEntries:2,snapshotId:"snap",nextPageToken:"",complete:true},
    });
  const result=await fetchRepoStatus("repo-1");
  expect(result.files.staged).toEqual(["a","b"]);
  expect(result.files.binary).toEqual(["a","b"]);
  expect(result.files.statuses).toEqual({a:"M",b:"A"});
  expect(result.summary.staged).toBe(2);
  expect(connectMocks.repoClient.getRepoStatus).toHaveBeenCalledTimes(2);
  expect(connectMocks.repoClient.getRepoStatus.mock.calls[1]?.[0]).toMatchObject({pageToken:"cursor",pageSize:512});
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

test("precommit declared inputs roundtrip through the settings API", async () => {
  const declaredInputs = [".pre-commit-config.yaml", "scripts/check.sh"];
  connectMocks.repoClient.getPrecommitConfig.mockResolvedValue({
    enabled: true, command: "./scripts/check.sh", workingDirectory: "/repo", timeoutSeconds: 30,
    runBeforeCommit: true, allowOverride: false, declaredInputs,
  });
  const loaded = await fetchPrecommitConfig("repo-1");
  expect(loaded.declared_inputs).toEqual(declaredInputs);
  expect(connectMocks.repoClient.getPrecommitConfig).toHaveBeenCalledOnce();

  connectMocks.humanControlClient.getAuthorityStatus.mockResolvedValue({ canMutate: true, reason: "" });
  connectMocks.humanControlClient.prepareMutation.mockResolvedValue({
    repositoryId: "repo-1", operation: "repo.precommit", expectedRevision: "rev-1", subjectDigest: "digest-1",
  });
  connectMocks.humanControlClient.confirmMutation.mockResolvedValue({ intentId: "intent-1" });
  connectMocks.repoClient.savePrecommitConfig.mockResolvedValue({
    enabled: true, command: "./scripts/check.sh", workingDirectory: "/repo", timeoutSeconds: 30,
    runBeforeCommit: true, allowOverride: false, declaredInputs,
  });
  const saved = await savePrecommitConfig({
    enabled: true, command: "./scripts/check.sh", working_directory: "/repo", timeout_seconds: 30,
    run_before_commit: true, allow_override: false, declared_inputs: declaredInputs,
  }, "repo-1");
  const saveRequest = connectMocks.repoClient.savePrecommitConfig.mock.calls[0]?.[0] as { declaredInputs?: string[] } | undefined;
  expect(saveRequest?.declaredInputs).toEqual(declaredInputs);
  expect(saved.declared_inputs).toEqual(declaredInputs);
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

// A push that git rejected comes back as a 200 with success=false. Resolving it would
// make a failed push indistinguishable from a successful one at every call site.
test("pushToRemote raises the git failure instead of resolving", async () => {
  connectMocks.humanControlClient.getAuthorityStatus.mockResolvedValue({ canMutate: true });
  connectMocks.humanControlClient.prepareMutation.mockResolvedValue({
    repositoryId: "repo-1", operation: "repo.push", expectedRevision: "rev", subjectDigest: "digest",
  });
  connectMocks.humanControlClient.confirmMutation.mockResolvedValue({ intentId: "intent-1" });
  connectMocks.repoClient.pushToRemote.mockResolvedValue({
    success: false, remote: "origin", branch: "agi", pushed: false, upToDate: false,
    verified: false, verificationError: "", error: "git push timed out before the transfer finished",
    timestamp: "t",
  });

  await expect(pushToRemote()).rejects.toThrow(/timed out before the transfer finished/);
  await expect(pushToRemote()).rejects.toBeInstanceOf(RemoteOperationError);
});

test("pushToRemote resolves with the verification fields on success", async () => {
  connectMocks.humanControlClient.getAuthorityStatus.mockResolvedValue({ canMutate: true });
  connectMocks.humanControlClient.prepareMutation.mockResolvedValue({
    repositoryId: "repo-1", operation: "repo.push", expectedRevision: "rev", subjectDigest: "digest",
  });
  connectMocks.humanControlClient.confirmMutation.mockResolvedValue({ intentId: "intent-1" });
  connectMocks.repoClient.pushToRemote.mockResolvedValue({
    success: true, remote: "origin", branch: "agi", pushed: true, upToDate: false,
    verified: true, verificationError: "", error: "", timestamp: "t",
  });

  const result = await pushToRemote();
  expect(result.pushed).toBe(true);
  expect(result.verified).toBe(true);
});

test("pullFromRemote raises the git failure instead of resolving", async () => {
  connectMocks.humanControlClient.getAuthorityStatus.mockResolvedValue({ canMutate: true });
  connectMocks.humanControlClient.prepareMutation.mockResolvedValue({
    repositoryId: "repo-1", operation: "repo.pull", expectedRevision: "rev", subjectDigest: "digest",
  });
  connectMocks.humanControlClient.confirmMutation.mockResolvedValue({ intentId: "intent-1" });
  connectMocks.repoClient.pullFromRemote.mockResolvedValue({
    success: false, remote: "origin", branch: "agi", error: "CONFLICT (content): merge conflict",
    hasConflicts: true, timestamp: "t",
  });

  await expect(pullFromRemote()).rejects.toThrow(/merge conflict/);
});
