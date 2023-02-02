import { exists } from "@shared/utils";
import { DeepPartialBooleanWithFragments, GqlPartial } from "../types";
import { fragmentsToString } from "./fragmentsToString";
import { partialToStringHelper } from "./partialToStringHelper";
import { rel } from "./relPartial";

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
export const partialToString = async <
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
}: PartialToStringProps<EndpointType, EndpointName, Partial, Selection>): Promise<{
    fragments: [string, string][],
    tag: string
}> => {
    // Initialize return data
    let fragments: [string, string][] = [];
    let tag = '';
    // Calculate the fragments and selection set by combining partials
    let combined: DeepPartialBooleanWithFragments<any> = {};
    if (exists(partial) && exists(selectionType)) {
        combined = await rel(partial, selectionType)
    }
    // If there are fragments, convert them to strings
    const { __define, ...rest } = combined;
    if (exists(__define) && Object.keys(__define).length > 0) {
        fragments = await fragmentsToString(__define)
        // For every fragment, add reference to it in the tag
        fragments.forEach(([fragmentName]) => {
            tag += `${' '.repeat(indent)}\${${fragmentName}}\n`
        })
    }
    // Add the query/mutation to the tag
    tag += `
${' '.repeat(indent)}${endpointType} ${endpointName}`;
    // If there is an input type, add it
    if (exists(inputType)) {
        tag += `($input: ${inputType}!)`;
    }
    // Add the opening bracket
    tag += ` {
${' '.repeat(indent + 2)}`;
    // Add name of the query/mutation and input
    tag += `${endpointName}${exists(inputType) ? '(input: $input)' : ''}`;
    // If there is a partial, add the fields
    if (exists(rest) && Object.keys(rest).length > 0) {
        // Add another opening bracket
        tag += ` {
`;
        tag += await partialToStringHelper(rest, indent + 4);
        // Add a closing brackets
        tag += `${' '.repeat(indent + 2)}}`;
    }
    // Add the final closing bracket
    tag += `
${' '.repeat(indent)}}`;
    // Return the tag and fragments
    return { fragments, tag };
};