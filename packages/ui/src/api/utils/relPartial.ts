import { DeepPartialBooleanWithFragments, GqlPartial } from "types";
import { findSelection } from "./findSelection";

/**
 * Adds a relation to an GraphQL selection set, while optionally omitting one or more fields.
 * @param partial The GqlPartial object containing the selection set
 * @param selection Which selection from GqlPartial to use
 * @param exceptions Exceptions object containing fields to omit
 */
export const relPartial = <
    Partial extends GqlPartial<any>,
    Selection extends 'common' | 'full' | 'list' | 'nav',
    OmitField extends string | number | symbol,
>(
    partial: Partial,
    selectionType: Selection,
    exceptions?: { omit: OmitField | OmitField[] }
): DeepPartialBooleanWithFragments<any> => {
    // Find correct selection to use
    const selection = partial[findSelection(partial, selectionType)]!;
    // If no exceptions, return selection
    if (!exceptions || !exceptions.omit) return selection;
    // Remove all exceptions
    if (Array.isArray(exceptions.omit)) exceptions.omit.forEach(e => delete selection[e]);
    else delete selection[exceptions.omit];
    return selection;
}