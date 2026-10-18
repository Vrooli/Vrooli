export type DiffKind = "add" | "remove" | "context" | "note";
export interface DiffLine { id: number; text: string; kind: DiffKind; old: number | null; next: number | null }
export interface DiffHunk { id: number; header: string; lines: DiffLine[] }
export interface DiffFile { id: number; name: string; oldName: string; hunks: DiffHunk[]; metadata: string[]; added: number; removed: number }
export interface ParsedDiff { files: DiffFile[]; valid: boolean }
export const MAX_DIFF_LINES = 12000;
export function parseDiff(content: string): ParsedDiff {
  const lines = content.split(/\r?\n/);
  if (lines.length > MAX_DIFF_LINES) return { files: [], valid: false };
  const files: DiffFile[] = [];
  let file: DiffFile | undefined, hunk: DiffHunk | undefined, old = 0, next = 0, oldLeft = 0, nextLeft = 0, valid = true, pendingOld = "";
  const newFile = (name: string) => { file = { id: files.length, name, oldName: "", hunks: [], metadata: [], added: 0, removed: 0 }; files.push(file); hunk = undefined; };
  const finish = () => { if (hunk && (oldLeft !== 0 || nextLeft !== 0)) valid = false; hunk = undefined; };
  const path = (value: string) => value.split("\t")[0]?.replace(/^[ab]\//, "") ?? value;
  for (let i = 0; i < lines.length; i++) {
    const line = lines[i] ?? "";
    if (line.startsWith("diff --git ")) { finish(); newFile(line.slice(11)); file?.metadata.push(line); pendingOld = ""; continue; }
    if (line.startsWith("@@@")) { valid = false; continue; }
    const header = /^@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@(?:.*)$/.exec(line);
    if (header) {
      finish(); if (!file) newFile("");
      old = Number(header[1]); next = Number(header[3]); oldLeft = Number(header[2] ?? 1); nextLeft = Number(header[4] ?? 1);
      hunk = { id: i, header: line, lines: [] }; file?.hunks.push(hunk); continue;
    }
    if (hunk && (oldLeft > 0 || nextLeft > 0 || line.startsWith("\\"))) {
      const prefix = line[0];
      if (prefix === "+" || prefix === "-" || prefix === " ") {
        const kind = prefix === "+" ? "add" : prefix === "-" ? "remove" : "context";
        hunk.lines.push({ id: i, text: line, kind, old: kind === "add" ? null : old++, next: kind === "remove" ? null : next++ });
        if (kind !== "add") oldLeft--; if (kind !== "remove") nextLeft--;
        if (oldLeft < 0 || nextLeft < 0) valid = false;
        if (file) { if (kind === "add") file.added++; if (kind === "remove") file.removed++; }
      } else if (prefix === "\\") hunk.lines.push({ id: i, text: line, kind: "note", old: null, next: null });
      else { valid = false; finish(); }
      continue;
    }
    if (line.startsWith("--- ")) { finish(); if (!file || file.hunks.length || pendingOld) newFile(""); pendingOld = path(line.slice(4)); if (file) file.oldName = pendingOld; continue; }
    if (line.startsWith("+++ ")) { if (!file) newFile(""); if (file) file.name = path(line.slice(4)); pendingOld = ""; continue; }
    if (line.startsWith("@@") || /^[+ -]/.test(line) && hunk) { valid = false; continue; }
    if (line && file) file.metadata.push(line);
  }
  finish();
  return { files, valid: valid && files.some((entry) => entry.hunks.length > 0 || entry.metadata.some((line) => /^(?:Binary files|GIT binary patch|rename |new file mode|deleted file mode|old mode|new mode)/.test(line))) };
}
export interface DiffPair { left: DiffLine | null; right: DiffLine | null }
export function splitHunk(lines: DiffLine[]): DiffPair[] {
  const rows: DiffPair[] = [];
  for (let i = 0; i < lines.length;) {
    const line = lines[i]; if (!line) break;
    if (line.kind === "remove" || line.kind === "add") {
      const removed: DiffLine[] = [], added: DiffLine[] = [];
      while (lines[i]?.kind === "remove") { const entry = lines[i++]; if (entry) removed.push(entry); }
      while (lines[i]?.kind === "add") { const entry = lines[i++]; if (entry) added.push(entry); }
      for (let j = 0; j < Math.max(removed.length, added.length); j++) rows.push({ left: removed[j] ?? null, right: added[j] ?? null });
    } else { rows.push({ left: line, right: line }); i++; }
  }
  return rows;
}
