import { GqlPartial } from "../types";
import { partialToString } from "./partialToString";

/**
 * Helper function for generating a GraphQL query for a given endpoint.
 * @param endpointName The name of the operation (endpoint).
 * @param inputType The name of the input type.
 * @param partial GqlPartial object containing selection data
 * @param selectionType Which selection in the GqlPartial to use
 * @returns An object containing:
 * - fragments: An array of names and gql-tag strings for each fragment used in the operation.
 * - tag: A gql-tag string for the operation.
 */
export const toQuery = async <
    Endpoint extends string,
    Partial extends GqlPartial<any>,
    Selection extends "common" | "full" | "list" | "nav",
>(
    endpointName: Endpoint,
    inputType: string | null,
    partial?: Partial,
    selectionType?: Selection | null | undefined,
) => {
    return await partialToString({
        endpointType: "query",
        endpointName,
        inputType,
        indent: 0,
        partial,
        selectionType,
    });
};
