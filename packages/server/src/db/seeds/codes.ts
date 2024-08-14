// Contains built-in Vrooli functions for seeding the database with codes. 
// These are helpful functions which are both used internally (it's faster for us to 
// use these functions directly than to load them from the database and run through 
// the sandbox) and used in routines to transform data.

/**
 * Converts plaintext to inputs and outputs. 
 * Useful to convert an LLM response into missing inputs and outputs for running a routine step.
 * @name parseRunIOFromPlaintext
 * @description When a routine is being run autonomously, we may need to generate inputs and/or outputs before we can perform the action associated with the routine. This function parses the expected output and returns an object with an inputs list and outputs list.
 * @returns An object with inputs and output mappings.
 */
export function parseRunIOFromPlaintext({ formData, text }: { formData: object; text: string }): { inputs: Record<string, string>; outputs: Record<string, string> } {
    const inputs: Record<string, string> = {};
    const outputs: Record<string, string> = {};
    const lines = text.trim().split("\n");

    for (const line of lines) {
        const [key, ...valueParts] = line.split(":");
        if (key && valueParts.length > 0) {
            const trimmedKey = key.trim();
            const value = valueParts.join(":").trim();

            if (Object.prototype.hasOwnProperty.call(formData, `input-${trimmedKey}`)) {
                inputs[trimmedKey] = value;
            } else if (Object.prototype.hasOwnProperty.call(formData, `output-${trimmedKey}`)) {
                outputs[trimmedKey] = value;
            }
            // If the key doesn't match any input or output, it's ignored
        }
    }

    return { inputs, outputs };
}

/**
 * Transforms plaintext containing formatted or plain search terms into a cleaned list.
 * Search terms can be prefaced with numbering (e.g., "1. term") or listed plainly.
 * The function ignores blank or whitespace-only lines and trims whitespace from text lines.
 * @name transformSearchTerms
 * @description Converts formatted or plain plaintext search terms into a cleaned array of terms.
 * @param {string} text - The plaintext containing search terms.
 * @returns {string[]} A cleaned array of search terms.
 */
export function transformSearchTerms(text) {
    const lines = text.split("\n");
    const cleanedTerms: string[] = [];
    let isNumberedList = false;

    // First, determine if the text contains a numbered list
    for (const line of lines) {
        if (line.trim().match(/^\d+\.\s+/)) {
            isNumberedList = true;
            break;
        }
    }

    for (let line of lines) {
        line = line.trim();
        if (line !== "") {
            if (isNumberedList) {
                // For numbered lists, only process lines matching the numbered list format
                const match = line.match(/^\d+\.\s*(.*)$/);
                if (match) {
                    cleanedTerms.push(match[1]);
                }
            } else {
                // If not a numbered list, process all non-empty lines
                cleanedTerms.push(line);
            }
        }
    }

    return cleanedTerms;
}

/**
 * Converts a list of strings into plaintext.
 * Useful for generating plaintext from an array of terms, formatted without numbers.
 * @name listToPlaintext
 * @description Converts an array of strings into a single string, each item separated by a newline.
 * @param {string[]} list - The array of strings to convert.
 * @returns {string} A plaintext string of the list items, separated by newlines.
 */
export function listToPlaintext(list) {
    return list.join("\n");
}

/**
 * Converts a list of strings into a numbered plaintext string.
 * Useful for generating a formatted list where each item is preceded by its index number followed by a dot.
 * @name listToNumberedPlaintext
 * @description Converts an array of strings into a single string, each item prefixed with its number in the list and separated by newlines.
 * @param {string[]} list - The array of strings to convert.
 * @returns {string} A numbered plaintext string of the list items, each prefixed with its number and a dot, separated by newlines.
 */
export function listToNumberedPlaintext(list) {
    return list.map((item, index) => `${index + 1}. ${item}`).join("\n");
}
