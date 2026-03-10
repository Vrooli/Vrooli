import type { DiffStats, RepoFileStats } from "./api";

export interface AggregateMetrics {
  totalAdditions: number;
  totalDeletions: number;
  totalNetLines: number;
  totalFiles: number;
  binaryCount: number;
  renameCount: number;
  fileTypeBreakdown: Record<string, number>;
  churnRatio: number;
  paretoPercent: number;
  paretoTopN: number;
  testFileCount: number;
  testAdditions: number;
  testDeletions: number;
  testToCodeRatio: number;
  newFileCount: number;
  deletedFileCount: number;
}

/** Extract lowercase file extension, or "(no ext)" for extensionless files. */
export function fileExtension(path: string): string {
  const dot = path.lastIndexOf(".");
  if (dot === -1 || dot === path.length - 1) return "(no ext)";
  // Avoid treating hidden files like ".gitignore" as having extension "gitignore"
  const slash = Math.max(path.lastIndexOf("/"), path.lastIndexOf("\\"));
  if (dot <= slash + 1) return "(no ext)";
  return path.slice(dot).toLowerCase();
}

/** Check if a file path looks like a test file. */
export function isTestFile(path: string): boolean {
  if (path.endsWith("_test.go")) return true;
  if (/\.(?:test|spec)\.(?:ts|tsx|js|jsx)$/.test(path)) return true;
  if (path.includes("/__tests__/")) return true;
  return false;
}

/** Human-readable label for churn ratio. */
export function churnLabel(ratio: number): string {
  if (ratio >= 0.7) return "rewriting";
  if (ratio >= 0.3) return "mixed";
  if (ratio > 0) return "net change";
  return "";
}

/** Per-file churn ratio: min(add,del)/max(add,del). 0 = pure growth/deletion, 1 = pure rewriting. */
export function fileChurnRatio(stats: DiffStats): number {
  const add = stats.additions ?? 0;
  const del = stats.deletions ?? 0;
  const maxVal = Math.max(add, del);
  if (maxVal === 0) return 0;
  return Math.min(add, del) / maxVal;
}

/** Average lines changed per hunk. */
export function avgLinesPerHunk(stats: DiffStats): number {
  const hunks = stats.hunk_count ?? 0;
  if (hunks === 0) return 0;
  return Math.round(((stats.additions ?? 0) + (stats.deletions ?? 0)) / hunks);
}

/** Change risk heuristic: hunk_count * largest_hunk. Higher = needs more review attention. */
export function changeRiskScore(stats: DiffStats): number {
  return (stats.hunk_count ?? 0) * (stats.largest_hunk ?? 0);
}

/** Human label for risk score. */
export function riskLabel(score: number): string {
  if (score <= 0) return "";
  if (score < 50) return "low";
  if (score < 200) return "moderate";
  return "high";
}

/** Format file type breakdown as a compact string. */
export function formatFileTypeBreakdown(breakdown: Record<string, number>, maxShow = 5): string {
  const entries = Object.entries(breakdown).sort((a, b) => b[1] - a[1]);
  if (entries.length === 0) return "";
  const shown = entries.slice(0, maxShow).map(([ext, count]) => `${count} ${ext}`);
  const remaining = entries.length - maxShow;
  if (remaining > 0) shown.push(`(+${remaining} more)`);
  return shown.join(", ");
}

