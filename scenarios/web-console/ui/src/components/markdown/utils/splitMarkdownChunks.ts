// A large markdown file (the reader accepts up to ~1 MiB) parses and reconciles
// as one tree, which blocks the main thread long enough to feel frozen on open.
// Splitting the source into block-aligned chunks lets the reader render the
// first screenful immediately and append the rest across idle frames, so the
// one long task becomes several short ones.
//
// Splitting is only safe at a blank line outside a fenced code block: markdown
// never puts a blank line inside a table or between the rows of one, so a cut
// there cannot break a table, and the fence check keeps it out of code blocks.
// Loose lists split at a blank line render as two lists — a cosmetic change the
// reader tolerates.

const FENCE = /^\s{0,3}(?:```|~~~)/;
// A link reference definition — `[label]: https://…` — may be used from any
// chunk, so every chunk carries a copy. Duplicate definitions are harmless:
// the first one wins.
const REF_DEF = /^\s{0,3}\[[^\]^][^\]]*\]:\s/;
// A footnote definition can be referenced far from where it sits; scattering it
// across chunks would render it wrong, so a document using footnotes is kept
// whole instead.
const FOOTNOTE_DEF = /^\s{0,3}\[\^[^\]]+\]:/;

export function splitMarkdownChunks(content: string, targetBytes = 8000): string[] {
  if (content.length <= targetBytes) return [content];
  const lines = content.split("\n");
  if (lines.some((line) => FOOTNOTE_DEF.test(line))) return [content];

  const refDefs = lines.filter((line) => REF_DEF.test(line));
  const suffix = refDefs.length ? `\n\n${refDefs.join("\n")}` : "";

  const chunks: string[] = [];
  let current: string[] = [];
  let size = 0;
  let inFence = false;
  const flush = () => {
    if (current.length === 0) return;
    chunks.push(current.join("\n") + suffix);
    current = [];
    size = 0;
  };
  for (const line of lines) {
    if (FENCE.test(line)) inFence = !inFence;
    current.push(line);
    size += line.length + 1;
    if (!inFence && size >= targetBytes && line.trim() === "") flush();
  }
  flush();
  return chunks.length > 0 ? chunks : [content];
}
