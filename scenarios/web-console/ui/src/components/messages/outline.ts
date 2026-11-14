/** What a long reply contains, shown in its collapsed footer. */
export interface MessageOutline {
  words: number;
  headings: number;
  codeBlocks: number;
}

const FENCE = /^\s*(```|~~~)/;
const HEADING = /^#{1,6}\s/;

/**
 * Counts words, markdown headings, and fenced code blocks. Lines inside a
 * fence are never headings, so a `# comment` in code does not count.
 */
export function messageOutline(text: string): MessageOutline {
  let headings = 0;
  let codeBlocks = 0;
  let inFence = false;
  for (const line of text.split("\n")) {
    if (FENCE.test(line)) {
      if (!inFence) codeBlocks += 1;
      inFence = !inFence;
      continue;
    }
    if (!inFence && HEADING.test(line)) headings += 1;
  }
  const trimmed = text.trim();
  return { words: trimmed ? trimmed.split(/\s+/).length : 0, headings, codeBlocks };
}

const PREVIEW_CHAR_LIMIT = 6000;
const PREVIEW_LINE_LIMIT = 80;

/**
 * A collapsed row needs a few screens of source, not the whole reply. Keep
 * complete fenced blocks interactive; a fence cut by the budget is a plain
 * excerpt so its Copy button cannot silently copy an incomplete code block.
 * The event itself remains the source for the reader and all message actions.
 */
export function messagePreview(text: string): { source: string; markdown: string; plainText: string; truncated: boolean } {
  const lines = text.slice(0, PREVIEW_CHAR_LIMIT).split("\n");
  const prefix = lines.slice(0, PREVIEW_LINE_LIMIT).join("\n");
  if (prefix.length === text.length) return { source: text, markdown: text, plainText: "", truncated: false };

  let fence: { marker: string; start: number; contentStart: number } | null = null;
  let offset = 0;
  for (const line of prefix.split("\n")) {
    const match = /^ {0,3}(`{3,}|~{3,})(.*)$/.exec(line);
    if (match) {
      const marker = match[1] ?? "";
      const tail = match[2] ?? "";
      if (fence) {
        if (marker[0] === fence.marker[0] && marker.length >= fence.marker.length && !tail.trim()) fence = null;
      } else if (marker[0] !== "`" || !tail.includes("`")) {
        fence = { marker, start: offset, contentStart: offset + line.length + 1 };
      }
    }
    offset += line.length + 1;
  }
  return fence
    ? { source: prefix, markdown: prefix.slice(0, fence.start), plainText: prefix.slice(fence.contentStart), truncated: true }
    : { source: prefix, markdown: prefix, plainText: "", truncated: true };
}
