import type { Terminal } from "@xterm/xterm";

const MAX_MATCH_LINES = 160;
const MATCH_PREFIX_CHARS = 240;

function normalizeForTtsMatch(text: string): string {
  return text
    .toLowerCase()
    .replace(/[\u2500-\u257f\u2580-\u259f\u25a0-\u25ff\u276f\u203a\u23f5]+/gu, " ")
    .replace(/[^\p{L}\p{N}\s.,!?;:'"()/_-]+/gu, " ")
    .replace(/\s+/g, " ")
    .trim();
}

export function getRecentTerminalText(terminal: Terminal, maxLines = MAX_MATCH_LINES): string {
  const buffer = terminal.buffer.active;
  const lineCount = buffer.length;
  const start = Math.max(0, lineCount - maxLines);
  const lines: string[] = [];
  for (let i = start; i < lineCount; i += 1) {
    const line = buffer.getLine(i);
    if (!line) continue;
    lines.push(line.translateToString(true));
  }
  return lines.join("\n");
}

export function terminalContainsCandidate(terminal: Terminal, candidateText: string): boolean {
  const haystack = normalizeForTtsMatch(getRecentTerminalText(terminal));
  const candidate = normalizeForTtsMatch(candidateText);
  if (!haystack || !candidate) return false;
  const needle = candidate.length > MATCH_PREFIX_CHARS ? candidate.slice(0, MATCH_PREFIX_CHARS) : candidate;
  return haystack.includes(needle);
}
