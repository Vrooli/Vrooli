/** Matches commit messages ending with ` p<N>`, e.g. "web-console TTS p10" */
const PN_PATTERN = /^(.+)\s+p(\d+)$/;

/**
 * Parse a commit message's continuation group.
 * Returns the topic prefix and part number, or null if not a continuation commit.
 *
 * @example parseCommitGroup("web-console TTS p10") // { prefix: "web-console TTS", part: 10 }
 * @example parseCommitGroup("fix typo")            // null
 */
export function parseCommitGroup(message: string): { prefix: string; part: number } | null {
  const match = message.match(PN_PATTERN);
  if (!match || !match[1] || !match[2]) return null;
  return { prefix: match[1], part: Number(match[2]) };
}

/**
 * Build the next continuation message by incrementing the part number.
 * Returns null if the message doesn't match the pN pattern.
 *
 * @example buildContinueMessage("web-console TTS p10") // "web-console TTS p11"
 * @example buildContinueMessage("fix typo")            // null
 */
export function buildContinueMessage(message: string): string | null {
  const group = parseCommitGroup(message);
  if (!group) return null;
  return `${group.prefix} p${group.part + 1}`;
}
