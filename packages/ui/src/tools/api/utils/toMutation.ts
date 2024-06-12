import { GqlPartial, SelectionType } from "../types";
import { partialToString } from "./partialToString";

/**
 * Helper function for generating a GraphQL muation for a given endpoint.
 * @param endpointName The name of the operation (endpoint).
 * @param inputType The name of the input type.
 * @param fragments An array of GraphQL fragments to include in the operation.
 * @param selectionSet The selection set for the operation.
 * @returns An object containing:
 * - fragments: An array of names and gql-tag strings for each fragment used in the operation.
 * - tag: A gql-tag string for the operation.
 */
export const toMutation = async <
    Endpoint extends string,
    Partial extends GqlPartial<any>,
    Selection extends SelectionType,
>(
    endpointName: Endpoint,
    inputType: string | null,
    partial?: Partial,
    selectionType?: Selection | null | undefined,
) => {
    return await partialToString({
        endpointType: "mutation",
        endpointName,
        inputType,
        indent: 0,
        partial,
        selectionType,
    });
};
