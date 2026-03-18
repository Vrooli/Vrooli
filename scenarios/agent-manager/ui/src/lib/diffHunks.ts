export interface HunkLine {
  type: "add" | "remove" | "context" | "no-newline";
  content: string;
  oldLine?: number;
  newLine?: number;
}

export interface Hunk {
  header: string;
  context: string;
  oldStart: number;
  newStart: number;
  lines: HunkLine[];
}

export function parseHunks(patch: string): Hunk[] {
  if (!patch) return [];
  const hunks: Hunk[] = [];
  let currentHunk: Hunk | null = null;
  let oldLine = 0;
  let newLine = 0;

  for (const raw of patch.split("\n")) {
    const hunkMatch = raw.match(/^@@ -(\d+)(?:,\d+)? \+(\d+)(?:,\d+)? @@(.*)/);
    if (hunkMatch) {
      currentHunk = {
        header: raw,
        context: hunkMatch[3]?.trim() || "",
        oldStart: Number.parseInt(hunkMatch[1] ?? "0", 10),
        newStart: Number.parseInt(hunkMatch[2] ?? "0", 10),
        lines: [],
      };
      oldLine = currentHunk.oldStart;
      newLine = currentHunk.newStart;
      hunks.push(currentHunk);
      continue;
    }

    if (!currentHunk) continue;

    if (raw.startsWith("\\ No newline")) {
      currentHunk.lines.push({ type: "no-newline", content: raw });
    } else if (raw.startsWith("+")) {
      currentHunk.lines.push({ type: "add", content: raw.slice(1), newLine: newLine++ });
    } else if (raw.startsWith("-")) {
      currentHunk.lines.push({ type: "remove", content: raw.slice(1), oldLine: oldLine++ });
    } else {
      currentHunk.lines.push({
        type: "context",
        content: raw.startsWith(" ") ? raw.slice(1) : raw,
        oldLine: oldLine++,
        newLine: newLine++,
      });
    }
  }

  return hunks;
}
