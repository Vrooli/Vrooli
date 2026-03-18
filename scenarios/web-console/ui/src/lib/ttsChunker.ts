/**
 * Maximum characters per TTS chunk. Must stay under the backend's
 * maxSynthesizeInputLength (5000) with margin for safety.
 */
export const TTS_MAX_CHUNK_LENGTH = 4500;

/**
 * Returns true if a chunk contains enough speakable content for TTS.
 * Filters out markdown syntax, lone punctuation, code fences, etc.
 * that cause Kokoro to return 0-byte audio.
 */
export function isSpeakable(text: string): boolean {
  // Strip markdown-style formatting: code fences, headings, HRs, list markers
  const stripped = text
    .replace(/^```\w*$/gm, "")     // code fence lines
    .replace(/^#{1,6}\s*/gm, "")   // heading markers (keep text after #)
    .replace(/^[-*_]{3,}\s*$/gm, "")  // horizontal rules
    .replace(/^[\s*>+-]+$/gm, "")  // lone list markers / blockquote markers
    .trim();
  // Must have at least one word character after stripping
  return /\w/.test(stripped);
}

/**
 * Split a message into speakable paragraphs that each fit within the
 * backend's synthesis character limit.
 *
 * Strategy (in order):
 *   1. Split on double-newline boundaries (paragraph breaks)
 *   2. For blocks > 500 chars, split further on single newlines
 *   3. Filter out non-speakable chunks (markdown syntax, lone punctuation)
 *   4. For chunks still over the limit, split on sentence boundaries
 *   5. As a last resort, hard-split at TTS_MAX_CHUNK_LENGTH
 */
export function splitIntoParagraphs(text: string): string[] {
  const raw = text.split(/\n\n+/).filter((p) => p.trim());
  const afterNewlines: string[] = [];
  for (const block of raw) {
    if (block.length > 500) {
      afterNewlines.push(...block.split(/\n/).filter((l) => l.trim()));
    } else {
      afterNewlines.push(block);
    }
  }
  const preliminary = afterNewlines.length > 0 ? afterNewlines : [text];

  // Filter non-speakable chunks and enforce max chunk length
  const result: string[] = [];
  for (const chunk of preliminary) {
    if (!isSpeakable(chunk)) continue;
    if (chunk.length <= TTS_MAX_CHUNK_LENGTH) {
      result.push(chunk);
    } else {
      result.push(...splitLongChunk(chunk));
    }
  }

  return result.length > 0 ? result : [text];
}

/**
 * Split a chunk that exceeds TTS_MAX_CHUNK_LENGTH. Tries sentence
 * boundaries first, then hard-splits as a fallback.
 */
function splitLongChunk(text: string): string[] {
  // Try splitting on sentence boundaries: . ! ? followed by space or end
  const sentences = text.match(/[^.!?]*[.!?]+(?:\s+|$)|[^.!?]+$/g);
  if (sentences && sentences.length > 1) {
    const result: string[] = [];
    let current = "";
    for (const sentence of sentences) {
      if (current.length + sentence.length > TTS_MAX_CHUNK_LENGTH) {
        if (current.trim()) result.push(current.trim());
        // If a single sentence exceeds the limit, hard-split it
        if (sentence.length > TTS_MAX_CHUNK_LENGTH) {
          result.push(...hardSplit(sentence.trim()));
        } else {
          current = sentence;
        }
      } else {
        current += sentence;
      }
    }
    if (current.trim()) {
      if (current.length > TTS_MAX_CHUNK_LENGTH) {
        result.push(...hardSplit(current.trim()));
      } else {
        result.push(current.trim());
      }
    }
    return result;
  }

  // No sentence boundaries found — hard split
  return hardSplit(text);
}

/** Hard-split text at TTS_MAX_CHUNK_LENGTH, preferring word boundaries. */
function hardSplit(text: string): string[] {
  const result: string[] = [];
  let remaining = text;
  while (remaining.length > TTS_MAX_CHUNK_LENGTH) {
    // Try to break at last space within the limit
    let splitAt = remaining.lastIndexOf(" ", TTS_MAX_CHUNK_LENGTH);
    if (splitAt <= 0) {
      splitAt = TTS_MAX_CHUNK_LENGTH;
    }
    result.push(remaining.slice(0, splitAt).trim());
    remaining = remaining.slice(splitAt).trim();
  }
  if (remaining.trim()) {
    result.push(remaining.trim());
  }
  return result;
}
