import { exists, isObject } from "@vrooli/shared";
import fs from "fs";
import { logger } from "../events/logger.js";

// AI_CHECK: TYPE_SAFETY=server-type-safety-fixes | LAST: 2025-07-02 - Replaced unsafe any types with unknown for better type safety

// Simple flatten function since lodash.flatten is not available
function flatten(arr: unknown[]): unknown[] {
    return arr.reduce((flat: unknown[], item: unknown) => {
        return flat.concat(Array.isArray(item) ? flatten(item) : item);
    }, [] as unknown[]);
}

let profanity: string[] = [];
let profanityRegex: RegExp | undefined;

/**
 * Resets the profanity state (for testing)
 */
export function resetProfanityState() {
    profanity = [];
    profanityRegex = undefined;
}

/**
 * Initializes the profanity array and regex
 */
export function initializeProfanity() {
    const profanityFile = `${process.env.PROJECT_DIR}/packages/server/dist/utils/censorDictionary.txt`;
    try {
        const fileContent = fs.readFileSync(profanityFile, "utf8");
        profanity = fileContent.toString().split("\n").filter(word => word.trim().length > 0);
        // Add word boundaries to prevent matching parts of words (e.g. "document" contains "cum", but shouldn't be censored)
        if (profanity.length > 0) {
            profanityRegex = new RegExp(profanity.map(word => `\\b${word.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")}\\b`).join("|"), "gi");
        } else {
            profanityRegex = /(?!.*)/; // Never matches anything
        }
    } catch (error) {
        logger.error(`Could not find or read profanity file at ${profanityFile}`, { trace: "0634" });
        profanity = [];
        profanityRegex = /(?!.*)/; // Never matches anything
    }
}

/**
 * Determines if a string contains any banned words
 * @param text The text which may contain bad words
 * @returns True if any bad words were found
 */
export function hasProfanity(...text: (string | null | undefined)[]): boolean {
    if (!profanityRegex) return false;
    return text.some(t => exists(t) && profanityRegex && t.search(profanityRegex) !== -1);
}

/**
 * Recursively converts an items to an array of its string values
 * @param item The item to convert
 * @param fields The fields to convert (supports dot notation). If not specified, all fields will be converted
 * @returns An array of strings
 */
export function toStringArray(item: unknown, fields: string[] | null): string[] | null {
    // Check if item is array
    if (Array.isArray(item)) {
        // Recursively convert each item in the array
        return flatten(item.map(i => toStringArray(i, fields))).filter(i => i !== null) as string[];
    }
    // Check if item is object (not a Date)
    else if (isObject(item) && Object.prototype.toString.call(item) !== "[object Date]") {
        const childFields = fields ? fields.map(s => s.split(".").slice(1).join(".")).filter(s => s.length > 0) : null;
        // Filter out fields that are not specified
        const valuesToCheck: unknown[] = [];
        for (const [key, value] of Object.entries(item)) {
            if (fields && !fields.includes(key)) continue;
            valuesToCheck.push(value);
        }
        // Recursively convert each value in the object
        return flatten(valuesToCheck.map(i => toStringArray(i, childFields))).filter(i => i !== null) as string[];
    }
    // Check if item is string
    else if (typeof item === "string") {
        // Return the item
        return [item];
    }
    return null;
}

/**
 * Removes profanity from a string
 * @param text The text to censor
 * @param censorCharacter The character to replace profanity with
 * @returns The censored text
 */
export function filterProfanity(text: string, censorCharacter = "*"): string {
    if (!profanityRegex) return text;
    return text.replace(profanityRegex, (s: string) => {
        return censorCharacter.repeat(s.length);
    });
}
