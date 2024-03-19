import { exists } from "@local/shared";
import { DeepPartialBooleanWithFragments, GqlPartial, SelectionType } from "../types";
import { fragmentsNeeded } from "./fragmentsNeeded";
import { fragmentsToString } from "./fragmentsToString";
import { partialToStringHelper } from "./partialToStringHelper";
import { rel } from "./rel";

type PartialToStringProps<
    EndpointType extends "query" | "mutation",
    EndpointName extends string,
    Partial extends GqlPartial<any>,
    Selection extends SelectionType,
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
    EndpointType extends "query" | "mutation",
    EndpointName extends string,
    Partial extends GqlPartial<any>,
    Selection extends SelectionType,
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
    let tag = "";
    // Calculate the fragments and selection set by combining partials
    let combined: DeepPartialBooleanWithFragments<any> = {};
    if (exists(partial) && exists(selectionType)) {
        combined = await rel(partial, selectionType);
    }
    // Split fragments from the rest, so we can handle them separately
    const { __define, ...rest } = combined;
    // Add the query/mutation to the tag
    tag += `
${" ".repeat(indent)}${endpointType} ${endpointName}`;
    // If there is an input type, add it
    if (exists(inputType)) {
        tag += `($input: ${inputType}!)`;
    }
    // Add the opening bracket
    tag += ` {
${" ".repeat(indent + 2)}`;
    // Add name of the query/mutation and input
    tag += `${endpointName}${exists(inputType) ? "(input: $input)" : ""}`;
    // If there is a partial, add the fields
    if (exists(rest) && Object.keys(rest).length > 0) {
        // Add another opening bracket
        tag += ` {
`;
        tag += await partialToStringHelper(rest, indent + 4);
        // Add a closing brackets
        tag += `${" ".repeat(indent + 2)}}`;
    }
    // Add the final closing bracket
    tag += `
${" ".repeat(indent)}}`;
    // Before returning, add fragments to the beginning of the tag. 
    // We do this here so we can filter out additional fragments which may have snuck in 
    // (e.g. if a relation has an omitted field which used a fragment, it could make it here). 
    // Ideally we'd fix this problem earlier in the process, but ¯\_(ツ)_/¯
    if (exists(__define) && Object.keys(__define).length > 0) {
        let fragmentsString = "\n";
        fragments = await fragmentsToString(__define);
        // Filter out fragments not found in the tag
        fragments = fragmentsNeeded(fragments, tag); //TODO for morning: commenting this out fixes bookmarklist findMany, but breaks many other things. For example, projectList findOne now adds Api_list fragment (among others), when that's not needed
        // Sort fragments by name, just because it looks nicer
        fragments.sort(([a], [b]) => a.localeCompare(b));
        // For every fragment, add reference to it in the tag
        fragments.forEach(([fragmentName]) => {
            fragmentsString += `${" ".repeat(indent)}\${${fragmentName}}\n`;
        });
        // Add the fragments to the beginning of the tag
        tag = fragmentsString + tag;
    }
    // Finally, return results
    return { fragments, tag };
};
