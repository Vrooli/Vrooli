import { DocumentNode } from 'graphql';
import { gql } from 'graphql-tag';

/**
 * Helper function for generating a GraphQL query or mutation string for a given endpoint.
 * 
 * Example: toGql('query', 'api', 'FindByIdInput', [fullFields], `...fullFields`) => 
 * gql`
 *      ${fullFields}
 *      query api($input: FindByIdInput!) {
 *          api(input: $input) {
 *              ...fullFields
 *          }
 *      }
 *  `
 * 
 * @param operationType The type of operation, either 'query' or 'mutation'.
 * @param operationName The name of the operation.
 * @param inputType The name of the input type.
 * @param fragments An array of GraphQL fragments to include in the operation.
 * @param selectionSet The selection set for the operation.
 * @returns a graphql-tag string for the endpoint.
 */
export const toGql = (
    operationType: 'query' | 'mutation',
    operationName: string,
    inputType: string | null,
    fragments: DocumentNode[],
    selectionSet: string | null
) => {
    const fragmentStrings = fragments.map(fragment => `${fragment}\n`);
    const selection = selectionSet ? `{\n${selectionSet}\n}` : '';
    const signature = inputType ? `($input: ${inputType}!)` : '';
    return gql`
        ${fragmentStrings}
        ${operationType} ${operationName}($input: ${inputType}!) {
            ${operationName}${signature} ${selection}
        }
    `;
}