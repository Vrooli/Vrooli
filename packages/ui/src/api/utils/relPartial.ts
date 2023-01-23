import { GqlPartial } from "types";

/**
 * Adds a relation to an GraphQL selection set, while optionally omitting one or more fields.
 * @param partial The GqlPartial object containing the selection set
 * @param selection Which selection from GqlPartial to use
 * @param exceptions Exceptions object containing fields to omit
 */
export const relPartial = <
    // T extends ({ __typename: string } & { [key: string | number | symbol]: any }),
    Partial extends GqlPartial<any>,
    Selection extends 'common' | 'full' | 'list' | 'nav',
    OmitField extends string | number | symbol,
    // OmitField extends keyof T,
>(
    partial: Partial,
    selection: Selection,
    exceptions?: { omit: OmitField | OmitField[] }
): any => { // DeepPartialBooleanWithFragments<NonMaybe<T>>
    return {}
    // // Find selection, using fallbacks if necessary
    // // Nav does not have a fallback
    // // TODO also handle combining with 'common'
    // const selectionFn = selection === 'nav' ? partial[selection] :
    //     (partial[selection] || partial.list || partial.full || partial.common);
    // if (!selectionFn) return {};
    // // If no exceptions, return selection
    // if (!exceptions || !exceptions.omit) return selectionFn();
    // // Get selection
    // const selectionData = selectionFn();
    // // Remove all exceptions
    // if (Array.isArray(exceptions.omit)) exceptions.omit.forEach(e => delete selectionData[e]);
    // else delete selectionData[exceptions.omit];
    // return selectionData;
}