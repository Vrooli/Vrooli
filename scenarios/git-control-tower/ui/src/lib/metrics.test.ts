import { describe, it, expect } from "vitest";
import { aggregateFileStats, formatNetLines, densityLabel, categoryStats, getFileStats, filterFileStats, filterCategoryStats, fileExtension, isTestFile, churnLabel, formatFileTypeBreakdown } from "./metrics";
import type { RepoFileStats, DiffStats } from "./api";

describe("aggregateFileStats", () => {
  it("returns zeros for undefined input", () => {
    const result = aggregateFileStats(undefined);
    expect(result.totalAdditions).toBe(0);
    expect(result.totalDeletions).toBe(0);
    expect(result.totalNetLines).toBe(0);
    expect(result.totalFiles).toBe(0);
    expect(result.binaryCount).toBe(0);
    expect(result.renameCount).toBe(0);
  });

  it("aggregates across categories", () => {
    const fileStats: RepoFileStats = {
      staged: { "a.ts": { additions: 10, deletions: 3, files: 1, net_lines: 7 } },
      unstaged: { "b.ts": { additions: 5, deletions: 2, files: 1, net_lines: 3 } },
      untracked: { "c.ts": { additions: 8, deletions: 0, files: 1, net_lines: 8 } },
    };
    const result = aggregateFileStats(fileStats);
    expect(result.totalAdditions).toBe(23);
    expect(result.totalDeletions).toBe(5);
    expect(result.totalNetLines).toBe(18);
    expect(result.totalFiles).toBe(3);
  });

  it("deduplicates files appearing in multiple categories", () => {
    const fileStats: RepoFileStats = {
      staged: { "a.ts": { additions: 10, deletions: 3, files: 1, net_lines: 7 } },
      unstaged: { "a.ts": { additions: 5, deletions: 2, files: 1, net_lines: 3 } },
    };
    const result = aggregateFileStats(fileStats);
    // Should only count the file once (from staged, which is processed first)
    expect(result.totalFiles).toBe(1);
    expect(result.totalAdditions).toBe(10);
  });

  it("counts binary and rename files", () => {
    const fileStats: RepoFileStats = {
      staged: {
        "img.png": { additions: 0, deletions: 0, files: 1, is_binary: true },
        "renamed.ts": { additions: 2, deletions: 1, files: 1, is_rename: true, old_path: "old.ts" },
      },
    };
    const result = aggregateFileStats(fileStats);
    expect(result.binaryCount).toBe(1);
    expect(result.renameCount).toBe(1);
  });

  it("falls back to computing net_lines when field is missing", () => {
    const fileStats: RepoFileStats = {
      staged: { "a.ts": { additions: 10, deletions: 3, files: 1 } as DiffStats },
    };
    const result = aggregateFileStats(fileStats);
    expect(result.totalNetLines).toBe(7);
  });

  it("computes fileTypeBreakdown", () => {
    const fileStats: RepoFileStats = {
      staged: {
        "a.go": { additions: 10, deletions: 3, files: 1 },
        "b.go": { additions: 5, deletions: 2, files: 1 },
        "c.ts": { additions: 8, deletions: 0, files: 1 },
      },
    };
    const result = aggregateFileStats(fileStats);
    expect(result.fileTypeBreakdown).toEqual({ ".go": 2, ".ts": 1 });
  });

  it("computes testFileCount", () => {
    const fileStats: RepoFileStats = {
      staged: {
        "app.go": { additions: 10, deletions: 3, files: 1 },
        "app_test.go": { additions: 5, deletions: 2, files: 1 },
        "utils.test.ts": { additions: 8, deletions: 0, files: 1 },
      },
    };
    const result = aggregateFileStats(fileStats);
    expect(result.testFileCount).toBe(2);
  });

  it("computes churnRatio", () => {
    const fileStats: RepoFileStats = {
      staged: { "a.ts": { additions: 10, deletions: 10, files: 1 } },
    };
    const result = aggregateFileStats(fileStats);
    expect(result.churnRatio).toBe(1);
  });

  it("computes churnRatio as 0 when only additions", () => {
    const fileStats: RepoFileStats = {
      staged: { "a.ts": { additions: 10, deletions: 0, files: 1 } },
    };
    const result = aggregateFileStats(fileStats);
    expect(result.churnRatio).toBe(0);
  });

  it("computes paretoPercent for skewed distribution", () => {
    const fileStats: RepoFileStats = {
      staged: {
        "a.ts": { additions: 100, deletions: 0, files: 1 },
        "b.ts": { additions: 1, deletions: 0, files: 1 },
        "c.ts": { additions: 1, deletions: 0, files: 1 },
        "d.ts": { additions: 1, deletions: 0, files: 1 },
        "e.ts": { additions: 1, deletions: 0, files: 1 },
      },
    };
    const result = aggregateFileStats(fileStats);
    // Top 20% = 1 file, which has 100 out of 104 total = 96%
    expect(result.paretoTopN).toBe(1);
    expect(result.paretoPercent).toBe(96);
  });

  it("computes paretoPercent for even distribution", () => {
    const fileStats: RepoFileStats = {
      staged: {
        "a.ts": { additions: 10, deletions: 0, files: 1 },
        "b.ts": { additions: 10, deletions: 0, files: 1 },
        "c.ts": { additions: 10, deletions: 0, files: 1 },
        "d.ts": { additions: 10, deletions: 0, files: 1 },
        "e.ts": { additions: 10, deletions: 0, files: 1 },
      },
    };
    const result = aggregateFileStats(fileStats);
    expect(result.paretoTopN).toBe(1);
    expect(result.paretoPercent).toBe(20);
  });

  it("handles single file for pareto", () => {
    const fileStats: RepoFileStats = {
      staged: { "a.ts": { additions: 10, deletions: 5, files: 1 } },
    };
    const result = aggregateFileStats(fileStats);
    expect(result.paretoTopN).toBe(1);
    expect(result.paretoPercent).toBe(100);
  });
});

