import { DeepPartialBooleanWithFragments } from "types";
import { fragmentsToString } from "./fragmentsToString";
import { unionToString } from "./unionToString";

/**
 * Helper function to convert part of a GqlPartial object or fragment to a graphql-tag string.
 * @param partial The partial object to convert.
 * @param indent The number of spaces to indent the partial by.
 * @returns a graphql-tag string for the partial.
 */
export const partialToStringHelper = (
    partial: DeepPartialBooleanWithFragments<any>,
    indent: number = 0,
) => {
    // Initialize the result string.
    let result = '';
    // Loop through the partial object.
    for (const [key, value] of Object.entries(partial)) {
        // Add indentation.
        result += ' '.repeat(indent);
        // If key is __define, use fragmentToString to convert the fragment.
        if (key === '__define') {
            result += fragmentsToString(value as Exclude<DeepPartialBooleanWithFragments<any>['__define'], undefined>, indent);
        }
        // If key is __union, use unionToString to convert the union.
        else if (key === '__union') {
            result += unionToString(value as Exclude<DeepPartialBooleanWithFragments<any>['__union'], undefined>, indent);
        }
        // If key is __use, use value as fragment name
        else if (key === '__use') {
            result += `...${value}\n`;
        }
        // If value is a boolean, add the key.
        if (typeof value === 'boolean') {
            result += `${key}\n`;
        } 
        // Otherwise, value must be an (possibly lazy) object. So we can recurse.
        else {
            result += `${key} {\n`;
            result += partialToStringHelper(typeof value === 'function' ? value() : value as any, indent + 4);
            result += `${' '.repeat(indent)}}\n`;
        }
    }
    return result;
}