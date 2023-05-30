import stopword from "stopword";

/**
 * Process a string according to the rules described in the assignment
 */
const processString = (str: string, language: string, minimalProcessing: boolean): string => {
    // Trim whitespace, remove extra spaces and newlines
    str = str.replace(/\s+/g, " ").trim();
    // Lowercase the string
    str = str.toLowerCase();
    // Limit the Unicode range depending on the language
    // This will remove any character not in the basic multilingual plane
    str = Array.from(str).filter(ch => ch.charCodeAt(0) <= 0xFFFF).join("");
    if (!minimalProcessing) {
        // Split the string into words
        let words = str.split(" ");
        // Remove stop words
        try {
            words = stopword.removeStopwords(words, stopword[language]);
        } catch (e) {
            throw new Error(`Stopwords for ${language} are not available`);
        }
        // Remove punctuation around words, but not inside words
        words = words.map(word => word.replace(/[\p{P}\p{S}](?!\p{L})|(?<!\p{L})[\p{P}\p{S}]/gu, ""));
        // Remove duplicate words
        words = Array.from(new Set(words));
        // Join the words with a comma and space
        // str = words.join(", ");
        str = words.join(" ");
    }
    // Limit the length of the field to 128 characters
    if (str.length > 128) {
        str = str.substring(0, 128);
    }
    return str;
};


/**
 * Shapes a string or object to be used as an embedding object. 
 * This includes: 
 * - Trimming whitespace, removing extra spaces, and removing newlines
 * - Lowercasing the string and limiting the unicode range depending on the language (including removing emojis and diacritics)
 * - Removing stopwords
 * - Removing punctuation around words, but not inside words
 * - Removing duplicate words
 * - Joining the words with a comma and space
 * - Limiting the length of each field to 128 characters
 * @param input The string or object to be shaped. If object, every value must be a string.
 * @param language The language code to use for shaping the string.
 * @param minimalProcessing If set to true, only trim, lowercasing, and character limit are performed. 
 * Use https://github.com/Vrooli/text-embedder-tests to test which processing steps are necessary for your target situation.
 */
export const getEmbeddableString = (
    input: string | Record<string, any>,
    language: string,
    minimalProcessing = false,
): string => {
    let processedInput: Record<string, any> | string;

    if (typeof input === "string") {
        processedInput = processString(input, language, minimalProcessing);
    } else if (typeof input === "object" && input !== null) {
        processedInput = {};

        for (const key in input) {
            if (typeof input[key] === "string") {
                processedInput[key] = processString(input[key], language, minimalProcessing);
            } else if (Array.isArray(input[key])) {
                processedInput[key] = input[key].map(el => typeof el === "string" ? processString(el, language, minimalProcessing) : el);
            }
        }
    } else {
        throw new Error("Invalid input: input must be a string, an array of strings, or an object.");
    }

    return JSON.stringify(processedInput);
};
