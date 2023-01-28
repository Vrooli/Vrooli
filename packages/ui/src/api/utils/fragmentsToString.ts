import { exists } from "@shared/utils";
import { DeepPartialBooleanWithFragments } from "types";
import { findSelection } from "./findSelection";
import { partialCombine } from "./partialCombine";
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
        // Get the selection type for the partial
        const actualType = findSelection(partial, type);
        // Get selection data for the partial
        let selectionData = partial[actualType]!;
        // If the selectiion type is 'full' or 'list', and the 'common' selection is defined, combine the two.
        if ((actualType === 'full' || actualType === 'list') && exists(partial.common)) {
            selectionData = partialCombine(selectionData, partial.common);
        }
        fragmentString += `${' '.repeat(indent)}fragment ${name} on ${partial.__typename} {\n`;
        fragmentString += partialToStringHelper(selectionData, indent + 4)
        fragmentString += `${' '.repeat(indent)}}\n\n`;
    }
    return fragmentString;
}