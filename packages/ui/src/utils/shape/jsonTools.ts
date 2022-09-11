import { isObject } from "@shared/utils";

/**
 * JSON format error object. Used when 
 * discrepancies are found between the JSON format and the 
 * inputted JSON.
 */
export interface JSONFormatError {
    fieldName: string;
    path: string[];
    message: string;
}

/**
 * Return object when validating that JSON input matches format
 */
export interface JSONValidationResult {
    isValid: boolean;
    errors?: JSONFormatError[];
}

/**
 * Markdown text that explains how to enter JSON data
 */
export const jsonHelpText =
    `JSON is a widely used format for storing data objects.  

If you are unfamiliar with JSON, you may [read this guide](https://www.tutorialspoint.com/json/json_quick_guide.htm) to learn how to use it.  

On top of the JSON format, we also support the following:  
- **Variables**: Any key or value can be substituted with a variable. These are used to make it easy for users to fill in their own data, as well as 
provide details about those parts of the JSON. Variables follow the format of &lt;variable_name&gt;.  
- **Optional fields**: Fields that are not required can be marked as optional. Optional fields must start with a question mark (e.g. ?field_name).  
- **Additional fields**: If an object contains arbitrary fields, add a field with brackets (e.g. [variable]).  
- **Data types**: Data types are specified by reserved words (e.g. &lt;string&gt;, &lt;number | boolean&gt;, &lt;any&gt;, etc.), variable names (e.g. &lt;variable_name&gt;), standard IDs (e.g. &lt;decdf6c8-4230-4777-b8e3-799df2503c42&gt;), or simply entered as normal JSON.
`

/**
 * Determines if input is an object, and not a Date or Array
 * @param obj - object to check
 * @returns true if object, false otherwise
 */
const isActualObject = (obj: any): boolean => !Array.isArray(obj) && isObject(obj) && Object.prototype.toString.call(obj) !== '[object Date]'

/**
 * Checks if a string can be parsed as JSON.
 * @param value The string to check.
 * @returns Error messages if the string is not valid JSON, or empty array if valid.
 */
export const checkJsonErrors = (value: { [x: string]: any } | string | null | undefined): string[] => {
    const errors: string[] = [];
    try {
        // If null or undefined, return false
        if (value === null || value === undefined) {
            errors.push('Cannot be empty');
            return errors;
        }
        // Initialize JSON object
        let jsonObject: { [x: string]: any };
        // If a string
        if (typeof value === 'string') {
            // Check if empty
            if (value.trim().length === 0) {
                errors.push('Cannot be empty');
                return errors;
            }
            // Try to convert to JSON object
            jsonObject = JSON.parse(value);
        }
        // If an object, set as JSON object
        else jsonObject = value;
        // Check if jsonObject has any properties
        if (Object.keys(jsonObject).length === 0) {
            errors.push('Must have at least one property');
        }
        // Check if JSON is too long
        //TODO
        return errors;
    } catch (e) {
        errors.push('Invalid JSON');
        return errors;
    }
};

/**
 * Recursive helper function for comparing a JSON format to JSON data
 * @param fieldName Name of field with format
 * @param path Path to field in format
 * @param format object representing the format that the data should conform to
 * @param data object representing the data to be compared
 * @returns A list of differences between the two, or empty list if there are no differences
 */
