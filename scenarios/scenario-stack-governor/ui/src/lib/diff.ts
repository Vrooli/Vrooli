export type DiffLine = {
  type: "added" | "removed" | "unchanged";
  content: string;
  oldLineNo?: number;
  newLineNo?: number;
};

export function computeLineDiff(before: string, after: string): DiffLine[] {
  const oldLines = before.split("\n");
  const newLines = after.split("\n");

  // Build LCS table
  const m = oldLines.length;
  const n = newLines.length;
  const dp: number[][] = Array.from({ length: m + 1 }, () => Array.from({ length: n + 1 }, () => 0));

  for (let i = 1; i <= m; i++) {
    const oldLine = oldLines[i - 1] ?? "";
    const row = dp[i];
    if (!row) {
      continue;
    }
    for (let j = 1; j <= n; j++) {
      const newLine = newLines[j - 1] ?? "";
      const top = dp[i - 1]?.[j] ?? 0;
      const left = row[j - 1] ?? 0;
      const diagonal = dp[i - 1]?.[j - 1] ?? 0;
      if (oldLine === newLine) {
        row[j] = diagonal + 1;
      } else {
        row[j] = Math.max(top, left);
      }
    }
  }

  // Backtrack to build diff
  let i = m, j = n;
  const stack: DiffLine[] = [];

  while (i > 0 || j > 0) {
    const oldLine = i > 0 ? (oldLines[i - 1] ?? "") : "";
    const newLine = j > 0 ? (newLines[j - 1] ?? "") : "";
    const left = j > 0 ? (dp[i]?.[j - 1] ?? 0) : 0;
    const top = i > 0 ? (dp[i - 1]?.[j] ?? 0) : 0;

    if (i > 0 && j > 0 && oldLine === newLine) {
      stack.push({ type: "unchanged", content: oldLine, oldLineNo: i, newLineNo: j });
      i--;
      j--;
    } else if (j > 0 && (i === 0 || left >= top)) {
      stack.push({ type: "added", content: newLine, newLineNo: j });
      j--;
    } else {
      stack.push({ type: "removed", content: oldLine, oldLineNo: i });
      i--;
    }
  }

  stack.reverse();
  return stack;
}
