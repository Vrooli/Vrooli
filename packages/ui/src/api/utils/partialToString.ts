import { exists } from "@shared/utils";
import { DeepPartialBooleanWithFragments, GqlPartial } from "types";
import { findSelection } from "./findSelection";
import { partialCombine } from "./partialCombine";
import { partialToStringHelper } from "./partialToStringHelper";

type PartialToStringProps<
    EndpointType extends 'query' | 'mutation',
    EndpointName extends string,
    Partial extends GqlPartial<any>,
    Selection extends 'common' | 'full' | 'list' | 'nav',
> = {
    endpointType: EndpointType,
    endpointName: EndpointName,
    /**
     * The number of spaces to indent the string
     */
    indent: number,
    inputType: string | null,
    partial?: Partial,
    selectionType?: Selection | null | undefined
}

/**
 * Converts a DeepPartialBooleanWithFragments object to a 
 * string that can be used in a GraphQL query/mutation.
 * @param obj The object to convert
 * @param indent The number of spaces to indent the string by
 * @returns A properly-indented string that can be used in a GraphQL query/mutation. 
 * The string is structured from top to bottom in the shape:
 * - Fragment definitions, with duplicate fragments omitted
 * - The query/mutation itself (e.g. 'query organization($input: FindByIdOrHandleInput!) {\norganization(input: $input) {\n')
 * - The fields of the query/mutation, with fragments referenced by name and unions formatted correctly
 * - The closing parentheses and brackets
 */
export const partialToString = <
    EndpointType extends 'query' | 'mutation',
    EndpointName extends string,
    Partial extends GqlPartial<any>,
    Selection extends 'common' | 'full' | 'list' | 'nav',
>({
    endpointType,
    endpointName,
    indent,
    inputType,
    partial,
    selectionType,
}: PartialToStringProps<EndpointType, EndpointName, Partial, Selection>): string => {
    // Calculate the fragments and selection set by combining partials
    let combined: DeepPartialBooleanWithFragments<any> = {};
    if (exists(partial) && exists(selectionType)) {
        // Find the actual selection type to use, based on which fields in partial are provided
        const actualSelectionType = findSelection(partial, selectionType);
        // To get the full selection, we must use partialCombine. This function 
        // combines two fields in partial, but also renames fields and removes duplicate 
        // fragments.
        // 'full' combines with 'common'
        if (actualSelectionType === 'full') {
            combined = partialCombine(partial.full!, partial.common ?? {});
        }
        // 'list' combines with 'common'
        else if (actualSelectionType === 'list') {
            combined = partialCombine(partial.list!, partial.common ?? {});
        }
        // 'common' and 'nav' combine with nothing, but we still use partialCombine for the 
        // renaming and fragment removal.
        else {
            combined = partialCombine(partial[actualSelectionType]!, {});
        }
    }
    // Initialize the string to return
    let str = '';
    // If there are fragments, add them first
    const { __define, ...rest } = combined;
    if (exists(__define) && Object.keys(__define).length > 0) {
        str += partialToStringHelper({ __define } as any, indent);
    }
    // Add the query/mutation itself
    str += `
${' '.repeat(indent)}${endpointType} ${endpointName}`;
    // If there is an input type, add it
    if (exists(inputType)) {
        str += `($input: ${inputType}!)`;
    }
    // Add the opening bracket
    str += ` {
${' '.repeat(indent + 2)}`;
    // If there is a partial, add the fields
    if (exists(combined)) {
        str += partialToStringHelper(rest, indent + 2);
    }
    // Add the closing bracket and parentheses
    str += `
${' '.repeat(indent)}}`;
    console.log('partialToString result', str)
    // Return the string
    return str;
};