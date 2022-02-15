import fs from 'fs';

const profanity = fs.readFileSync(`${process.env.PROJECT_DIR}/packages/server/src/utils/censorDictionary.txt`).toString().split("\n");
const profanityRegex = new RegExp(profanity.join('|'), 'gi');

/**
 * Determines if a string contains any banned words
 * @param text The text which may contain bad words
 * @returns True if any bad words were found
 */
export const hasProfanity = (...text: (string | null | undefined)[]): boolean => {
    return text.some(t => t !== null && t !== undefined && t.search(profanityRegex) !== -1);
}

/**
 * Determines if any strings in the object contain any banned words
 * @param obj The object which may contain bad words
 * @params includeKeys The keys of the object which may contain bad words
 * @returns True if any bad words were found
 */
export const hasProfanityRecursive = (obj: { [x: string]: any }, includeKeys: string[] = []): boolean => {
    // Loop through object's keys
    return Object.keys(obj).some(key => {
        // If the key is not one we're interested in, skip
        if (!includeKeys.includes(key)) return false;
        // If the value is an array, check each element
        if (Array.isArray(obj[key])) return obj[key].some((v: any) => {
            // If the key is a string, check if it contains profanity
            if (typeof v === 'string') return hasProfanity(v);
            // If the key is an object, recurse
            if (typeof v === 'object') return hasProfanityRecursive(v, includeKeys);
            return false;
        });
        // If the key is a string, check if it contains profanity
        if (typeof obj[key] === 'string') return hasProfanity(obj[key]);
        // If the key is an object, recurse
        if (typeof obj[key] === 'object') return hasProfanityRecursive(obj[key], includeKeys);
        return false;
    });
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