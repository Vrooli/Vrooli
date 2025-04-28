/**
 * Public ID implementation
 * 
 * Human-friendly short IDs for public URLs. These IDs are:
 * - URL-friendly (alphanumeric)
 * - Short (10-12 characters by default)
 * - Unique enough for public URLs
 * 
 * Uses the nanoid library internally.
 */
import { customAlphabet } from "nanoid";

// Constants
export const PUBLIC_ID_LENGTH = 12;
export const MIN_PUBLIC_ID_LENGTH = 10;
export const MAX_PUBLIC_ID_LENGTH = 12;
// URL-friendly alphabet (numbers and lowercase letters only)
export const ALPHABET = "0123456789abcdefghijklmnopqrstuvwxyz";

// Create a customized nanoid generator
const nanoid = customAlphabet(ALPHABET, PUBLIC_ID_LENGTH);

/**
 * Generate a human-friendly public ID for URLs
 * @param length Optional length (default: 12)
 * @returns A URL-friendly ID string
 */
export function generatePublicId(length: number = PUBLIC_ID_LENGTH): string {
    if (length !== PUBLIC_ID_LENGTH) {
        return customAlphabet(ALPHABET, length)();
    }
    return nanoid();
}

/**
 * Validates if a string is a valid public ID
 * @param id The ID to validate
 * @returns boolean
 */
export function validatePublicId(id: unknown): boolean {
    if (typeof id !== "string") return false;

    // Public IDs should only contain characters from our alphabet
    const regex = new RegExp(`^[${ALPHABET}]+$`);
    if (!regex.test(id)) return false;

    // Length should be between 10-12 characters
    return id.length >= MIN_PUBLIC_ID_LENGTH && id.length <= MAX_PUBLIC_ID_LENGTH;
}
