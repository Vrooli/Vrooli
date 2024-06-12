import { exists } from "@local/shared";
import { GqlPartial, SelectionType } from "../types";

/**
 * Finds the best DeepPartialBooleanWithFragments 
 * to use in a GqlPartial object
 * @param obj The GqlPartial object to search
 * @param selection The preferred selection to use
 * @returns The preferred selection if it exists, otherwise the best selection
 */
export const findSelection = (
    obj: GqlPartial<any>,
    selection: SelectionType,
): SelectionType => {
    const selectionOrder: Record<SelectionType, SelectionType[]> = {
        common: ["common", "list", "full", "nav"],
        list: ["list", "common", "full", "nav"],
        full: ["full", "list", "common", "nav"],
        nav: ["nav", "common", "list", "full"],
    };

    // Find the first existing selection based on the preferred order
    const result = selectionOrder[selection].find(sel => exists(obj[sel]));

    if (!result) {
        throw new Error(`Could not determine actual selection type for '${obj.__typename}' '${selection}'`);
    }

    if (result !== selection) {
        console.warn(`Specified selection type '${selection}' for '${obj.__typename}' does not exist. Try using '${result}' instead.`);
    }

    return result;
};
