import _ from "lodash";

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
 * Determines if input is an object, and not a Date or Array
 * @param obj - object to check
 * @returns true if object, false otherwise
 */
 const isActualObject = (obj: any): boolean => !Array.isArray(obj) && _.isObject(obj) && Object.prototype.toString.call(obj) !== '[object Date]'

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