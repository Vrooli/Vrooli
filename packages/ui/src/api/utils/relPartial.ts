import { exists } from "@shared/utils";
import { DeepPartialBooleanWithFragments, GqlPartial } from "types";
import { removeValuesUsingDot } from "utils";
import { findSelection } from "./findSelection";
import { partialCombine } from "./partialCombine";

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
    console.log('relPartial start', partial.__typename, selectionType, exceptions)
    const hasExceptions = exists(exceptions) && exists(exceptions.omit)
    // Find correct selection to use
    const actualSelectionType = findSelection(partial, selectionType);
    // Get selection data for the partial
    let selectionData = partial[actualSelectionType]!;
    console.log('relPartial selectionData', actualSelectionType, {...selectionData})
    // Remove all exceptions. Supports dot notation.
    hasExceptions && removeValuesUsingDot(selectionData, ...(Array.isArray(exceptions.omit) ? exceptions.omit : [exceptions.omit]));
    // If the selectiion type is 'full' or 'list', and the 'common' selection is defined, combine the two.
    if ((actualSelectionType === 'full' || actualSelectionType === 'list') && exists(partial.common)) {
        selectionData = partialCombine(selectionData, partial.common);
    }
    // Remove exceptions again, in case they were added to the common selection
    hasExceptions && removeValuesUsingDot(selectionData, ...(Array.isArray(exceptions.omit) ? exceptions.omit : [exceptions.omit]));
    // TODO seems like issue may be that relPartial renames fields, while fields remain with inital name 
    // during __define logic somewhere
    console.log('relPartial complete', { __typename: partial.__typename, ...selectionData });
    return { __typename: partial.__typename, ...selectionData } as any;
}