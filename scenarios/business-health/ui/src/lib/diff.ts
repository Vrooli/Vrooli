/**
 * Zero-dependency line diff for the wizard scaffold + fix-preview surfaces.
 *
 * Produces a unified line list (context / added / removed) from a classic
 * longest-common-subsequence over lines. Kept deliberately small and pure so
 * it is trivially unit-testable and adds nothing to the bundle beyond a few
 * hundred bytes. Not a full Myers diff — inputs here are short generated
 * artifacts (PRD sections, requirement JSON, config edits), so an O(n·m) LCS
 * is comfortably fast and produces stable, readable output.
 */

export type DiffOp = "context" | "add" | "remove";

export interface DiffLine {
  readonly op: DiffOp;
  /** Line number in the "before" text (undefined for added lines). */
  readonly beforeLine?: number;
  /** Line number in the "after" text (undefined for removed lines). */
  readonly afterLine?: number;
  readonly text: string;
}

export interface DiffStats {
  readonly added: number;
  readonly removed: number;
}

const splitLines = (value: string): string[] => {
  if (value === "") return [];
  // Normalize CRLF so a Windows-authored artifact doesn't show every line as changed.
  return value.replace(/\r\n/g, "\n").split("\n");
};

/**
 * Compute a unified line diff between two texts. Empty `before` renders as a
 * pure-addition (new file); empty `after` as a pure-removal.
 */
export function diffLines(before: string, after: string): DiffLine[] {
  const a = splitLines(before);
  const b = splitLines(after);
  const n = a.length;
  const m = b.length;

  // LCS length table, addressed through a flat array so index reads return a
  // definite number (avoids non-null assertions under noUncheckedIndexedAccess).
  const width = m + 1;
  const lcs = new Array<number>((n + 1) * width).fill(0);
  const at = (i: number, j: number): number => lcs[i * width + j] ?? 0;
  const set = (i: number, j: number, value: number): void => {
    lcs[i * width + j] = value;
  };
  for (let i = n - 1; i >= 0; i -= 1) {
    for (let j = m - 1; j >= 0; j -= 1) {
      if (a[i] === b[j]) {
        set(i, j, at(i + 1, j + 1) + 1);
      } else {
        set(i, j, Math.max(at(i + 1, j), at(i, j + 1)));
      }
    }
  }

  const out: DiffLine[] = [];
  let i = 0;
  let j = 0;
  while (i < n && j < m) {
    const left = a[i] ?? "";
    const right = b[j] ?? "";
    if (left === right) {
      out.push({ op: "context", beforeLine: i + 1, afterLine: j + 1, text: left });
      i += 1;
      j += 1;
    } else if (at(i + 1, j) >= at(i, j + 1)) {
      out.push({ op: "remove", beforeLine: i + 1, text: left });
      i += 1;
    } else {
      out.push({ op: "add", afterLine: j + 1, text: right });
      j += 1;
    }
  }
  while (i < n) {
    out.push({ op: "remove", beforeLine: i + 1, text: a[i] ?? "" });
    i += 1;
  }
  while (j < m) {
    out.push({ op: "add", afterLine: j + 1, text: b[j] ?? "" });
    j += 1;
  }
  return out;
}

/** Count added/removed lines in a computed diff. */
export function diffStats(lines: DiffLine[]): DiffStats {
  let added = 0;
  let removed = 0;
  for (const line of lines) {
    if (line.op === "add") added += 1;
    else if (line.op === "remove") removed += 1;
  }
  return { added, removed };
}
