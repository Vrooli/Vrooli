import type { Run, RunDiff } from "../types";

export type ReviewFileCount =
  | { available: true; count: number }
  | { available: false };

// changedFiles is a useful positive summary, but its zero value cannot prove
// that a diff was completely read. The API marks a file index verified only
// after validating the provider evidence; older API responses default false.
export function reviewFileCount(
  run: Pick<Run, "id" | "changedFiles">,
  diff?: Pick<RunDiff, "runId" | "files" | "fileCountVerified"> | null,
): ReviewFileCount {
  if (diff?.runId === run.id && diff.fileCountVerified && Array.isArray(diff.files)) {
    return { available: true, count: diff.files.length };
  }
  if (run.changedFiles > 0) return { available: true, count: run.changedFiles };
  return { available: false };
}
