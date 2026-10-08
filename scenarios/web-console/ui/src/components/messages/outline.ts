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
