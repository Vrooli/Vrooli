/** A weak string hash for things like cache keys. */
export function weakHash(str: string) {
    let hash = 0;
    if (str.length === 0) {
        return hash.toString();
    }
    for (let i = 0; i < str.length; i++) {
        const char = str.charCodeAt(i);
        hash = (hash << 5) - hash + char;
        hash = hash & hash; // Convert to 32bit integer
    }
    return hash.toString();
}

export function chatMatchHash(userIds: string[]): string {
    // Sort for consistent ordering
    const toHash = userIds.sort().join("|");
    // Create a  hash of the sortedIDs
    return weakHash(toHash);
}

const DEFAULT_RANDOM_STRING_LENGTH = 7;
const DEFAULT_RANDOM_STRING_RADIX = 36;
const MAX_RANDOM_STRING_LENGTH = 10;
const MIN_RANDOM_RADIX_LENGTH = 2;
const MAX_RANDOM_RADIX_LENGTH = 256;
const RANDOM_STRING_OFFSET = 2; // Offset to remove the "0." from the random number

/**
 * Generate a random string of the specified length, consisting of the specified characters.
 * WARNING: This is not cryptographically secure.
 * @param length The length of sting to generate
 * @param radix The base of the number to use
 * @returns A random string of the specified length, consisting of the specified characters
 */
export function randomString(
    length = DEFAULT_RANDOM_STRING_LENGTH,
    radix = DEFAULT_RANDOM_STRING_RADIX,
): string {
    // Check for valid parameters
    if (!Number.isInteger(length) || length <= 0 || length > MAX_RANDOM_STRING_LENGTH) {
        console.error(`Length must be bewteen 1 and ${MAX_RANDOM_STRING_LENGTH}.`);
        length = DEFAULT_RANDOM_STRING_LENGTH;
    }
    if (!Number.isInteger(radix) || radix < MIN_RANDOM_RADIX_LENGTH || radix > MAX_RANDOM_RADIX_LENGTH) {
        console.error(`Radix must be bewteen ${MIN_RANDOM_RADIX_LENGTH} and ${MAX_RANDOM_RADIX_LENGTH}.`);
        radix = DEFAULT_RANDOM_STRING_RADIX;
    }
    // Generate random bytes
    return Math.random().toString(radix).substring(RANDOM_STRING_OFFSET, length + RANDOM_STRING_OFFSET);
}
