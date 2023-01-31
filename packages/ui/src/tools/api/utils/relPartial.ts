import { exists } from "@shared/utils";
import { DeepPartialBooleanWithFragments, GqlPartial } from "../types";
import { removeValuesUsingDot } from "../../../utils/shape/general/objectTools";
import { findSelection } from "./findSelection";
import { partialShape } from "./partialShape";
import pkg from 'lodash';
const { merge } = pkg;

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
    // temporary number for debugging
    const num = Math.floor(Math.random() * 10000);
    console.log('relPartial', num, 'a', partial.__typename, selectionType, exceptions)
    const hasExceptions = exists(exceptions) && exists(exceptions.omit)
    // Find correct selection to use
    const actualSelectionType = findSelection(partial, selectionType);
    // Get selection data for the partial
    let selectionData = partial[actualSelectionType]!;
    console.log('relPartial', num, 'b', actualSelectionType, selectionData)
    // Remove all exceptions. Supports dot notation.
    hasExceptions && removeValuesUsingDot(selectionData, ...(Array.isArray(exceptions.omit) ? exceptions.omit : [exceptions.omit]));
    // Shape selection data
    selectionData = partialShape(selectionData);
    console.log('relPartial', num, 'c', selectionData)
    // If the selectiion type is 'full' or 'list', and the 'common' selection is defined, combine the two.
    if (['full', 'list'].includes(actualSelectionType) && exists(partial.common)) {
        let commonData = partial.common;
        console.log('relPartial', num, 'd', commonData)
        // Remove exceptions from common selection
        hasExceptions && removeValuesUsingDot(commonData, ...(Array.isArray(exceptions.omit) ? exceptions.omit : [exceptions.omit]));
        // Shape common selection
        commonData = partialShape(commonData);
        console.log('relPartial', num, 'e', commonData)
        // Merge common selection into selection data
        selectionData = merge(selectionData, commonData);
    }
    console.log('relPartial', num, 'f', actualSelectionType, {...selectionData})
    // TODO seems like issue may be that relPartial renames fields, while fields remain with inital name 
    // during __define logic somewhere
    console.log('relPartial', num, 'g', { __typename: partial.__typename, __selectionType: actualSelectionType, ...selectionData });
    return { __typename: partial.__typename, __selectionType: actualSelectionType, ...selectionData } as any;
}