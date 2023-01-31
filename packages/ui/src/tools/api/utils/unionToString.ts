import { DeepPartialBooleanWithFragments } from "../types";
import { partialToStringHelper } from "./partialToStringHelper";

/**
 * Converts union data (from DeepPartialBooleanWithFragments.__union) into a graphql-tag string.
 * @param union The union data.
 * @param indent The number of spaces to indent the union by.
 * @returns a graphql-tag string for the union, without outer braces.
 */
export const unionToString = async (
    union: Exclude<DeepPartialBooleanWithFragments<any>['__union'], undefined>,
    indent: number = 0,
) => {
    // Initialize the result string.
    let result = '';
    // Loop through the union object.
    for (const [key, value] of Object.entries(union)) {
        // Add indentation.
        result += ' '.repeat(indent);
        // Add ellipsis, object type, and open brace.
        result += `... on ${key} {\n`;
        // Value should be either a string, object, or function.
        // If a string, treat as fragment name
        if (typeof value === 'string') {
            result += `${' '.repeat(indent + 4)}...${value}\n`;
        }
        // If an object or function, convert 
        else if (typeof value === 'object' || typeof value === 'function') {
            result += await partialToStringHelper(typeof value === 'function' ? await value() : value, indent + 4);
        }
        // Shouldn't be anything else. If so, there was likely an issue with 
        // converting union references (which can be a string, number, or symbol) to unique strings
        else {
            console.error('unionToString got unexpected value', key, value);
        }
        // Close the brace.
        result += `${' '.repeat(indent)}}\n`;
    }
    return result;
}