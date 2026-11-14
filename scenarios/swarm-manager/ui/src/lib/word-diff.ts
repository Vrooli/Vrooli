/**
 * Minimal LCS diff over words and lines.
 *
 * Proposal payloads carry prose (item descriptions run to ~1.3k characters), so
 * the decision surface needs a real before/after, not a "changed" flag. This is
 * deliberately dependency-free: pulling a diff library through package
 * governance for ~120 lines of well-understood dynamic programming is a worse
 * trade than owning it.
 *
 * Cost control is two-stage: common prefix/suffix are trimmed before the
 * matrix is built (edits to prose are usually local), and a cell budget caps
 * the quadratic step so a pathological pair degrades to replace-the-block
 * instead of hanging the tab.
 */

export type ChangeKind = "equal" | "insert" | "delete";

export interface Change<T> {
  kind: ChangeKind;
  value: T;
}

/** Above this many DP cells, fall back to a whole-block replace. */
const MAX_LCS_CELLS = 250_000;

/**
 * Longest-common-subsequence diff over two sequences.
 *
 * Emits deletions before insertions at each divergence so callers can pair
 * them into before/after rows without re-sorting.
 */
export function diffSequence<T extends NonNullable<unknown>>(before: readonly T[], after: readonly T[]): Change<T>[] {
  let start = 0;
  const maxStart = Math.min(before.length, after.length);
  while (start < maxStart) {
    const left = before[start];
    const right = after[start];
    if (left === undefined || right === undefined || left !== right) break;
    start++;
  }

  let end = 0;
  const maxEnd = Math.min(before.length - start, after.length - start);
  while (end < maxEnd) {
    const left = before[before.length - 1 - end];
    const right = after[after.length - 1 - end];
    if (left === undefined || right === undefined || left !== right) break;
    end++;
  }

  const head = before.slice(0, start).map<Change<T>>((value) => ({ kind: "equal", value }));
  const tail = end > 0 ? before.slice(before.length - end).map<Change<T>>((value) => ({ kind: "equal", value })) : [];
  const midBefore = before.slice(start, before.length - end);
  const midAfter = after.slice(start, after.length - end);

  if (midBefore.length === 0 && midAfter.length === 0) return [...head, ...tail];

  const middle: Change<T>[] = (midBefore.length + 1) * (midAfter.length + 1) > MAX_LCS_CELLS
    ? [
      ...midBefore.map<Change<T>>((value) => ({ kind: "delete", value })),
      ...midAfter.map<Change<T>>((value) => ({ kind: "insert", value })),
    ]
    : backtrack(midBefore, midAfter);

  return [...head, ...middle, ...tail];
}

function backtrack<T extends NonNullable<unknown>>(before: readonly T[], after: readonly T[]): Change<T>[] {
  const width = after.length + 1;
  const dp = new Uint32Array((before.length + 1) * width);
  // Reads outside the allocated matrix are the LCS base case (zero), so the
  // fallback is the algorithm's boundary condition rather than error masking.
  const cell = (row: number, column: number): number => dp[row * width + column] ?? 0;

  for (let i = before.length - 1; i >= 0; i--) {
    for (let j = after.length - 1; j >= 0; j--) {
      const left = before[i];
      const right = after[j];
      dp[i * width + j] = left !== undefined && right !== undefined && left === right
        ? cell(i + 1, j + 1) + 1
        : Math.max(cell(i + 1, j), cell(i, j + 1));
    }
  }

  const changes: Change<T>[] = [];
  let i = 0;
  let j = 0;
  while (i < before.length && j < after.length) {
    const left = before[i];
    const right = after[j];
    if (left === undefined || right === undefined) break;
    if (left === right) {
      changes.push({ kind: "equal", value: left });
      i++;
      j++;
    } else if (cell(i + 1, j) >= cell(i, j + 1)) {
      changes.push({ kind: "delete", value: left });
      i++;
    } else {
      changes.push({ kind: "insert", value: right });
      j++;
    }
  }
  for (; i < before.length; i++) {
    const value = before[i];
    if (value !== undefined) changes.push({ kind: "delete", value });
  }
  for (; j < after.length; j++) {
    const value = after[j];
    if (value !== undefined) changes.push({ kind: "insert", value });
  }
  return changes;
}