describe("fileExtension", () => {
  it("extracts .go", () => expect(fileExtension("main.go")).toBe(".go"));
  it("returns (no ext) for extensionless", () => expect(fileExtension("Makefile")).toBe("(no ext)"));
  it("extracts .test.ts as .ts", () => expect(fileExtension("foo.test.ts")).toBe(".ts"));
  it("handles hidden files", () => expect(fileExtension(".gitignore")).toBe("(no ext)"));
  it("handles paths with dirs", () => expect(fileExtension("src/lib/api.ts")).toBe(".ts"));
});

describe("isTestFile", () => {
  it("matches _test.go", () => expect(isTestFile("foo_test.go")).toBe(true));
  it("matches .test.tsx", () => expect(isTestFile("bar.test.tsx")).toBe(true));
  it("matches .spec.js", () => expect(isTestFile("baz.spec.js")).toBe(true));
  it("matches __tests__ path", () => expect(isTestFile("src/__tests__/baz.ts")).toBe(true));
  it("rejects regular file", () => expect(isTestFile("app.ts")).toBe(false));
  it("rejects test_utils.go", () => expect(isTestFile("test_utils.go")).toBe(false));
});

describe("churnLabel", () => {
  it("returns empty for 0", () => expect(churnLabel(0)).toBe(""));
  it("returns net change for 0.29", () => expect(churnLabel(0.29)).toBe("net change"));
  it("returns mixed for 0.3", () => expect(churnLabel(0.3)).toBe("mixed"));
  it("returns mixed for 0.69", () => expect(churnLabel(0.69)).toBe("mixed"));
  it("returns rewriting for 0.7", () => expect(churnLabel(0.7)).toBe("rewriting"));
  it("returns rewriting for 1.0", () => expect(churnLabel(1.0)).toBe("rewriting"));
});

describe("formatFileTypeBreakdown", () => {
  it("sorts by count descending", () => {
    const bd = { ".ts": 2, ".go": 3, ".css": 1 };
    expect(formatFileTypeBreakdown(bd)).toBe("3 .go, 2 .ts, 1 .css");
  });

  it("truncates with +N more", () => {
    const bd = { ".a": 1, ".b": 1, ".c": 1, ".d": 1, ".e": 1, ".f": 1, ".g": 1 };
    const result = formatFileTypeBreakdown(bd, 3);
    expect(result).toContain("(+4 more)");
  });

  it("returns empty string for empty breakdown", () => {
    expect(formatFileTypeBreakdown({})).toBe("");
  });
});

describe("formatNetLines", () => {
  it("formats positive", () => expect(formatNetLines(42)).toBe("+42"));
  it("formats negative", () => expect(formatNetLines(-7)).toBe("-7"));
  it("formats zero", () => expect(formatNetLines(0)).toBe("0"));
});

describe("densityLabel", () => {
  it("returns empty for zero", () => expect(densityLabel(0)).toBe(""));
  it("returns empty for undefined", () => expect(densityLabel(undefined)).toBe(""));
  it("returns focused for low density", () => expect(densityLabel(0.05)).toBe("focused"));
  it("returns moderate for medium density", () => expect(densityLabel(0.2)).toBe("moderate"));
  it("returns scattered for high density", () => expect(densityLabel(0.5)).toBe("scattered"));
});

