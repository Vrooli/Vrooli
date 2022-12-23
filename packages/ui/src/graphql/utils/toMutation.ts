import { gql } from 'graphql-tag';
import { toFragment } from './toFragment';

/**
 * Helper function for generating a GraphQL muation for a given endpoint.
 * 
 * Example: toMutation('api', 'FindByIdInput', [fullFields], `...fullFields`) => 
 * gql`
 *      ${fullFields}
 *      mutation api($input: FindByIdInput!) {
 *          api(input: $input) {
 *              ...fullFields
 *          }
 *      }
 *  `
 * 
 * @param operationName The name of the operation.
 * @param inputType The name of the input type.
 * @param fragments An array of GraphQL fragments to include in the operation.
 * @param selectionSet The selection set for the operation.
 * @returns a graphql-tag string for the endpoint.
 */
export const toMutation = (
    operationName: string,
    inputType: string | null,
    fragments: Array<readonly [string, string]>,
    selectionSet: string | null
) => {
    let fragmentStrings: string[] = [];
    for (let i = 0; i < fragments.length; i++) {
        fragmentStrings.push(`${toFragment(operationName + i, fragments[i])}\n`);
    }
    const selection = selectionSet ? `{\n${selectionSet}\n}` : '';
    const signature = inputType ? `($input: ${inputType}!)` : '';
    return gql`
        ${fragmentStrings}
        mutation ${operationName}($input: ${inputType}!) {
            ${operationName}${signature} ${selection}
        }
    `;
}