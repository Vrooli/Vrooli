import { exists } from "@shared/utils";
import { GqlPartial } from "types";

/**
 * Finds the best DeepPartialBooleanWithFragments 
 * to use in a GqlPartial object
 * @param obj The GqlPartial object to search
 * @param selection The preferred selection to use
 * @returns The preferred selection if it exists, otherwise the best selection
 */
export const findSelection = (
    obj: GqlPartial<any>,
    selection: 'common' | 'list' | 'full' | 'nav',
): 'common' | 'list' | 'full' | 'nav' => {
    let result: 'common' | 'full' | 'list' | 'nav' | undefined;
    // Common prefers common -> list -> full -> nav
    if (selection === 'common')
        result = exists(obj.common) ? 'common' : exists(obj.list) ? 'list' : exists(obj.full) ? 'full' : exists(obj.nav) ? 'nav' : undefined;
    // List prefers list -> common -> full -> nav
    else if (selection === 'list')
        result = exists(obj.list) ? 'list' : exists(obj['common']) ? 'common' : exists(obj.full) ? 'full' : exists(obj.nav) ? 'nav' : undefined;
    // Full prefers full -> list -> common -> nav
    else if (selection === 'full')
        result = exists(obj.full) ? 'full' : exists(obj.list) ? 'list' : exists(obj['common']) ? 'common' : exists(obj.nav) ? 'nav' : undefined;
    // Nav prefers nav -> common -> list -> full
    else if (selection === 'nav')
        result = exists(obj.nav) ? 'nav' : exists(obj['common']) ? 'common' : exists(obj.list) ? 'list' : exists(obj.full) ? 'full' : undefined;
    // If result is undefined, throw an error
    if (!exists(result)) {
        throw new Error(`Could not determine actual selection type for '${obj.__typename}' '${selection}'`);
    }
    // If result is not the same as the specified selection type, throw a warning
    if (result !== selection) {
        console.warn(`Specified selection type '${selection}' for '${obj.__typename}' does not exist. Using '${result}' instead.`);
    }
    return result;
}