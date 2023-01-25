import gql from 'graphql-tag';
import { GqlPartial } from 'types';
import { partialToString } from './partialToString';

/**
 * Helper function for generating a GraphQL query for a given endpoint.
 * @param endpointName The name of the operation (endpoint).
 * @param inputType The name of the input type.
 * @param partial GqlPartial object containing selection data
 * @param selectionType Which selection in the GqlPartial to use
 * @returns a tuple of: a graphql-tag string for the endpoint, and the endpoint's name.
 */
export const toQuery = <
    Endpoint extends string,
    Partial extends GqlPartial<any>,
    Selection extends 'common' | 'full' | 'list' | 'nav',
>(
    endpointName: Endpoint,
    inputType: string | null,
    partial?: Partial,
    selectionType?: Selection | null | undefined
) => {
    return [gql`${partialToString({
        endpointType: 'query',
        endpointName,
        inputType,
        indent: 0,
        partial,
        selectionType
    })}`, endpointName] as const
}