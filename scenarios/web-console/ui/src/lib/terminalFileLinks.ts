import type { IBufferRange, ILink } from "@xterm/xterm";

export interface TerminalFileLink {
  text: string;
  path: string;
  range: IBufferRange;
}

// Keep this intentionally conservative. A token must look like a path and
// have either a separator, a dot-prefixed segment, or a known file extension;
// ordinary shell prose should never become interactive by accident.
const FILE_TOKEN = /(?:\/[^\s"'`]+|(?:\.\.?[\\/]\s*)?[A-Za-z0-9_.-]+(?:[\\/][A-Za-z0-9_.-]+)+|(?:\.\.?[\\/]\s*)?[A-Za-z0-9_.-]+\.[A-Za-z][A-Za-z0-9_-]{0,31})(?:[:(]\d+(?::\d+)?\)?)?/g;

function trimTrailingPunctuation(value: string): string {
  return value.replace(/[),.;!?]+$/, "");
}

export function findTerminalFileLinks(text: string, bufferLineNumber: number): TerminalFileLink[] {
  const links: TerminalFileLink[] = [];
  FILE_TOKEN.lastIndex = 0;
  for (const match of text.matchAll(FILE_TOKEN)) {
    const raw = match[0];
    const token = trimTrailingPunctuation(raw);
    const pathEnd = token.search(/[:(]\d+(?::\d+)?\)?$/);
    const path = pathEnd >= 0 ? token.slice(0, pathEnd) : token;
    if (!path || path === "/" || path.startsWith("//")) continue;

    const locationMatch = token.match(/[:(](\d+)(?::(\d+))?\)?$/);
    const location = locationMatch ? token.slice(token.length - locationMatch[0].length).replace("(", ":").replace(")", "") : "";
    const pathWithLocation = `${path}${location}`;
    const startX = match.index + 1;
    const endX = startX + token.length - 1;
    links.push({
      text: token,
      path: pathWithLocation,
      range: {
        start: { x: startX, y: bufferLineNumber },
        end: { x: endX, y: bufferLineNumber },
      },
    });
  }
  return links;
}

export function terminalFileLinksForLine(
  text: string,
  bufferLineNumber: number,
  activate: (path: string) => void,
  stringOffsetToCell?: (offset: number) => number,
): ILink[] {
  return findTerminalFileLinks(text, bufferLineNumber).map((link) => ({
    range: {
      start: {
        x: (stringOffsetToCell?.(link.range.start.x - 1) ?? link.range.start.x - 1) + 1,
        y: link.range.start.y,
      },
      end: {
        x: (stringOffsetToCell?.(link.range.end.x - 1) ?? link.range.end.x - 1) + 1,
        y: link.range.end.y,
      },
    },
    text: link.text,
    decorations: { pointerCursor: true, underline: true },
    activate: () => { activate(link.path); },
  }));
}
