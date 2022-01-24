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