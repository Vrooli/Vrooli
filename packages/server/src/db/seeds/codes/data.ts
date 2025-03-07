import { CodeLanguage, CodeType, SEEDED_IDS, SEEDED_TAGS, uuid } from "@local/shared";
import { CodeImportData } from "../../../builders/importExport.js";

const VERSION = "1.0.0" as const;
const CODE_LANGUAGE = CodeLanguage.Javascript;
const CODE_TYPE = CodeType.DataConvert;

const baseRoot = {
    __version: VERSION,
    __typename: "Code" as const,
};

const baseRootShape = {
    isPrivate: false,
    permissions: JSON.stringify({}),
    tags: [{ __typename: "Tag" as const, tag: SEEDED_TAGS.Vrooli }],
};

const baseVersion = {
    __version: VERSION,
    __typename: "CodeVersion" as const,
};

const baseVersionShape = {
    codeLanguage: CODE_LANGUAGE,
    codeType: CODE_TYPE,
    isComplete: true,
    isPrivate: false,
    versionIndex: 0,
    versionLabel: "1.0.0",
};

export const data: CodeImportData[] = [
    {
        ...baseRoot,
        shape: {
            ...baseRootShape,
            id: SEEDED_IDS.Code.parseRunIOFromPlaintext,
            versions: [{
                ...baseVersion,
                shape: {
                    ...baseVersionShape,
                    id: SEEDED_IDS.CodeVersion.parseRunIOFromPlaintext,
                    content: `/**
 * Converts plaintext to inputs and outputs. 
 * Useful to convert an LLM response into missing inputs and outputs for running a routine step.
 * 
 * @returns An object with inputs and output mappings.
 */
function parseRunIOFromPlaintext(
    { formData, text },
) {
    const inputs = {};
    const outputs = {};
    const lines = text.trim().split(String.fromCharCode(10));

    for (const line of lines) {
        const [key, ...valueParts] = line.split(":");
        if (key && valueParts.length > 0) {
            const trimmedKey = key.trim();
            const value = valueParts.join(":").trim();

            if (Object.prototype.hasOwnProperty.call(formData, \`input-\${trimmedKey}\`)) {
                inputs[trimmedKey] = value;
            } else if (Object.prototype.hasOwnProperty.call(formData, \`output-\${trimmedKey}\`)) {
                outputs[trimmedKey] = value;
            }
            // If the key doesn't match any input or output, it's ignored
        }
    }

    return { inputs, outputs };
}`,
                    data: {
                        __version: VERSION,
                        inputConfig: {
                            inputSchema: {
                                type: "object",
                                properties: {
                                    formData: {
                                        type: "object",
                                    },
                                    text: {
                                        type: "string",
                                    },
                                },
                                required: ["formData", "text"],
                            },
                            shouldSpread: false,
                        },
                        outputConfig: [{
                            type: "object",
                            properties: {
                                inputs: {
                                    type: "object",
                                },
                                outputs: {
                                    type: "object",
                                },
                            },
                            required: ["inputs", "outputs"],
                        }],
                        testCases: [
                            {
                                description: "should parse inputs and outputs correctly",
                                input: {
                                    formData: {
                                        "input-name": true,
                                        "output-result": true,
                                    },
                                    text: "name: John Doe\nresult: Success",
                                },
                                expectedOutput: {
                                    inputs: {
                                        "name": "John Doe",
                                    },
                                    outputs: {
                                        "result": "Success",
                                    },
                                },
                            },
                            {
                                description: "should ignore keys that do not match any input or output",
                                input: {
                                    formData: {
                                        "input-name": true,
                                        "output-result": true,
                                    },
                                    text: "name: John Doe\nage: 30\nresult: Success",
                                },
                                expectedOutput: {
                                    inputs: {
                                        "name": "John Doe",
                                    },
                                    outputs: {
                                        "result": "Success",
                                    },
                                },
                            },
                            {
                                description: "should handle multiple colons in the value correctly",
                                input: {
                                    formData: {
                                        "input-time": true,
                                    },
                                    text: "time: 10:30:00",
                                },
                                expectedOutput: {
                                    inputs: {
                                        "time": "10:30:00",
                                    },
                                    outputs: {},
                                },
                            },
                            {
                                description: "should return empty objects when no inputs or outputs match",
                                input: {
                                    formData: {
                                        "input-location": true,
                                    },
                                    text: "name: John Doe\nresult: Success",
                                },
                                expectedOutput: {
                                    inputs: {},
                                    outputs: {},
                                },
                            },
                            {
                                description: "should handle empty text correctly",
                                input: {
                                    formData: {
                                        "input-name": true,
                                    },
                                    text: "",
                                },
                                expectedOutput: {
                                    inputs: {},
                                    outputs: {},
                                },
                            },
                            {
                                description: "should handle text with only whitespaces correctly",
                                input: {
                                    formData: {
                                        "input-name": true,
                                    },
                                    text: "    ",
                                },
                                expectedOutput: {
                                    inputs: {},
                                    outputs: {},
                                },
                            },
                            {
                                description: "should correctly handle mixed case input and output keys",
                                input: {
                                    formData: {
                                        "input-Name": true,
                                        "output-Result": true,
                                    },
                                    text: "Name: John Doe\nResult: Success",
                                },
                                expectedOutput: {
                                    inputs: {
                                        "Name": "John Doe",
                                    },
                                    outputs: {
                                        "Result": "Success",
                                    },
                                },
                            },
                            {
                                description: "should ignore random text that does not form key-value pairs",
                                input: {
                                    formData: {
                                        "input-name": true,
                                        "output-result": true,
                                    },
                                    text: "Sure! Here is what you asked for:\nname: John Doe\nresult: Success",
                                },
                                expectedOutput: {
                                    inputs: {
                                        "name": "John Doe",
                                    },
                                    outputs: {
                                        "result": "Success",
                                    },
                                },
                            },
                            {
                                description: "should handle random text between key-value pairs",
                                input: {
                                    formData: {
                                        "input-name": true,
                                        "output-result": true,
                                    },
                                    text: "name: John Doe\nPlease note the following details are important.\nresult: Success",
                                },
                                expectedOutput: {
                                    inputs: {
                                        "name": "John Doe",
                                    },
                                    outputs: {
                                        "result": "Success",
                                    },
                                },
                            },
                            {
                                description: "should properly ignore lines without a colon",
                                input: {
                                    formData: {
                                        "input-name": true,
                                        "output-result": true,
                                    },
                                    text: "name: John Doe\nThis line has no colon\nresult: Success",
                                },
                                expectedOutput: {
                                    inputs: {
                                        "name": "John Doe",
                                    },
                                    outputs: {
                                        "result": "Success",
                                    },
                                },
                            },
                            {
                                description: "should handle text with mixed legitimate and illegitimate lines",
                                input: {
                                    formData: {
                                        "input-name": true,
                                        "output-result": true,
                                    },
                                    text: "Here's the info you requested:\nname: John Doe\nrandom statement here\nresult: Success\nEnd of message.",
                                },
                                expectedOutput: {
                                    inputs: {
                                        "name": "John Doe",
                                    },
                                    outputs: {
                                        "result": "Success",
                                    },
                                },
                            },
                        ],
                    },
                    translations: [{
                        language: "en",
                        id: uuid(),
                        name: "Parse Run IO From Plaintext",
                        description: "When a routine is being run autonomously, we may need to generate inputs and/or outputs before we can perform the action associated with the routine. This function parses the expected output and returns an object with an inputs list and outputs list.",
                    }],
                },
            }],
        },
    },
    {
        ...baseRoot,
        shape: {
            ...baseRootShape,
            id: SEEDED_IDS.Code.parseSearchTermsFromPlaintext,
            versions: [{
                ...baseVersion,
                shape: {
                    ...baseVersionShape,
                    id: SEEDED_IDS.CodeVersion.parseSearchTermsFromPlaintext,
                    content: String.raw`/**
 * Transforms plaintext containing formatted or plain search terms into a cleaned list.
 * Search terms can be prefaced with numbering (e.g., "1. term") or listed plainly.
 * The function ignores blank or whitespace-only lines and trims whitespace from text lines.
 * 
 * @returns A cleaned array of search terms.
 */
function parseSearchTermsFromPlaintext(text) {
    const lines = text.split(String.fromCharCode(10));
    const cleanedTerms = [];
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
}`,
                    data: {
                        __version: VERSION,
                        inputConfig: {
                            inputSchema: {
                                type: "string",
                            },
                            shouldSpread: false,
                        },
                        outputConfig: [{
                            type: "array",
                            items: {
                                type: "string",
                            },
                        }],
                        testCases: [
                            {
                                description: "should correctly parse plain search terms without numbers",
                                input: "how to cook a chicken\ncooking a chicken",
                                expectedOutput: ["how to cook a chicken", "cooking a chicken"],
                            },
                            {
                                description: "should parse numbered search terms and remove numbering",
                                input: "1. how to cook a chicken\n2. cooking a chicken",
                                expectedOutput: ["how to cook a chicken", "cooking a chicken"],
                            },
                            {
                                description: "should ignore blank and whitespace-only lines",
                                input: "1. how to cook a chicken\n   \n2. cooking a chicken\n",
                                expectedOutput: ["how to cook a chicken", "cooking a chicken"],
                            },
                            {
                                description: "should ignore random introductory or additional text",
                                input: "Here are your search queries:\n1. how to cook a chicken\n2. cooking a chicken\nThat's all!",
                                expectedOutput: ["how to cook a chicken", "cooking a chicken"],
                            },
                            {
                                description: "should process terms with numbers not followed by a dot as regular terms",
                                input: "1 how to cook a chicken\n2cooking a chicken",
                                expectedOutput: ["1 how to cook a chicken", "2cooking a chicken"],
                            },
                            {
                                description: "should handle empty strings gracefully",
                                input: "",
                                expectedOutput: [],
                            },
                            {
                                description: "should trim and process terms with additional spaces before and after numbers",
                                input: "   1.  how to cook a chicken   \n   2.   cooking a chicken   ",
                                expectedOutput: ["how to cook a chicken", "cooking a chicken"],
                            },
                            {
                                description: "should handle terms with multiple digits in numbering",
                                input: "10. how to cook a chicken\n20. cooking a chicken",
                                expectedOutput: ["how to cook a chicken", "cooking a chicken"],
                            },
                            {
                                description: "should handle non-English characters and special characters in terms",
                                input: "1. cómo cocinar un pollo\n2. cooking a chicken#",
                                expectedOutput: ["cómo cocinar un pollo", "cooking a chicken#"],
                            },
                            {
                                description: "should ignore lines not part of a structured numbered list",
                                input: "Sure! Here are the search queries:\n1. how to cook a chicken\n2. cooking a chicken\nPlease follow up if you need more info.",
                                expectedOutput: ["how to cook a chicken", "cooking a chicken"],
                            },
                            {
                                description: "should only process lines in the structured list even with interruptions",
                                input: "1. how to cook a chicken\nHere's something unrelated\n2. cooking a chicken\nEnd of list",
                                expectedOutput: ["how to cook a chicken", "cooking a chicken"],
                            },
                            {
                                description: "should ignore lines that contain numbers but are not properly formatted as list items",
                                input: "1 how to cook a chicken\n2: Here is an improperly formatted line\n2. cooking a chicken",
                                expectedOutput: ["cooking a chicken"],
                            },
                            {
                                description: "should handle text where valid list items are scattered among random text",
                                input: "Here are your instructions:\n1. how to cook a chicken\nSome random text.\n2. cooking a chicken\nMore random text here.",
                                expectedOutput: ["how to cook a chicken", "cooking a chicken"],
                            },
                            {
                                description: "should ignore non-list related text completely, even if it contains colons or numbers",
                                input: "Introduction:\nWe have some points to consider:\n1. how to cook a chicken\n2. cooking a chicken\nConclusion: Thank you for your attention!",
                                expectedOutput: ["how to cook a chicken", "cooking a chicken"],
                            },
                        ],
                    },
                    translations: [{
                        language: "en",
                        id: uuid(),
                        name: "Parse Search Terms From Plaintext",
                        description: "Converts formatted or plain plaintext search terms into a cleaned array of terms.",
                    }],
                },
            }],
        },
    },
    {
        ...baseRoot,
        shape: {
            ...baseRootShape,
            id: SEEDED_IDS.Code.listToPlaintext,
            versions: [{
                ...baseVersion,
                shape: {
                    ...baseVersionShape,
                    id: SEEDED_IDS.CodeVersion.listToPlaintext,
                    content: `/**
 * Converts a list of strings into plaintext.
 * Useful for generating plaintext from an array of terms, formatted without numbers.
 * 
 * @param list The array of strings to convert.
 * @param delimiter The delimiter to use between items. Defaults to a newline.
 * @returns A plaintext string of the list items, separated by newlines.
 */
function listToPlaintext(list, delimiter = String.fromCharCode(10)) {
    if (!Array.isArray(list) || list.length === 0) {
        return "";
    }
    return list.join(delimiter);
}`,
                    data: {
                        __version: VERSION,
                        inputConfig: {
                            inputSchema: {
                                type: "array",
                                items: [
                                    {
                                        type: "array",
                                        items: {
                                            type: "string",
                                        },
                                    },
                                    {
                                        type: "string",
                                    },
                                ],
                            },
                            shouldSpread: true,
                        },
                        outputConfig: [{
                            type: "string",
                        }],
                        testCases: [
                            {
                                description: "should return an empty string for an empty array",
                                input: [[]],
                                expectedOutput: "",
                            },
                            {
                                description: "should return the single item as is",
                                input: [["apple"]],
                                expectedOutput: "apple",
                            },
                            {
                                description: "should join multiple items with newlines",
                                input: [["apple", "banana", "cherry"]],
                                expectedOutput: "apple\nbanana\ncherry",
                            },
                            {
                                description: "should preserve spaces in items",
                                input: [["hello world", "foo bar"]],
                                expectedOutput: "hello world\nfoo bar",
                            },
                            {
                                description: "should support custom delimiters",
                                input: [["apple", "banana", "cherry"], ", "],
                                expectedOutput: "apple, banana, cherry",
                            },
                        ],
                    },
                    translations: [{
                        language: "en",
                        id: uuid(),
                        name: "List To Plaintext",
                        description: "Converts an array of strings into a single string, each item separated by a newline.",
                    }],
                },
            }],
        },
    },
    {
        ...baseRoot,
        shape: {
            ...baseRootShape,
            id: SEEDED_IDS.Code.listToNumberedPlaintext,
            versions: [{
                ...baseVersion,
                shape: {
                    ...baseVersionShape,
                    id: SEEDED_IDS.CodeVersion.listToNumberedPlaintext,
                    content: `/**
 * Converts a list of strings into a numbered plaintext string.
 * Useful for generating a formatted list where each item is preceded by its index number followed by a dot.
 *
 * @param list The array of strings to convert.
 * @param delimiter The delimiter to use between items. Defaults to a newline.
 * @returns A numbered plaintext string of the list items, each prefixed with its number and a dot, separated by newlines.
 */
function listToNumberedPlaintext(list, delimiter = String.fromCharCode(10)) {
    if (!Array.isArray(list) || list.length === 0) {
        return "";
    }
    return list.map((item, index) => (index + 1) + ". " + item).join(delimiter);
}
`,
                    data: {
                        __version: VERSION,
                        inputConfig: {
                            inputSchema: {
                                type: "array",
                                items: [
                                    {
                                        type: "array",
                                        items: {
                                            type: "string",
                                        },
                                    },
                                    {
                                        type: "string",
                                    },
                                ],
                            },
                            shouldSpread: true,
                        },
                        outputConfig: [{
                            type: "string",
                        }],
                        testCases: [
                            {
                                description: "should return an empty string for an empty array",
                                input: [[]],
                                expectedOutput: "",
                            },
                            {
                                description: "should return the single item with a number and a dot",
                                input: [["apple"]],
                                expectedOutput: "1. apple",
                            },
                            {
                                description: "should join multiple items with newlines and numbers",
                                input: [["apple", "banana", "cherry"]],
                                expectedOutput: "1. apple\n2. banana\n3. cherry",
                            },
                            {
                                description: "should preserve spaces in items",
                                input: [["hello world", "foo bar"]],
                                expectedOutput: "1. hello world\n2. foo bar",
                            },
                            {
                                description: "should support custom delimiters",
                                input: [["apple", "banana", "cherry"], ", "],
                                expectedOutput: "1. apple, 2. banana, 3. cherry",
                            },
                        ],
                    },
                    translations: [{
                        language: "en",
                        id: uuid(),
                        name: "List To Numbered Plaintext",
                        description: "Converts an array of strings into a single string, each item prefixed with its number in the list and separated by newlines.",
                    }],
                },
            }],
        },
    },
];