describe("categoryStats", () => {
  it("returns zeros for undefined", () => {
    const result = categoryStats(undefined);
    expect(result.additions).toBe(0);
    expect(result.count).toBe(0);
  });

  it("sums stats across files", () => {
    const stats: Record<string, DiffStats> = {
      "a.ts": { additions: 10, deletions: 3, files: 1, net_lines: 7 },
      "b.ts": { additions: 5, deletions: 2, files: 1, net_lines: 3 },
    };
    const result = categoryStats(stats);
    expect(result.additions).toBe(15);
    expect(result.deletions).toBe(5);
    expect(result.netLines).toBe(10);
    expect(result.count).toBe(2);
  });
});

describe("getFileStats", () => {
  it("returns undefined for undefined fileStats", () => {
    expect(getFileStats("a.ts", undefined)).toBeUndefined();
  });

  it("returns undefined for missing file", () => {
    const fs: RepoFileStats = { staged: { "b.ts": { additions: 1, deletions: 0, files: 1 } } };
    expect(getFileStats("a.ts", fs)).toBeUndefined();
  });

  it("returns staged stats with priority over unstaged", () => {
    const stagedStats: DiffStats = { additions: 10, deletions: 3, files: 1, net_lines: 7 };
    const unstagedStats: DiffStats = { additions: 5, deletions: 2, files: 1, net_lines: 3 };
    const fs: RepoFileStats = { staged: { "a.ts": stagedStats }, unstaged: { "a.ts": unstagedStats } };
    expect(getFileStats("a.ts", fs)).toBe(stagedStats);
  });

  it("falls back to unstaged then untracked", () => {
    const untracked: DiffStats = { additions: 8, deletions: 0, files: 1, net_lines: 8 };
    const fs: RepoFileStats = { untracked: { "c.ts": untracked } };
    expect(getFileStats("c.ts", fs)).toBe(untracked);
  });
});

describe("filterFileStats", () => {
  const testFileStats: RepoFileStats = {
    staged: {
      "a.ts": { additions: 10, deletions: 3, files: 1, net_lines: 7 },
      "b.ts": { additions: 5, deletions: 2, files: 1, net_lines: 3 },
    },
    unstaged: {
      "c.ts": { additions: 8, deletions: 0, files: 1, net_lines: 8 },
    },
    untracked: {
      "d.ts": { additions: 4, deletions: 0, files: 1, net_lines: 4 },
    },
  };

  it("returns empty object for undefined fileStats", () => {
    expect(filterFileStats(["a.ts"], undefined)).toEqual({});
  });

  it("filters to matching paths only", () => {
    const result = filterFileStats(["a.ts", "c.ts"], testFileStats);
    expect(result.staged?.["a.ts"]).toBeDefined();
    expect(result.staged?.["b.ts"]).toBeUndefined();
    expect(result.unstaged?.["c.ts"]).toBeDefined();
    expect(result.untracked).toBeUndefined();
  });

  it("returns undefined for empty categories", () => {
    const result = filterFileStats(["x.ts"], testFileStats);
    expect(result.staged).toBeUndefined();
    expect(result.unstaged).toBeUndefined();
    expect(result.untracked).toBeUndefined();
  });
});

describe("filterCategoryStats", () => {
  const testFileStats: RepoFileStats = {
    staged: {
      "a.ts": { additions: 10, deletions: 3, files: 1, net_lines: 7 },
      "b.ts": { additions: 5, deletions: 2, files: 1, net_lines: 3 },
    },
    unstaged: {
      "c.ts": { additions: 8, deletions: 0, files: 1, net_lines: 8 },
    },
  };

  it("returns empty object for undefined fileStats", () => {
    expect(filterCategoryStats(["a.ts"], "staged", undefined)).toEqual({});
  });

  it("returns empty object for missing category", () => {
    expect(filterCategoryStats(["a.ts"], "untracked", testFileStats)).toEqual({});
  });

  it("filters single category to matching paths", () => {
    const result = filterCategoryStats(["a.ts"], "staged", testFileStats);
    expect(result.staged?.["a.ts"]).toBeDefined();
    expect(result.staged?.["b.ts"]).toBeUndefined();
    expect(result.unstaged).toBeUndefined();
  });

  it("returns empty object when no paths match", () => {
    expect(filterCategoryStats(["x.ts"], "staged", testFileStats)).toEqual({});
  });
});