/** Splits into words and whitespace runs so a rejoin is lossless. */
export function tokenizeWords(text: string): string[] {
  return text.match(/\s+|\S+/g) ?? [];
}

export interface DiffSegment {
  kind: ChangeKind;
  text: string;
}

/** Word-level diff with adjacent same-kind tokens coalesced into one segment. */
export function diffWords(before: string, after: string): DiffSegment[] {
  const changes = diffSequence(tokenizeWords(before), tokenizeWords(after));
  const segments: DiffSegment[] = [];
  for (const change of changes) {
    const last = segments.at(-1);
    if (last && last.kind === change.kind) last.text += change.value;
    else segments.push({ kind: change.kind, text: change.value });
  }
  return segments;
}

export type DiffRowKind = "context" | "delete" | "insert";

export interface DiffRow {
  kind: DiffRowKind;
  /** Word-level segments when this row was paired with its counterpart. */
  segments: DiffSegment[];
}

/** Fraction of shared word tokens above which two lines are diffed inline. */
const PAIRING_SIMILARITY = 0.34;

function similarity(before: string, after: string): number {
  const left = tokenizeWords(before).filter((token) => token.trim());
  const right = new Set(tokenizeWords(after).filter((token) => token.trim()));
  if (left.length === 0 || right.size === 0) return 0;
  const shared = left.filter((token) => right.has(token)).length;
  return shared / Math.max(left.length, right.size);
}

/**
 * Unified line diff. Deleted/inserted runs are paired positionally, and a pair
 * that is similar enough gets word-level segments so the operator sees the
 * words that actually moved rather than two walls of text.
 */
export function buildLineDiff(before: string, after: string): DiffRow[] {
  const changes = diffSequence(before.split("\n"), after.split("\n"));
  const rows: DiffRow[] = [];
  let pendingDeletes: string[] = [];
  let pendingInserts: string[] = [];

  const flush = () => {
    const paired = Math.min(pendingDeletes.length, pendingInserts.length);
    for (let index = 0; index < paired; index++) {
      const deleted = pendingDeletes[index];
      const inserted = pendingInserts[index];
      if (deleted === undefined || inserted === undefined) continue;
      if (similarity(deleted, inserted) >= PAIRING_SIMILARITY) {
        const segments = diffWords(deleted, inserted);
        rows.push({ kind: "delete", segments: segments.filter((segment) => segment.kind !== "insert") });
        rows.push({ kind: "insert", segments: segments.filter((segment) => segment.kind !== "delete") });
      } else {
        rows.push({ kind: "delete", segments: [{ kind: "delete", text: deleted }] });
        rows.push({ kind: "insert", segments: [{ kind: "insert", text: inserted }] });
      }
    }
    for (const deleted of pendingDeletes.slice(paired)) rows.push({ kind: "delete", segments: [{ kind: "delete", text: deleted }] });
    for (const inserted of pendingInserts.slice(paired)) rows.push({ kind: "insert", segments: [{ kind: "insert", text: inserted }] });
    pendingDeletes = [];
    pendingInserts = [];
  };

  for (const change of changes) {
    if (change.kind === "delete") pendingDeletes.push(change.value);
    else if (change.kind === "insert") pendingInserts.push(change.value);
    else {
      flush();
      rows.push({ kind: "context", segments: [{ kind: "equal", text: change.value }] });
    }
  }
  flush();
  return rows;
}

export interface DiffStat {
  removed: number;
  added: number;
}

/** Characters removed and added, for the `−412 +1,043` readout on a field. */
export function diffStat(before: string, after: string): DiffStat {
  let removed = 0;
  let added = 0;
  for (const segment of diffWords(before, after)) {
    if (segment.kind === "delete") removed += segment.text.length;
    else if (segment.kind === "insert") added += segment.text.length;
  }
  return { removed, added };
}
