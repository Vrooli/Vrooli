import type { BacklogFile, DecisionOption } from "../types/domain";

/** Sentinel for the operator-provided freeform response option. */
export const OTHER_KEY = "__other__";

/** Keep one consistent freeform option instead of duplicating agent-provided labels. */
export function filterAgentOther(options: DecisionOption[]): DecisionOption[] {
  return options.filter((option) => option.label.toLowerCase().trim() !== "other");
}

/** Resolve a selected file from a nested backlog file tree. */
export function findBacklogFileByPath(
  files: BacklogFile[] | undefined,
  targetPath: string,
): BacklogFile | null {
  if (!files) return null;
  for (const file of files) {
    if (file.path === targetPath) return file;
    const nested = findBacklogFileByPath(file.children, targetPath);
    if (nested) return nested;
  }
  return null;
}
