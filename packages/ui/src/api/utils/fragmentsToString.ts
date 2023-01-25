import { DeepPartialBooleanWithFragments } from "types";
import { findSelection } from "./findSelection";
import { partialToStringHelper } from "./partialToStringHelper";

/**
 * Converts fragment data (from DeepPartialBooleanWithFragments.__define) into a graphql-tag string.
 * @param fragments The fragment data.
 * @param indent The number of spaces to indent the fragment by.
 * @returns a graphql-tag string for the fragment.
 */
export const fragmentsToString = (
    fragments: Exclude<DeepPartialBooleanWithFragments<any>['__define'], undefined>,
    indent: number = 0,
) => {
    // Initialize the fragment string.
    let fragmentString = '';
    for (const [name, [partial, type]] of Object.entries(fragments)) {
        fragmentString += `${' '.repeat(indent)}fragment ${name} on ${type} {\n`;
        fragmentString += partialToStringHelper(partial[findSelection(partial, type)]!, indent + 4);
        fragmentString += `${' '.repeat(indent)}}\n`;
    }
    return fragmentString;
}