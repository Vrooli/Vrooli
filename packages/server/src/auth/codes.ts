import { createHash, randomBytes } from "crypto";

const DEFAULT_RANDOM_STRING_LENGTH = 64;
const MAX_RANDOM_STRING_LENGTH = 2048;
const MIN_RANDOM_STRING_CHARSET_LENGTH = 2;
const MAX_RANDOM_STRING_CHARSET_LENGTH = 256;

/**
 * Generate a random string of the specified length, consisting of the specified characters
 * @param length The length of sting to generate
 * @param chars The available characters to use in the string
 * @returns A random string of the specified length, consisting of the specified characters
 */
export function randomString(
    length = DEFAULT_RANDOM_STRING_LENGTH,
    chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789",
): string {
    // Check for valid parameters
    if (!Number.isInteger(length) || length <= 0 || length > MAX_RANDOM_STRING_LENGTH) {
        throw new Error(`Length must be bewteen 1 and ${MAX_RANDOM_STRING_LENGTH}.`);
    }
    const charsLength = chars.length;
    if (typeof chars !== "string" || charsLength < MIN_RANDOM_STRING_CHARSET_LENGTH || chars.length > MAX_RANDOM_STRING_CHARSET_LENGTH) {
        throw new Error(`Chars must be bewteen ${MIN_RANDOM_STRING_CHARSET_LENGTH} and ${MAX_RANDOM_STRING_CHARSET_LENGTH} characters long.`);
    }
    // Generate random bytes
    const bytes = randomBytes(length);
    // Create result array
    const result = new Array(length);
    // Fill result array with bytes, modified to consist of the specified characters
    let cursor = 0;
    for (let i = 0; i < length; i++) {
        cursor += bytes[i];
        result[i] = chars[cursor % charsLength];
    }
    // Return result as string
    return result.join("");
}

/**
 * Verifies if a code is valid and not expired
 * @param providedCode The code to check
 * @param storedCode The code to check against
 * @param dateRequested Date that code was issued
 * @param timeoutInMs Timeout in milliseconds
 * @returns Boolean indicating if the code is valid
 */
export function validateCode(
    providedCode: string | null,
    storedCode: string | null,
    dateRequested: Date | null,
    timeoutInMs: number,
): boolean {
    // Check for valid codes
    if (!providedCode || !storedCode || !dateRequested) return false;
    // Check for valid timeout
    if (!Number.isInteger(timeoutInMs) || timeoutInMs <= 0) return false;
    // Check that codes match
    if (providedCode !== storedCode) return false;
    // Check that code is not expired
    return Date.now() - new Date(dateRequested).getTime() < timeoutInMs;
}

/**
 * Non-cryptographic hashing function for strings
 * @param str The string to hash
 * @returns The hashed string
 */
export function hashString(str: string): string {
    const hash = createHash("md5");
    hash.update(str);
    return hash.digest("hex");
}

