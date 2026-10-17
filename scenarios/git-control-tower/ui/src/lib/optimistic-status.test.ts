import { describe, expect, it } from "vitest";
import {
  applyStageOptimistic,
  applyUnstageOptimistic,
  applyDiscardOptimistic,
} from "./hooks-core";
import type { RepoStatus } from "./api";

function makeStatus(overrides?: Partial<RepoStatus["files"]>): RepoStatus {
  const files = {
    staged: [] as string[],
    unstaged: [] as string[],
    untracked: [] as string[],
    conflicts: [] as string[],
    statuses: {} as Record<string, string>,
    ...overrides,
  };
  return {
    repo_dir: "/repo",
    branch: { head: "main" },
    files,
    summary: {
      staged: files.staged.length,
      unstaged: files.unstaged.length,
      untracked: files.untracked.length,
      conflicts: files.conflicts.length,
    },
    author: {},
    timestamp: "now",
  };
}

// The server omits empty bucket arrays, so at runtime the cached status can have
// undefined staged/unstaged/untracked/conflicts despite the non-optional types.
function makeSparseStatus(files: Partial<RepoStatus["files"]>): RepoStatus {
  return {
    repo_dir: "/repo",
    branch: { head: "main" },
    files: files as RepoStatus["files"],
    summary: { staged: 0, unstaged: 0, untracked: 0, conflicts: 0 },
    author: {},
    timestamp: "now",
  };
}

describe("optimistic reducers tolerate omitted bucket arrays", () => {
  it("stages a file when the staged/conflicts buckets are absent", () => {
    const status = makeSparseStatus({ unstaged: ["a.ts"] });
    const next = applyStageOptimistic(status, ["a.ts"]);
    expect(next.files.staged).toEqual(["a.ts"]);
    expect(next.files.unstaged).toEqual([]);
    expect(next.summary.staged).toBe(1);
  });

  it("unstages a file when the unstaged/untracked buckets are absent", () => {
    const status = makeSparseStatus({ staged: ["a.ts"] });
    const next = applyUnstageOptimistic(status, ["a.ts"]);
    expect(next.files.staged).toEqual([]);
    expect(next.files.unstaged).toEqual(["a.ts"]);
  });

  it("discards a file when other buckets are absent", () => {
    const status = makeSparseStatus({ untracked: ["new.ts"] });
    const next = applyDiscardOptimistic(status, ["new.ts"]);
    expect(next.files.untracked).toEqual([]);
    expect(next.summary.untracked).toBe(0);
  });
});

describe("applyStageOptimistic", () => {
  it("moves an unstaged file into staged and updates summary", () => {
    const status = makeStatus({ unstaged: ["a.ts", "b.ts"], staged: ["z.ts"] });
    const next = applyStageOptimistic(status, ["a.ts"]);
    expect(next.files.staged).toEqual(["z.ts", "a.ts"]);
    expect(next.files.unstaged).toEqual(["b.ts"]);
    expect(next.summary.staged).toBe(2);
    expect(next.summary.unstaged).toBe(1);
  });

  it("moves an untracked file into staged", () => {
    const status = makeStatus({ untracked: ["new.ts"] });
    const next = applyStageOptimistic(status, ["new.ts"]);
    expect(next.files.staged).toEqual(["new.ts"]);
    expect(next.files.untracked).toEqual([]);
    expect(next.summary.untracked).toBe(0);
  });

  it("is additive for mixed hunk state (file already staged) without duplicating", () => {
    const status = makeStatus({ staged: ["mixed.ts"], unstaged: ["mixed.ts"] });
    const next = applyStageOptimistic(status, ["mixed.ts"]);
    expect(next.files.staged).toEqual(["mixed.ts"]);
    expect(next.files.unstaged).toEqual([]);
  });

  it("does not mutate the input status", () => {
    const status = makeStatus({ unstaged: ["a.ts"] });
    applyStageOptimistic(status, ["a.ts"]);
    expect(status.files.unstaged).toEqual(["a.ts"]);
    expect(status.files.staged).toEqual([]);
  });
});

describe("applyUnstageOptimistic", () => {
  it("moves a modified staged file back to unstaged", () => {
    const status = makeStatus({ staged: ["a.ts"], statuses: { "a.ts": "M." } });
    const next = applyUnstageOptimistic(status, ["a.ts"]);
    expect(next.files.staged).toEqual([]);
    expect(next.files.unstaged).toEqual(["a.ts"]);
    expect(next.files.untracked).toEqual([]);
  });

  it("returns a staged-new file (index status A) to untracked", () => {
    const status = makeStatus({ staged: ["new.ts"], statuses: { "new.ts": "A." } });
    const next = applyUnstageOptimistic(status, ["new.ts"]);
    expect(next.files.staged).toEqual([]);
    expect(next.files.untracked).toEqual(["new.ts"]);
    expect(next.files.unstaged).toEqual([]);
  });

  it("keeps summary counts consistent", () => {
    const status = makeStatus({ staged: ["a.ts", "b.ts"], statuses: { "a.ts": "M." } });
    const next = applyUnstageOptimistic(status, ["a.ts"]);
    expect(next.summary.staged).toBe(1);
    expect(next.summary.unstaged).toBe(1);
  });
});

describe("applyDiscardOptimistic", () => {
  it("removes discarded paths from worktree buckets", () => {
    const status = makeStatus({
      unstaged: ["a.ts"],
      untracked: ["new.ts"],
      conflicts: ["c.ts"],
    });
    const next = applyDiscardOptimistic(status, ["a.ts", "new.ts", "c.ts"]);
    expect(next.files.unstaged).toEqual([]);
    expect(next.files.untracked).toEqual([]);
    expect(next.files.conflicts).toEqual([]);
    expect(next.summary.unstaged).toBe(0);
    expect(next.summary.untracked).toBe(0);
  });

  it("leaves a still-staged file staged when its worktree change is discarded", () => {
    const status = makeStatus({ staged: ["a.ts"], unstaged: ["a.ts"] });
    const next = applyDiscardOptimistic(status, ["a.ts"]);
    expect(next.files.staged).toEqual(["a.ts"]);
    expect(next.files.unstaged).toEqual([]);
  });
});