export const compareJSONHelper = (fieldName: string, path: string[], format: object | string, data: object | string): JSONFormatError[] => {
    return {} as any;
    // Check if format is array
    if (Array.isArray(format)) {
        // If data is not the same shape, return error
        if (!Array.isArray(data)) return [{
            fieldName,
            path,
            message: `Expected array, got ${typeof data}`
        }];
        // Recursively compare each element in the array to the first element in the format array. 
        // This is because the format array can only have one element. If you need to set more 
        // advanced validation (e.g. multiple types), you will have to convert this array to 
        // a variable that uses yup validation.
        //TODO
    }
    // Check if format is object
    else if (isActualObject(format)) {
        // If data is not the same shape, return error
        if (!isActualObject(data)) return [{
            fieldName,
            path,
            message: `Expected object, got ${typeof data}`
        }];
        // Recursively compare each key in the object to the format object.
        //TODO
    }
    // Check if format is a variable (i.e. wrapped in <>. May also have ? at the beginning)
    else if (typeof format === 'string' && ((format as string).startsWith('<') || (format as string).startsWith('?<')) && (format as string).endsWith('>')) {
        //TODO
    }
    // Check if format is a catch-all (i.e. wrapped in [])
    else if (typeof format === 'string' && (format as string).startsWith('[') && (format as string).endsWith(']')) {
        //TODO
    }
    // Format must be primitive
    else {
        // If data is not the same type, return error
        if (Array.isArray(data) || isActualObject(data)) return [{
            fieldName,
            path,
            message: `Expected ${format}, got ${typeof data}`
        }];
    }
}

/**
 * Compares a JSON format string to a JSON object and returns a list of differences. 
 * A key wrapped in <> is a variable. Its value will be compared to any of the possible 
 * values on that level of the format object, to see if it matches. 
 * A key prefixed with ? is optional. 
 * A key wrapped in [] means that additional keys can be added to the object. 
 * The value of this wrapped key (which will also be wrapped in []) specifies the
 * type of values allowed.
 * ex: {
 * "<721>": {
 *   "<policy_id>": {
 *     "<asset_name>": {
 *       "name": "<asset_name>",
 *       "?image": "<ipfs_link>",
 *       "?mediaType": "<mime_type>",
 *       "?description": "<description>",
 *       "?files": [
 *         {
 *          "name": "<asset_name>",
 *           "mediaType": "<mime_type>",
 *           "src": "<ipfs_link>"
 *         }
 *       ],
 *       "[x]": "[any]"
 *     }
 *   },
 *   "version": "1.0"
 *  }
 * }
 * @param format The JSON format string
 * @param data The JSON object to compare
 * @returns Result object with isValid and errors fields
 */
export const compareJSONToFormat = (format: string, data: any): JSONValidationResult => {
    let formatJSON: any;
    let dataJSON: any;
    try {
        formatJSON = JSON.parse(format);
        dataJSON = JSON.parse(data);
        const errors = compareJSONHelper('$root', [], formatJSON, dataJSON);
        return {
            isValid: errors.length === 0,
            errors
        };
    }
    catch (error) {
        console.error('Error parsing JSON format or data', error);
        return {
            isValid: false,
            errors: [{
                fieldName: '$root',
                path: [],
                message: 'Failed to parse JSON format or data'
            }]
        }
    }
}

type Position = { start: number; end: number };
type VariablePosition = { field: Position; value: Position };
/**
 * Finds the start and end positions of every matching variable/value pair in a JSON format
 * @param variable Name of variable
 * @param format format to search
 * @returns Array of start and end positions of every matching variable/value pair
 */
