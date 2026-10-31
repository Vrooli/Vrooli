import type { IBufferCell, Terminal } from "@xterm/xterm";

export interface TerminalCommandBlock {
  kind: "command";
  value: string;
  text: string;
  startRow: number;
  endRow: number;
  lineCount: number;
}

export interface TerminalCommandActivation extends TerminalCommandBlock {
  position: { x: number; y: number };
}

export const MAX_COMMAND_BLOCK_ROWS = 12;
const MAX_COMMAND_BLOCK_CHARS = 4000;
const MAX_SCAN_ROWS = 40;
const WRAP_MARGIN = 3;
const MIN_STYLED_GLYPHS = 2;
const MIN_STYLED_RATIO = 0.15;
const MIN_STYLED_SPAN = 4;

const COMMAND_WORDS = [
  "sudo", "doas", "timeout", "env", "nohup", "watch", "xargs", "tee",
  "bash", "sh", "zsh", "fish", "python", "python3", "node", "deno", "bun",
  "npm", "npx", "pnpm", "pnpx", "yarn", "bunx", "pip", "pip3", "pipx", "uv", "uvx",
  "poetry", "go", "cargo", "rustc", "make", "cmake", "ninja", "gcc", "clang",
  "git", "gh", "curl", "wget", "ssh", "scp", "rsync", "tar", "zip", "unzip",
  "docker", "podman", "kubectl", "helm", "systemctl", "service", "journalctl",
  "apt", "apt-get", "dnf", "yum", "pacman", "brew",
  "bpftrace", "strace", "ltrace", "perf", "dmesg", "sysctl", "modprobe", "insmod", "lsmod",
  "grep", "rg", "find", "sed", "awk", "cat", "head", "tail", "less", "more",
  "ls", "cd", "pwd", "echo", "printf", "openssl", "chmod", "chown", "chgrp",
  "rm", "rmdir", "mv", "cp", "mkdir", "touch", "ln", "export", "source", "eval", "exec",
  "kill", "pkill", "killall", "mount", "umount", "lsblk", "df", "du", "free",
  "top", "ps", "ss", "netstat", "ip", "iptables", "nft", "file", "stat", "which",
  "type", "command", "test", "true", "false", "sleep", "seq", "date", "uname",
] as const;

