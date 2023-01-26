import { DeepPartialBooleanWithFragments, GqlPartial } from "types";
import { removeValuesUsingDot } from "utils";
import { findSelection } from "./findSelection";

/**
 * Adds a relation to an GraphQL selection set, while optionally omitting one or more fields.
 * @param partial The GqlPartial object containing the selection set
 * @param selection Which selection from GqlPartial to use
 * @param exceptions Exceptions object containing fields to omit. Supports dot notation.
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
    let selection = partial[findSelection(partial, selectionType)]!;
    // If no exceptions, return selection
    if (!exceptions || !exceptions.omit) return { __typename: partial.__typename, ...selection } as any;
    // Remove all exceptions. Supports dot notation.
    removeValuesUsingDot(selection, ...(Array.isArray(exceptions.omit) ? exceptions.omit : [exceptions.omit]));
    return { __typename: partial.__typename, ...selection } as any;
}