/** Aggregate per-file DiffStats across all categories into summary metrics. */
export function aggregateFileStats(fileStats?: RepoFileStats): AggregateMetrics {
  const result: AggregateMetrics = {
    totalAdditions: 0,
    totalDeletions: 0,
    totalNetLines: 0,
    totalFiles: 0,
    binaryCount: 0,
    renameCount: 0,
    fileTypeBreakdown: {},
    churnRatio: 0,
    paretoPercent: 0,
    paretoTopN: 0,
    testFileCount: 0,
    testAdditions: 0,
    testDeletions: 0,
    testToCodeRatio: 0,
    newFileCount: 0,
    deletedFileCount: 0,
  };
  if (!fileStats) return result;

  const seen = new Set<string>();
  const fileSizes: number[] = [];

  for (const category of [fileStats.staged, fileStats.unstaged, fileStats.untracked]) {
    if (!category) continue;
    for (const [path, stats] of Object.entries(category)) {
      if (seen.has(path)) continue;
      seen.add(path);

      const add = stats.additions ?? 0;
      const del = stats.deletions ?? 0;
      result.totalAdditions += add;
      result.totalDeletions += del;
      result.totalNetLines += stats.net_lines ?? add - del;
      result.totalFiles++;
      if (stats.is_binary) result.binaryCount++;
      if (stats.is_rename) result.renameCount++;

      // File type breakdown
      const ext = fileExtension(path);
      result.fileTypeBreakdown[ext] = (result.fileTypeBreakdown[ext] ?? 0) + 1;

      // Test file detection
      if (isTestFile(path)) result.testFileCount++;

      // Track test additions/deletions separately
      if (isTestFile(path)) {
        result.testAdditions += add;
        result.testDeletions += del;
      }

      // Count new/deleted files
      if (stats.is_new_file) result.newFileCount++;
      if (stats.is_deleted_file) result.deletedFileCount++;

      // Track per-file change size for Pareto
      fileSizes.push(add + del);
    }
  }

  // Churn ratio
  const maxVal = Math.max(result.totalAdditions, result.totalDeletions);
  const minVal = Math.min(result.totalAdditions, result.totalDeletions);
  result.churnRatio = maxVal > 0 ? minVal / maxVal : 0;

  // Pareto concentration
  if (fileSizes.length > 0) {
    fileSizes.sort((a, b) => b - a);
    const topN = Math.max(1, Math.ceil(fileSizes.length * 0.2));
    const totalChanges = fileSizes.reduce((a, b) => a + b, 0);
    const topSum = fileSizes.slice(0, topN).reduce((a, b) => a + b, 0);
    result.paretoTopN = topN;
    result.paretoPercent = totalChanges > 0 ? Math.round((topSum / totalChanges) * 100) : 0;
  }

  // Test-to-code ratio
  const codeChanges = (result.totalAdditions + result.totalDeletions) - (result.testAdditions + result.testDeletions);
  if (codeChanges > 0) {
    result.testToCodeRatio = (result.testAdditions + result.testDeletions) / codeChanges;
  }

  return result;
}

/** Format net lines as a signed string: "+42", "-7", or "0". */
export function formatNetLines(net: number): string {
  if (net > 0) return `+${net}`;
  if (net < 0) return `${net}`;
  return "0";
}

/** Human-readable label for change density. */
export function densityLabel(density?: number): string {
  if (density == null || density === 0) return "";
  if (density <= 0.1) return "focused";
  if (density <= 0.3) return "moderate";
  return "scattered";
}

/** Per-category summary for aggregate modal display. */
export function categoryStats(stats?: Record<string, DiffStats>): { additions: number; deletions: number; netLines: number; count: number } {
  const result = { additions: 0, deletions: 0, netLines: 0, count: 0 };
  if (!stats) return result;
  for (const s of Object.values(stats)) {
    result.additions += s.additions ?? 0;
    result.deletions += s.deletions ?? 0;
    result.netLines += s.net_lines ?? (s.additions ?? 0) - (s.deletions ?? 0);
    result.count++;
  }
  return result;
}

/** Look up DiffStats for a file across all categories. */
export function getFileStats(path: string, fileStats?: RepoFileStats): DiffStats | undefined {
  if (!fileStats) return undefined;
  return fileStats.staged?.[path] ?? fileStats.unstaged?.[path] ?? fileStats.untracked?.[path];
}

/** Filter RepoFileStats to only include specified file paths. */
export function filterFileStats(paths: string[], fileStats?: RepoFileStats): RepoFileStats {
  if (!fileStats) return {};
  const pathSet = new Set(paths);
  const pick = (cat?: Record<string, DiffStats>) => {
    if (!cat) return undefined;
    const result: Record<string, DiffStats> = {};
    for (const [p, s] of Object.entries(cat)) {
      if (pathSet.has(p)) result[p] = s;
    }
    return Object.keys(result).length > 0 ? result : undefined;
  };
  return {
    staged: pick(fileStats.staged),
    unstaged: pick(fileStats.unstaged),
    untracked: pick(fileStats.untracked),
  };
}

/** Filter RepoFileStats to a single category with specified paths. */
export function filterCategoryStats(
  paths: string[],
  category: "staged" | "unstaged" | "untracked",
  fileStats?: RepoFileStats,
): RepoFileStats {
  if (!fileStats) return {};
  const source = fileStats[category];
  if (!source) return {};
  const pathSet = new Set(paths);
  const filtered: Record<string, DiffStats> = {};
  for (const [p, s] of Object.entries(source)) {
    if (pathSet.has(p)) filtered[p] = s;
  }
  return Object.keys(filtered).length > 0 ? { [category]: filtered } : {};
}