export const findVariablePositions = (variable: string, format: { [x: string]: any | undefined }): VariablePosition[] => {
    // Initialize array of variable positions
    const variablePositions: VariablePosition[] = [];
    // Check for valid format
    if (!format) return variablePositions;
    const formatString = JSON.stringify(format);
    // Create substring to search for
    const keySubstring = `"<${variable}>":`;
    // Create fields to store data about the current position in the format string
    let openBracketCounter = 0; // Net number of open brackets since last variable
    let inQuotes = false; // If in quotes, ignore brackets
    let index = 1; // Current index in format string. Ignore first character, since we check the previous character in the loop (and the first character will always be a bracket)
    let keyStartIndex = -1; // Start index of variable's key
    let valueStartIndex = -1; // Start index of variable's value
    // Loop through format string, until end is reached. 
    while (index < formatString.length) {
        const currChar = formatString[index];
        const lastChar = formatString[index - 1];
        // If current and last char are "\", then the next character is escaped and should be ignored
        if (currChar !== '\\' && lastChar !== '\\') {
            // If variableStart has not been found, then check if we are at the start of the next variable instance
            if (keyStartIndex === -1) {
                // If the rest of the formatString is long enough to contain the variable substring, 
                // and the substring is found, then we are at the start of a new variable instance
                if (formatString.length - index >= keySubstring.length && formatString.substring(index, index + keySubstring.length) === keySubstring) {
                    keyStartIndex = index;
                }
            }
            // If variableStart has been found, check for the start of the value
            else if (index === keyStartIndex + keySubstring.length) {
                // If the current character is not a quote or an open bracket, then the value is invalid
                if (currChar !== '"' && currChar !== '{') {
                    keyStartIndex = -1;
                    valueStartIndex = -1;
                }
                // Otherwise, set up the startIndex, bracket counter and inQuotes flag
                valueStartIndex = index;
                openBracketCounter = Number(currChar === '{');
                inQuotes = currChar === '"';
            }
            // If variableStart has been found, check if we are at the end
            else if (index > keyStartIndex + keySubstring.length) {
                openBracketCounter += Number(currChar === '{');
                openBracketCounter -= Number(currChar === '}');
                if (currChar === '"') inQuotes = !inQuotes;
                // If the bracket counter is 0, and we are not in quotes, then we are at the end of the variable
                if (openBracketCounter === 0 && !inQuotes) {
                    // Add the variable to the array
                    variablePositions.push({
                        field: {
                            start: keyStartIndex,
                            end: keyStartIndex + keySubstring.length,
                        },
                        value: {
                            start: valueStartIndex,
                            end: index,
                        },
                    });
                }
            }
        } else index++;
        index++;
    }
    return variablePositions;
}

/**
 * Converts a JSON string to a pretty-printed JSON markdown string.
 * @param value The JSON string to convert.
 * @returns The pretty-printed JSON string, to be rendered in <Markdown />.
 */
export const jsonToMarkdown = (value: { [x: string]: any } | string | null): string | null => {
    try {
        const parsed = typeof value === 'string' ? JSON.parse(value) : value;
        return value ? `\`\`\`json\n${JSON.stringify(parsed, null, 2)}\n\`\`\`` : null;
    } catch (e) {
        return null;
    }
}

/**
 * Converts a JSON object to a pretty-printed JSON string. This includes adding 
 * newlines and indentation to make the JSON more readable.
 * @param value The JSON object to convert.
 * @returns The pretty-printed JSON string, to be rendered in multiline <TextField />.
 */
export const jsonToString = (value: { [x: string]: any } | string | null | undefined): string | null => {
    try {
        const parsed = typeof value === 'string' ? JSON.parse(value) : value;
        return value ? JSON.stringify(parsed, null, 2) : null;
    } catch (e) {
        return null;
    }
}

/**
 * Simple check to determine if two JSON objects are equal. 
 * If objects have the same keys/values but they are ordered differently, then they are not equal.
 * @param a First JSON object or string to compare
 * @param b Second JSON object or string to compare
 */
export const isEqualJSON = (
    a: { [x: string]: any } | string | null | undefined,
    b: { [x: string]: any } | string | null | undefined
): boolean => {
    try {
        const parsedA = typeof a === 'string' ? JSON.parse(a) : a;
        const parsedB = typeof b === 'string' ? JSON.parse(b) : b;
        return JSON.stringify(parsedA) === JSON.stringify(parsedB);
    } catch (e) {
        return false;
    }
}

/**
 * Stringifies an object, only if it is not already stringified.
 * @param value The object to stringify.
 * @returns The stringified object.
 */
export const safeStringify = (value: any): string => {
    if (typeof value === 'string' && value.length > 1 && value[0] === '{' && value[value.length - 1] === '}') {
        return value;
    }
    return JSON.stringify(value);
}