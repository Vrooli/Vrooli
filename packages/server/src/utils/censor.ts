import { CustomError } from '../error';
import fs from 'fs';
import { CODE } from '@shared/consts';
import { isObject } from '@shared/utils';
import { genErrorCode } from '../logger';
import pkg from 'lodash';
const { flatten } = pkg;

const profanity = fs.readFileSync(`${process.env.PROJECT_DIR}/packages/server/src/utils/censorDictionary.txt`).toString().split("\n");
// Add spacing around words (e.g. "document" contains "cum", but shouldn't be censored)
const profanityRegex = new RegExp(profanity.map(word => `(?=\\b)${word}(?=\\b)`).join('|'), 'gi');

/**
 * Determines if a string contains any banned words
 * @param text The text which may contain bad words
 * @returns True if any bad words were found
 */
export const hasProfanity = (...text: (string | null | undefined)[]): boolean => {
    return text.some(t => t !== null && t !== undefined && t.search(profanityRegex) !== -1);
}

/**
 * Recursively converts an items to an array of its string values
 * @param item The item to convert
 * @param fields The fields to convert (supports dot notation). If not specified, all fields will be converted
 * @returns An array of strings
 */
export const toStringArray = (item: any, fields: string[] | null): string[] | null => {
    // Check if item is array
    if (Array.isArray(item)) {
        // Recursively convert each item in the array
        return flatten(item.map(i => toStringArray(i, fields))).filter(i => i !== null) as string[];
    }
    // Check if item is object (not a Date)
    else if (isObject(item) && Object.prototype.toString.call(item) !== '[object Date]') {
        const childFields = fields ? fields.map(s => s.split('.').slice(1).join('.')).filter(s => s.length > 0) : null;
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
    else if (typeof item === 'string') {
        // Return the item
        return [item];
    }
    return null
}

/**
 * Throws an error if any string/object contains any banned words
 * @param items The items to check
 * @param fields The fields to check (supports dot notation). If not specified, all fields will be checked
 */
export const validateProfanity = (items: any[], fields: string[] | null = null): void => {
    // Convert items to strings. For objects, recursively convert values to strings
    let strings: string[] = flatten(items.map(i => toStringArray(i, fields))).filter(i => i !== null) as string[];
    if (hasProfanity(...strings))
        throw new CustomError(CODE.BannedWord, 'Banned word detected', { code: genErrorCode('0042') });
}

/**
 * Removes profanity from a string
 * @param text The text to censor
 * @param censorCharacter The character to replace profanity with
 * @returns The censored text
 */
export const filterProfanity = (text: string, censorCharacter: string = '*'): string => {
    return text.replace(profanityRegex, (s: string) => {
        var i = 0;
        var asterisks = '';

        while (i < s.length) {
            asterisks += censorCharacter;
            i++;
        }

        return asterisks;
    });
}