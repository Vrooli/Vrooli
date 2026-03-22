// DOC: docs/internal/SEAMS.md#voice-command-parser-seam
//
// Voice Command Parser — detects and parses voice commands from segment-final text.
//
// All command detection logic is centralized here. The parser:
// 1. Strips the configurable prefix from the text
// 2. Fuzzy-matches the remainder against known command patterns
// 3. Extracts numeric arguments (e.g., "tab 3" → number: 3)
//
// Command detection runs ONLY on segment-final text (never on partials)
// to ensure we're parsing the highest-quality transcription.

import { VOICE_COMMANDS, type VoiceCommand } from "./commands";

export interface ParsedCommand {
  command: VoiceCommand;
  confidence: number;
  rawText: string;
  args: Record<string, unknown>;
}

/**
 * Compute Levenshtein edit distance between two strings.
 * Used for fuzzy matching voice transcriptions against command patterns.
 */
export function levenshtein(a: string, b: string): number {
  const m = a.length;
  const n = b.length;
  if (m === 0) return n;
  if (n === 0) return m;

  // Use two rows instead of full matrix for O(min(m,n)) space
  let prev = new Array<number>(n + 1);
  let curr = new Array<number>(n + 1);
  for (let j = 0; j <= n; j++) prev[j] = j;

  for (let i = 1; i <= m; i++) {
    curr[0] = i;
    for (let j = 1; j <= n; j++) {
      const cost = a[i - 1] === b[j - 1] ? 0 : 1;
      curr[j] = Math.min(
        (prev[j] ?? 0) + 1,       // deletion
        (curr[j - 1] ?? 0) + 1,   // insertion
        (prev[j - 1] ?? 0) + cost, // substitution
      );
    }
    [prev, curr] = [curr, prev];
  }
  return prev[n] ?? m;
}

/** Word-to-number mapping for extracting numeric arguments from voice input. */
const WORD_NUMBERS: Record<string, number> = {
  one: 1, two: 2, three: 3, four: 4, five: 5,
  six: 6, seven: 7, eight: 8, nine: 9, ten: 10,
  first: 1, second: 2, third: 3, fourth: 4, fifth: 5,
  sixth: 6, seventh: 7, eighth: 8, ninth: 9, tenth: 10,
};

/**
 * Extract a number from the text following a command pattern.
 * Handles both digits ("tab 3") and words ("tab three").
 */
function extractNumber(text: string): number | null {
  // Try digit match
  const digitMatch = text.match(/\d+/);
  if (digitMatch) {
    const n = parseInt(digitMatch[0], 10);
    if (n > 0 && n <= 100) return n;
  }
  // Try word match
  const words = text.toLowerCase().split(/\s+/);
  for (const word of words) {
    const n = WORD_NUMBERS[word];
    if (n !== undefined) return n;
  }
  return null;
}

/**
 * Parse a segment-final transcript for a voice command.
 *
 * @param text - The segment-final transcription text
 * @param prefix - The command prefix to detect (e.g., "hey do")
 * @returns Parsed command with confidence, or null if no command detected
 */
export function parseCommand(text: string, prefix: string): ParsedCommand | null {
  if (!text || !prefix) return null;

  const normalizedText = text.toLowerCase().trim();
  const normalizedPrefix = prefix.toLowerCase().trim();

  // Check if text starts with the prefix
  if (!normalizedText.startsWith(normalizedPrefix)) return null;

  // Strip prefix and trim
  const remainder = normalizedText.slice(normalizedPrefix.length).trim();
  if (!remainder) return null;

  let bestMatch: { command: VoiceCommand; distance: number; pattern: string; patternLen: number; args: Record<string, unknown> } | null = null;

  for (const command of VOICE_COMMANDS) {
    for (const pattern of command.patterns) {
      const remainderWords = remainder.split(/\s+/);
      const patternWords = pattern.split(/\s+/);

      // Try matching the pattern as a prefix of the remainder (for commands with arguments)
      let textToMatch = remainder;
      let trailingText = "";

      if (remainderWords.length > patternWords.length) {
        const candidateMatch = remainderWords.slice(0, patternWords.length).join(" ");
        trailingText = remainderWords.slice(patternWords.length).join(" ");
        const prefixDist = levenshtein(candidateMatch, pattern);
        if (prefixDist <= maxEditDistance(pattern)) {
          textToMatch = candidateMatch;
        }
      }

      const distance = levenshtein(textToMatch, pattern);
      const maxDist = maxEditDistance(pattern);

      if (distance <= maxDist) {
        // Prefer longer pattern matches (e.g., "stop listening" over "stop")
        // to avoid shorter commands shadowing longer ones.
        const isBetter = !bestMatch
          || pattern.length > bestMatch.patternLen
          || (pattern.length === bestMatch.patternLen && distance < bestMatch.distance);
        if (isBetter) {
          const args: Record<string, unknown> = {};
          if (trailingText) {
            const num = extractNumber(trailingText);
            if (num !== null) args.number = num;
          }
          bestMatch = { command, distance, pattern, patternLen: pattern.length, args };
        }
      }
    }
  }

  if (!bestMatch) return null;

  // Confidence: 1.0 for exact match, decreasing with edit distance
  const patternLen = bestMatch.pattern.length;
  const confidence = patternLen > 0
    ? Math.max(0, 1 - bestMatch.distance / patternLen)
    : 0;

  // Minimum confidence threshold
  if (confidence < 0.5) return null;

  return {
    command: bestMatch.command,
    confidence,
    rawText: text,
    args: bestMatch.args,
  };
}

/**
 * Maximum allowed edit distance for a pattern, scaled by pattern length.
 * Short patterns (≤ 5 chars) allow 1 edit; longer patterns allow 2.
 */
function maxEditDistance(pattern: string): number {
  return pattern.length <= 5 ? 1 : 2;
}

/**
 * Check if a partial transcript contains the command prefix.
 * Used as a hint to temporarily reduce the segment silence threshold.
 * This is intentionally loose — false positives are acceptable since
 * actual command detection only runs on segment finals.
 */
export function partialContainsPrefix(partial: string, prefix: string): boolean {
  if (!partial || !prefix) return false;
  return partial.toLowerCase().includes(prefix.toLowerCase());
}
