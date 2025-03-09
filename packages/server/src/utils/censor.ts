import { exists, isObject } from "@local/shared";
import fs from "fs";
import pkg from "lodash";
import { logger } from "../events/logger.js";

const { flatten } = pkg;

const profanityFile = `${process.env.PROJECT_DIR}/packages/server/dist/utils/censorDictionary.txt`;

let profanity: string[] = [];
let profanityRegex: RegExp;

/**
 * Initializes the profanity array and regex
 */
export function initializeProfanity() {
    try {
        const fileContent = fs.readFileSync(profanityFile, "utf8");
        profanity = fileContent.toString().split("\n");
        // Add spacing around words (e.g. "document" contains "cum", but shouldn't be censored)
        profanityRegex = new RegExp(profanity.map(word => `(?=\\b)${word}(?=\\b)`).join("|"), "gi");
    } catch (error) {
        logger.error(`Could not find or read profanity file at ${profanityFile}`, { trace: "0634" });
    }
}

/**
 * Determines if a string contains any banned words
 * @param text The text which may contain bad words
 * @returns True if any bad words were found
 */
export function hasProfanity(...text: (string | null | undefined)[]): boolean {
    return text.some(t => exists(t) && t.search(profanityRegex) !== -1);
}

/**
 * Recursively converts an items to an array of its string values
 * @param item The item to convert
 * @param fields The fields to convert (supports dot notation). If not specified, all fields will be converted
 * @returns An array of strings
 */
export function toStringArray(item: any, fields: string[] | null): string[] | null {
    // Check if item is array
    if (Array.isArray(item)) {
        // Recursively convert each item in the array
        return flatten(item.map(i => toStringArray(i, fields))).filter(i => i !== null) as string[];
    }
    // Check if item is object (not a Date)
    else if (isObject(item) && Object.prototype.toString.call(item) !== "[object Date]") {
        const childFields = fields ? fields.map(s => s.split(".").slice(1).join(".")).filter(s => s.length > 0) : null;
        // Filter out fields that are not specified
        const valuesToCheck: any[] = [];
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
    return text.replace(profanityRegex, (s: string) => {
        let i = 0;
        let asterisks = "";

        while (i < s.length) {
            asterisks += censorCharacter;
            i++;
        }

        return asterisks;
    });
}
