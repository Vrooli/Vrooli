import type { Terminal } from "@xterm/xterm";

/** A detected link, independent of where it sits on the grid. */
export interface TerminalLink {
  /** "file" opens the in-app preview; "url" opens in a new tab. */
  kind: "file" | "url";
  /** For "file", the path (optionally `:line:col`); for "url", the full URI. */
  value: string;
  /** The visible token text, shown as the menu title. */
  text: string;
}

/** A detected link plus its character offset within the source string. */
export interface TerminalLinkMatch extends TerminalLink {
  startOffset: number;
}

/** What the user activated, with the pointer position to anchor the menu. */
export interface TerminalLinkActivation extends TerminalLink {
  position: { x: number; y: number };
}

// Keep path matching intentionally conservative. A token must look like a path
// and have either a separator, a dot-prefixed segment, or a known file
// extension; ordinary shell prose should never become interactive by accident.
const FILE_TOKEN = /(?:\/[^\s"'`]+|(?:\.\.?[\\/]\s*)?[A-Za-z0-9_.-]+(?:[\\/][A-Za-z0-9_.-]+)+|(?:\.\.?[\\/]\s*)?[A-Za-z0-9_.-]+\.[A-Za-z][A-Za-z0-9_-]{0,31})(?:[:(]\d+(?::\d+)?\)?)?/g;

// URLs are matched explicitly so a path embedded in a URL (e.g. the pathname of
// an https:// link) does not become a second, competing file link.
const URL_TOKEN = /(?:https?|file):\/\/[^\s"'`<>]+/gi;

function trimTrailingPunctuation(value: string): string {
  return value.replace(/[),.;!?'"`]+$/, "");
}

/** Match URLs within a string, reporting each token's offset. */
function matchTerminalUrls(text: string): TerminalLinkMatch[] {
  const matches: TerminalLinkMatch[] = [];
  URL_TOKEN.lastIndex = 0;
  for (const match of text.matchAll(URL_TOKEN)) {
    const token = trimTrailingPunctuation(match[0]);
    if (!token) continue;
    // A file:// URI resolves to the same in-app preview as a bare path.
    const kind = /^file:/i.test(token) ? "file" : "url";
    matches.push({ kind, value: token, text: token, startOffset: match.index });
  }
  return matches;
}

/** Match file paths within a string, reporting each token's offset. */
export function matchTerminalFilePaths(text: string): TerminalLinkMatch[] {
  const matches: TerminalLinkMatch[] = [];
  FILE_TOKEN.lastIndex = 0;
  for (const match of text.matchAll(FILE_TOKEN)) {
    const token = trimTrailingPunctuation(match[0]);
    const pathEnd = token.search(/[:(]\d+(?::\d+)?\)?$/);
    const path = pathEnd >= 0 ? token.slice(0, pathEnd) : token;
    if (!path || path === "/" || path.startsWith("//")) continue;

    const locationMatch = token.match(/[:(](\d+)(?::(\d+))?\)?$/);
    const location = locationMatch ? token.slice(token.length - locationMatch[0].length).replace("(", ":").replace(")", "") : "";
    matches.push({ kind: "file", value: `${path}${location}`, text: token, startOffset: match.index });
  }
  return matches;
}

/**
 * Detect both URLs and file paths in a string. URLs win any overlap so a path
 * inside a URL is not reported twice. Results are ordered by offset.
 */
export function matchTerminalLinks(text: string): TerminalLinkMatch[] {
  const urls = matchTerminalUrls(text);
  const covered = (offset: number, length: number): boolean =>
    urls.some((u) => offset < u.startOffset + u.text.length && offset + length > u.startOffset);
  const paths = matchTerminalFilePaths(text).filter((p) => !covered(p.startOffset, p.text.length));
  return [...urls, ...paths].sort((a, b) => a.startOffset - b.startOffset);
}

// ---------------------------------------------------------------------------
// Grid walking: a path/URL can span rows, and a TUI (codex) wraps it in two
// ways that defeat xterm's isWrapped flag — it breaks mid-token a few columns
// short of the terminal edge (reserving a right margin) and indents the
// continuation (a hanging indent). So instead of trusting isWrapped or the
// terminal width, we estimate the wrap column from the content itself and walk
// the contiguous non-space run outward from the tapped cell, skipping the
// hanging indent when we bridge to the next row. The URL/path regexes then
// extract the real token from the assembled run.
// ---------------------------------------------------------------------------

/** Rows scanned up/down when bridging a wrapped token. */
const BLOCK_SCAN_BUDGET = 128;

/** Columns a TUI may leave unused at the right edge before wrapping. A row whose
 *  content reaches within this many columns of term.cols is treated as wrapped;
 *  a shorter row ended on its own. Tuned to codex, which wraps ~2 columns shy. */
const WRAP_MARGIN = 3;

/** A run character is anything that could belong to a path or URL: not
 *  whitespace and not a quote/backtick that would bound the token. */
function isRunChar(ch: string): boolean {
  return ch !== "" && !/[\s"'`]/.test(ch);
}

/** The row's content, trailing whitespace trimmed (leading indent preserved),
 *  so its length is the column just past the last glyph. */
function rowText(term: Terminal, row: number): string {
  const line = term.buffer.active.getLine(row);
  return line ? line.translateToString(true) : "";
}

/**
 * Whether `row` ends at this block's own wrap column rather than the terminal
 * edge. A styled TUI panel (e.g. Claude Code's queued-input box) fills each line
 * to the box width — many columns shy of term.cols — before wrapping, so the
 * geometric "near term.cols" gate misses it. A real hard wrap leaves evidence:
 * another contiguous row of the same block ends at the *same* column. A lone
 * short line has no such corroborator, so it is not mistaken for a wrap. We scan
 * outward over contiguous non-blank rows; a blank line bounds the block, and a
 * longer neighbour proves `row` was itself short (its end is not the wrap width).
 */
function endsAtBlockWrapColumn(term: Terminal, row: number): boolean {
  const target = rowText(term, row).length;
  if (target === 0) return false;
  for (const dir of [-1, 1]) {
    for (let step = 1; step < BLOCK_SCAN_BUDGET; step++) {
      const len = rowText(term, row + dir * step).length;
      if (len === 0 || len > target) break; // block boundary, or `row` was the short one
      if (len === target) return true; // a second full line at the same wrap column
    }
  }
  return false;
}

/**
 * Whether `row`'s token continues onto the next row: its content ended at a hard
 * wrap (within WRAP_MARGIN of the terminal edge, OR at a block-local wrap column
 * shared by a sibling row — not a short line that merely ended) and both sides of
 * the seam are run characters, so a single token was split there. The term.cols
 * test handles full-width output; the block-wrap-column test handles indented or
 * margin-reserving TUI panels that wrap well short of the edge. Requiring a
 * corroborating sibling row stops two lone short lines from being joined just
 * because the first is the longest.
 */
function wrapsIntoNext(term: Terminal, row: number): boolean {
  const text = rowText(term, row);
  if (text.length === 0) return false;
  const reachedEdge = text.length >= term.cols - WRAP_MARGIN;
  if (!reachedEdge && !endsAtBlockWrapColumn(term, row)) return false;
  const next = rowText(term, row + 1);
  const firstNonSpace = next.search(/\S/);
  if (firstNonSpace < 0) return false;
  return isRunChar(text.charAt(text.length - 1)) && isRunChar(next.charAt(firstNonSpace));
}

/**
 * Find the link at (zero-based) grid cell (bufferRow, col). Walks the contiguous
 * run outward from the tap, bridging TUI hard wraps (skipping the hanging
 * indent), then extracts the path/URL token covering the tap. `margin` widens
 * the hit area so an imprecise tap still lands — essential on touch.
 */
export function linkAtCell(term: Terminal, bufferRow: number, col: number, margin = 1): TerminalLink | null {
  const pivotText = rowText(term, bufferRow);
  // Snap the tap onto a run character within the forgiving margin.
  let startCol = -1;
  for (let d = 0; d <= margin && startCol < 0; d++) {
    if (isRunChar(pivotText.charAt(col + d))) startCol = col + d;
    else if (d > 0 && isRunChar(pivotText.charAt(col - d))) startCol = col - d;
  }
  if (startCol < 0) return null;

  // Walk left to the start of the run, bridging up through wrapped rows.
  let sRow = bufferRow;
  let sCol = startCol;
  for (let guard = 0; guard < BLOCK_SCAN_BUDGET; guard++) {
    const text = rowText(term, sRow);
    while (sCol - 1 >= 0 && isRunChar(text.charAt(sCol - 1))) sCol--;
    const firstNonSpace = text.search(/\S/);
    if (sCol === firstNonSpace && sRow > 0 && wrapsIntoNext(term, sRow - 1)) {
      sRow -= 1;
      sCol = rowText(term, sRow).length; // one past end; next pass walks left into it
      continue;
    }
    break;
  }

  // Walk right to the end of the run, assembling the string and tracking where
  // the tapped cell falls so the right token can be chosen from a multi-run.
  const parts: string[] = [];
  let pivotOffset = 0;
  let assembled = 0;
  let eRow = sRow;
  let eCol = sCol;
  for (let guard = 0; guard < BLOCK_SCAN_BUDGET; guard++) {
    const text = rowText(term, eRow);
    let c = eCol;
    while (c < text.length && isRunChar(text.charAt(c))) c++;
    if (eRow === bufferRow && startCol >= eCol && startCol < c) {
      pivotOffset = assembled + (startCol - eCol);
    }
    parts.push(text.slice(eCol, c));
    assembled += c - eCol;
    if (c >= text.length && wrapsIntoNext(term, eRow)) {
      eRow += 1;
      const nextFirst = rowText(term, eRow).search(/\S/);
      eCol = nextFirst < 0 ? 0 : nextFirst;
      continue;
    }
    break;
  }

  const run = parts.join("");
  if (!run) return null;

  const links = matchTerminalLinks(run);
  if (links.length === 0) return null;
  // Prefer the token covering the tap; fall back to the longest match.
  const covering = links.find((l) => pivotOffset >= l.startOffset - margin && pivotOffset <= l.startOffset + l.text.length + margin);
  const chosen = covering ?? links.reduce((a, b) => (b.text.length > a.text.length ? b : a));
  return { kind: chosen.kind, value: chosen.value, text: chosen.text };
}
