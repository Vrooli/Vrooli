// DOC: docs/internal/SEAMS.md#voice-command-parser-seam
//
// Voice Command Parser — detects and parses voice commands from transcribed text.
//
// All command detection logic is centralized here. The parser:
// 1. Fuzzy-matches the input text against known command patterns
// 2. Extracts numeric arguments (e.g., "tab 3" → number: 3)
//
// Command detection runs ONLY on segment-final text (never on partials)
// to ensure we're parsing the highest-quality transcription.
//
// Wake word detection happens at the audio level (see wakeword/ module).
// This parser only handles the text-based command identification step.
import { VOICE_COMMANDS } from "./commands";
/**
 * Compute Levenshtein edit distance between two strings.
 * Used for fuzzy matching voice transcriptions against command patterns.
 */
export function levenshtein(a, b) {
    const m = a.length;
    const n = b.length;
    if (m === 0)
        return n;
    if (n === 0)
        return m;
    // Use two rows instead of full matrix for O(min(m,n)) space
    let prev = new Array(n + 1);
    let curr = new Array(n + 1);
    for (let j = 0; j <= n; j++)
        prev[j] = j;
    for (let i = 1; i <= m; i++) {
        curr[0] = i;
        for (let j = 1; j <= n; j++) {
            const cost = a[i - 1] === b[j - 1] ? 0 : 1;
            curr[j] = Math.min((prev[j] ?? 0) + 1, // deletion
            (curr[j - 1] ?? 0) + 1, // insertion
            (prev[j - 1] ?? 0) + cost);
        }
        [prev, curr] = [curr, prev];
    }
    return prev[n] ?? m;
}
/** Word-to-number mapping for extracting numeric arguments from voice input. */
const WORD_NUMBERS = {
    one: 1, two: 2, three: 3, four: 4, five: 5,
    six: 6, seven: 7, eight: 8, nine: 9, ten: 10,
    first: 1, second: 2, third: 3, fourth: 4, fifth: 5,
    sixth: 6, seventh: 7, eighth: 8, ninth: 9, tenth: 10,
};
/**
 * Extract a number from the text following a command pattern.
 * Handles both digits ("tab 3") and words ("tab three").
 */
function extractNumber(text) {
    // Try digit match
    const digitMatch = text.match(/\d+/);
    if (digitMatch) {
        const n = parseInt(digitMatch[0], 10);
        if (n > 0 && n <= 100)
            return n;
    }
    // Try word match
    const words = text.toLowerCase().split(/\s+/);
    for (const word of words) {
        const n = WORD_NUMBERS[word];
        if (n !== undefined)
            return n;
    }
    return null;
}
/**
 * Strip punctuation that Whisper commonly inserts between words (commas,
 * periods, etc.) so that "new tab." normalizes to "new tab"
 * and matches command patterns.
 */
function stripPunctuation(s) {
    return s.replace(/[.,;:!?'"]/g, "").replace(/\s+/g, " ").trim();
}
/**
 * Maximum allowed edit distance for a pattern, scaled by pattern length.
 * Short patterns (≤ 5 chars) allow 1 edit; longer patterns allow 2.
 */
function maxEditDistance(pattern) {
    return pattern.length <= 5 ? 1 : 2;
}
/**
 * Parse transcribed text for a voice command (no prefix needed).
 *
 * This function matches the full input text against known command patterns
 * using fuzzy Levenshtein matching. Wake word detection happens at the
 * audio level before this function is called.
 *
 * @param text - The transcribed text to check for commands.
 * @returns Parsed command with confidence, or null if no command detected.
 */
export function parseCommandDirect(text) {
    if (!text)
        return null;
    const normalizedText = stripPunctuation(text.toLowerCase());
    if (!normalizedText)
        return null;
    let bestMatch = null;
    for (const command of VOICE_COMMANDS) {
        for (const pattern of command.patterns) {
            const textWords = normalizedText.split(/\s+/);
            const patternWords = pattern.split(/\s+/);
            // Try matching the pattern as a prefix of the text (for commands with arguments)
            let textToMatch = normalizedText;
            let trailingText = "";
            if (textWords.length > patternWords.length) {
                const candidateMatch = textWords.slice(0, patternWords.length).join(" ");
                trailingText = textWords.slice(patternWords.length).join(" ");
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
                    const args = {};
                    if (trailingText) {
                        const num = extractNumber(trailingText);
                        if (num !== null)
                            args.number = num;
                    }
                    bestMatch = { command, distance, pattern, patternLen: pattern.length, args };
                }
            }
        }
    }
    if (!bestMatch)
        return null;
    // Confidence: 1.0 for exact match, decreasing with edit distance
    const patternLen = bestMatch.pattern.length;
    const confidence = patternLen > 0
        ? Math.max(0, 1 - bestMatch.distance / patternLen)
        : 0;
    // Minimum confidence threshold
    if (confidence < 0.5)
        return null;
    return {
        command: bestMatch.command,
        confidence,
        rawText: text,
        args: bestMatch.args,
    };
}