const COMMAND_WORD_SET = new Set<string>(COMMAND_WORDS);
const COMMAND_START_RE = new RegExp(
  `(?:^|\\n|[-*]\\s+|[.!?:;]\\s+)(?<cmd>${COMMAND_WORDS.join("|")})\\b`,
  "i",
);
const PROMPT_RE = /(^|\n)[ \t]*[$❯➜][ \t]+/;
const SHEBANG_RE = /(^|\n)[ \t]*(#!)/;
const CODE_KEYWORD_RE = /(^|\n)[ \t]*(?:def|class|function|import|from|package|public|private|protected|fn|func|let|const|var|namespace|module|use|#include)\b/;

interface RowInfo {
  row: number;
  text: string;
  wrapped: boolean;
  blank: boolean;
  styled: boolean;
  glyphs: number;
  styledGlyphs: number;
  firstStyled: number;
  lastStyled: number;
  cellChars: string[];
  cellStyles: boolean[];
}

interface DelimiterState {
  quoteOpen: boolean;
  depth: number;
}

function readRow(term: Terminal, row: number, reuse?: IBufferCell): RowInfo | null {
  const line = term.buffer.active.getLine(row);
  if (!line) return null;
  const text = line.translateToString(true);
  const blank = text.trim().length === 0;
  const cellChars: string[] = [];
  const cellStyles: boolean[] = [];
  let glyphs = 0;
  let styledGlyphs = 0;
  let firstStyled = -1;
  let lastStyled = -1;
  let span = 0;
  let maxSpan = 0;

  for (let col = 0; col < term.cols; col++) {
    const cell = line.getCell(col, reuse);
    const width = cell?.getWidth() ?? 0;
    const chars = cell && width !== 0 ? cell.getChars() : "";
    const styled = cell
      ? !cell.isFgDefault() || cell.isBold() !== 0 || cell.isItalic() !== 0 || cell.isUnderline() !== 0
      : false;
    cellChars.push(chars);
    cellStyles.push(styled);
    if (width === 0) continue;
    if (!chars || chars.trim().length === 0) {
      span = 0;
      continue;
    }
    glyphs++;
    if (styled) {
      styledGlyphs++;
      span++;
      maxSpan = Math.max(maxSpan, span);
      if (firstStyled < 0) firstStyled = col;
      lastStyled = col;
    } else {
      span = 0;
    }
  }

  const styled = glyphs >= MIN_STYLED_GLYPHS
    && styledGlyphs >= MIN_STYLED_GLYPHS
    && (styledGlyphs / glyphs >= MIN_STYLED_RATIO || maxSpan >= MIN_STYLED_SPAN);

  return {
    row,
    text,
    wrapped: line.isWrapped,
    blank,
    styled,
    glyphs,
    styledGlyphs,
    firstStyled,
    lastStyled,
    cellChars,
    cellStyles,
  };
}

function findTappedRow(term: Terminal, bufferRow: number, col: number, margin: number): RowInfo | null {
  const row = readRow(term, bufferRow);
  if (!row || row.blank || !row.styled) return null;
  if (row.firstStyled < 0) return null;
  if (col < row.firstStyled - margin || col > row.lastStyled + margin) return null;
  return row;
}

function isStyledGlyph(row: RowInfo, col: number): boolean {
  const chars = row.cellChars[col] ?? "";
  return chars.trim().length > 0 && row.cellStyles[col] === true;
}

interface CellSpan {
  text: string;
  start: number;
  end: number;
}

function styledSpanAt(row: RowInfo, col: number, margin: number): CellSpan | null {
  let snap = -1;
  for (let distance = 0; distance <= margin && snap < 0; distance++) {
    if (isStyledGlyph(row, col - distance)) snap = col - distance;
    else if (distance > 0 && isStyledGlyph(row, col + distance)) snap = col + distance;
  }
  if (snap < 0) return null;

  let start = snap;
  for (let current = snap - 1; current >= 0; current--) {
    const chars = row.cellChars[current] ?? "";
    if (chars.trim().length === 0) continue;
    if (row.cellStyles[current] === true) start = current;
    else break;
  }

  let end = snap;
  for (let current = snap + 1; current < row.cellChars.length; current++) {
    const chars = row.cellChars[current] ?? "";
    if (chars.trim().length === 0) continue;
    if (row.cellStyles[current] === true) end = current;
    else break;
  }

  const text = row.cellChars.slice(start, end + 1).join("").trim();
  return text ? { text, start, end } : null;
}

function firstNonSpace(row: RowInfo): number {
  for (let col = 0; col < row.cellChars.length; col++) {
    if ((row.cellChars[col] ?? "").trim().length > 0) return col;
  }
  return -1;
}

function lastNonSpace(row: RowInfo): number {
  for (let col = row.cellChars.length - 1; col >= 0; col--) {
    if ((row.cellChars[col] ?? "").trim().length > 0) return col;
  }
  return -1;
}

function unstyledGlyphs(row: RowInfo, from: number, to: number): number {
  let count = 0;
  for (let col = Math.max(0, from); col < Math.min(to, row.cellChars.length); col++) {
    const chars = row.cellChars[col] ?? "";
    if (chars.trim().length > 0 && row.cellStyles[col] !== true) count++;
  }
  return count;
}

function hasProseBefore(row: RowInfo): boolean {
  return row.firstStyled > 0 && unstyledGlyphs(row, 0, row.firstStyled) >= 2;
}

function hasProseAfter(row: RowInfo): boolean {
  return row.lastStyled >= 0 && unstyledGlyphs(row, row.lastStyled + 1, row.cellChars.length) >= 2;
}

function startsWithUnstyledGlyph(row: RowInfo): boolean {
  for (let col = 0; col < row.cellChars.length; col++) {
    const chars = row.cellChars[col] ?? "";
    if (chars.trim().length === 0) continue;
    return row.cellStyles[col] !== true;
  }
  return false;
}

function looksLikeInlineSpan(pivot: RowInfo, next: RowInfo | undefined): boolean {
  if (hasProseAfter(pivot)) return true;
  if (!hasProseBefore(pivot)) return false;
  if (!next) return true;
  return startsWithUnstyledGlyph(next) || hasProseAfter(next);
}

function seamBetween(term: Terminal, previous: RowInfo, next: RowInfo): string {
  if (/^\s/.test(next.text)) return " ";
  const previousText = previous.text.trimEnd();
  const nextText = next.text.trimStart();
  const hardWrap = previous.wrapped
    && previousText.length >= term.cols - WRAP_MARGIN
    && /[^\s"'`]/.test(previousText.slice(-1))
    && /[^\s"'`]/.test(nextText.charAt(0));
  return hardWrap ? "" : " ";
}

function inlineSpanAcrossRows(term: Terminal, pivot: RowInfo, col: number, margin: number): string | null {
  const pivotSpan = styledSpanAt(pivot, col, margin);
  if (!pivotSpan) return null;

  const after: Array<{ row: RowInfo; span: CellSpan }> = [];
  let row = pivot;
  let span = pivotSpan;
  while (span.end >= lastNonSpace(row)) {
    const next = readRow(term, row.row + 1);
    const nextFirst = next ? firstNonSpace(next) : -1;
    if (!next || next.blank || !next.styled || nextFirst < 0 || next.cellStyles[nextFirst] !== true) break;
    const nextSpan = styledSpanAt(next, nextFirst, 0);
    if (!nextSpan || nextSpan.start !== nextFirst) break;
    after.push({ row: next, span: nextSpan });
    row = next;
    span = nextSpan;
  }

  const before: Array<{ row: RowInfo; span: CellSpan }> = [];
  row = pivot;
  span = pivotSpan;
  while (span.start <= firstNonSpace(row)) {
    const previous = readRow(term, row.row - 1);
    const previousLast = previous ? lastNonSpace(previous) : -1;
    if (!previous || previous.blank || !previous.styled || previousLast < 0 || previous.cellStyles[previousLast] !== true) break;
    const previousSpan = styledSpanAt(previous, previousLast, 0);
    if (!previousSpan || previousSpan.end !== previousLast) break;
    before.unshift({ row: previous, span: previousSpan });
    row = previous;
    span = previousSpan;
  }

  let text = pivotSpan.text;
  let previousRow = pivot;
  for (const part of after) {
    text += `${seamBetween(term, previousRow, part.row)}${part.span.text}`;
    previousRow = part.row;
  }
  if (before.length > 0) {
    let prefix = before[0]?.span.text ?? "";
    for (let index = 1; index < before.length; index++) {
      const part = before[index];
      const prior = before[index - 1];
      if (!part || !prior) continue;
      prefix += `${seamBetween(term, prior.row, part.row)}${part.span.text}`;
    }
    const lastBefore = before[before.length - 1];
    if (lastBefore) text = `${prefix}${seamBetween(term, lastBefore.row, pivot)}${text}`;
  }
  return text.trim();
}

function collectRows(term: Terminal, pivot: RowInfo): RowInfo[] {
  const rows: RowInfo[] = [pivot];
  let start = pivot.row;
  let end = pivot.row;

  while (start > 0 && pivot.row - start < MAX_SCAN_ROWS) {
    const previous = readRow(term, start - 1);
    if (!previous || previous.blank || (!previous.styled && !previous.wrapped)) break;
    rows.unshift(previous);
    start--;
  }
  while (end < pivot.row + MAX_SCAN_ROWS) {
    const next = readRow(term, end + 1);
    if (!next || next.blank || (!next.styled && !next.wrapped)) break;
    rows.push(next);
    end++;
  }
  return rows;
}

function delimiterState(text: string): DelimiterState {
  let single = false;
  let double = false;
  let backtick = false;
  let depth = 0;
  for (let i = 0; i < text.length; i++) {
    const ch = text[i];
    if (ch === "\\") {
      i++;
      continue;
    }
    if (single) {
      if (ch === "'") single = false;
      continue;
    }
    if (double) {
      if (ch === '"') double = false;
      continue;
    }
    if (backtick) {
      if (ch === "`") backtick = false;
      continue;
    }
    if (ch === "'") { single = true; continue; }
    if (ch === '"') { double = true; continue; }
    if (ch === "`") { backtick = true; continue; }
    if (ch === "(" || ch === "[" || ch === "{") depth++;
    else if (ch === ")" || ch === "]" || ch === "}") depth = Math.max(0, depth - 1);
  }
  return { quoteOpen: single || double || backtick, depth };
}

function firstWord(text: string): string {
  return text.trimStart().split(/\s/, 1)[0] ?? "";
}

function startsWithCommand(text: string): boolean {
  const trimmed = text.trimStart();
  if (/^(?:[$❯➜]|#!|\.{1,2}\/|~\/)/.test(trimmed)) return true;
  if (CODE_KEYWORD_RE.test(trimmed)) return true;
  const word = firstWord(trimmed).replace(/[;|&<>]+$/, "");
  return COMMAND_WORD_SET.has(word);
}

function continuesNextRow(current: RowInfo, next: RowInfo, accumulated: string, wrapWidth: number, wrapEvidence: boolean): boolean {
  if (next.blank) return false;
  const currentText = current.text.trimEnd();
  const nextText = next.text.trimStart();
  if (!currentText) return false;
  if (/^[$❯➜]\s+/.test(nextText)) return false;
  if (startsWithCommand(currentText) && startsWithCommand(nextText)) return false;

  if (current.wrapped) return true;
  if (currentText.length >= (wrapWidth > 0 ? wrapWidth : 0) - 1 && wrapEvidence) return true;

  const delimiters = delimiterState(accumulated);
  if (delimiters.quoteOpen || delimiters.depth > 0) return true;
  if (/[|&\\,=([{]$/.test(currentText)) return true;
  if (/(?:^|\s)(?:then|do|else|elif|in)$/.test(currentText)) return true;
  if (/^([)\]{}>|&]|2?>|2>&1|then\b|do\b|else\b|elif\b|fi\b|done\b|esac\b)/.test(nextText)) return true;

  if (wrapEvidence && currentText.length + 1 + firstWord(nextText).length > wrapWidth + 1) return true;
  return false;
}

function joinRows(rows: RowInfo[], term: Terminal): string {
  const wrapWidth = rows.reduce((max, row) => Math.max(max, row.text.trimEnd().length), 0);
  const nearWidthRows = rows.filter((row) => row.text.trimEnd().length >= wrapWidth - 2).length;
  const wrapEvidence = rows.some((row) => row.wrapped) || nearWidthRows >= 2;

  let assembled = rows[0]?.text.trimEnd() ?? "";
  let logical = assembled;
  const lines: string[] = [];

  for (let i = 0; i < rows.length; i++) {
    const current = rows[i];
    if (!current) continue;
    if (i > 0) {
      const previous = rows[i - 1];
      const previousText = previous?.text.trimEnd() ?? "";
      if (continuesNextRow(previous ?? current, current, logical, wrapWidth, wrapEvidence)) {
        const nextText = current.text.trimStart();
        const seamSpace = /^\s/.test(current.text);
        const hardWrap = !seamSpace
          && previousText.length >= term.cols - WRAP_MARGIN
          && /[^\s"'`]/.test(previousText.slice(-1))
          && /[^\s"'`]/.test(nextText.charAt(0));
        const joiner = hardWrap ? "" : " ";
        assembled += `${joiner}${nextText}`;
        logical += `${joiner}${nextText}`;
      } else {
        lines.push(assembled.trimEnd());
        assembled = current.text.trimStart();
        logical = assembled;
      }
    }
  }
  lines.push(assembled.trimEnd());
  return lines.filter((line) => line.trim().length > 0).join("\n");
}

function findCodeStart(text: string): number | null {
  const prompt = PROMPT_RE.exec(text);
  if (prompt) return prompt.index + prompt[0].length;
  const shebang = SHEBANG_RE.exec(text);
  if (shebang) return shebang.index + shebang[0].length - (shebang[2]?.length ?? 0);
  const command = COMMAND_START_RE.exec(text);
  if (command?.groups?.cmd) {
    return command.index + command[0].indexOf(command.groups.cmd);
  }
  const keyword = CODE_KEYWORD_RE.exec(text);
  if (keyword) return keyword.index + keyword[0].length - keyword[0].trimStart().length;
  return null;
}

function looksLikeCode(text: string): boolean {
  const trimmed = text.trim();
  if (!trimmed) return false;
  if (/^(?:#!|[$❯➜])/.test(trimmed)) return true;
  if (/^(?:def|class|function|import|from|package|public|private|protected|fn|func|let|const|var|namespace|module|use|#include)\b/.test(trimmed)) return true;
  if (startsWithCommand(trimmed)) return true;

  let score = 0;
  if (/[|&;<>]|\$\(|`|\\/.test(trimmed)) score++;
  if (/["']/.test(trimmed)) score++;
  if (/(?:^|\s)--?[A-Za-z]/.test(trimmed)) score++;
  if (/[=(){}[\]]/.test(trimmed)) score++;
  if (/\//.test(trimmed)) score++;
  return score >= 3;
}

function dedent(lines: string[]): string[] {
  const nonEmpty = lines.filter((line) => line.trim().length > 0);
  if (nonEmpty.length === 0) return lines;
  let indent = Infinity;
  for (const line of nonEmpty) {
    const match = /^[ \t]*/.exec(line);
    indent = Math.min(indent, match?.[0].length ?? 0);
  }
  if (!Number.isFinite(indent) || indent <= 0) return lines;
  return lines.map((line) => line.slice(Math.min(indent, line.length - line.trimStart().length)));
}

export function commandBlockAtCell(term: Terminal, bufferRow: number, col: number, margin = 2): TerminalCommandBlock | null {
  const pivot = findTappedRow(term, bufferRow, col, margin);
  if (!pivot) return null;

  const rows = collectRows(term, pivot);
  if (rows.length === 0) return null;

  const pivotIndex = rows.findIndex((row) => row.row === pivot.row);
  const next = pivotIndex >= 0 ? rows[pivotIndex + 1] : undefined;
  if (rows.length === 1 || looksLikeInlineSpan(pivot, next)) {
    const span = inlineSpanAcrossRows(term, pivot, col, margin);
    if (!span || !looksLikeCode(span)) return null;
    const value = span.trim();
    if (!value || value.length > MAX_COMMAND_BLOCK_CHARS) return null;
    return {
      kind: "command",
      value,
      text: value,
      startRow: pivot.row,
      endRow: pivot.row,
      lineCount: 1,
    };
  }

  const assembled = joinRows(rows, term);
  if (!assembled.trim()) return null;

  const start = findCodeStart(assembled);
  if (start === null && !looksLikeCode(assembled)) return null;

  const raw = (start === null ? assembled : assembled.slice(start))
    .replace(/^[ \t]+/m, "")
    .trim();
  if (!raw) return null;

  const lines = dedent(raw.split("\n").map((line) => line.trimEnd()));
  const value = lines.join("\n").trim();
  if (!value || value.length > MAX_COMMAND_BLOCK_CHARS) return null;

  const lineCount = lines.filter((line) => line.trim().length > 0).length;
  if (lineCount === 0 || lineCount > MAX_COMMAND_BLOCK_ROWS) return null;
  if (start === null && !looksLikeCode(value)) return null;

  return {
    kind: "command",
    value,
    text: value.split("\n")[0] ?? value,
    startRow: rows[0]?.row ?? bufferRow,
    endRow: rows[rows.length - 1]?.row ?? bufferRow,
    lineCount,
  };
}
