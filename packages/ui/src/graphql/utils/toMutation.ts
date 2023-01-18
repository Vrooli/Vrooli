import { gql } from 'graphql-tag';
import { toFragment } from './toFragment';

/**
 * Helper function for generating a GraphQL muation for a given endpoint.
 * 
 * Example: toMutation('api', 'FindByIdInput', fullFields[1]) => 
 * gql`
 *      ${fullFields}
 *      mutation api($input: FindByIdInput!) {
 *          api(input: $input) {
 *              ...fullFields
 *          }
 *      }
 *  `
 * 
 * @param endpointName The name of the operation (endpoint).
 * @param inputType The name of the input type.
 * @param fragments An array of GraphQL fragments to include in the operation.
 * @param selectionSet The selection set for the operation.
 * @returns a tuple of: a graphql-tag string for the endpoint, and the endpoint's name.
 */
export const toMutation = <Endpoint extends string>(
    endpointName: Endpoint,
    inputType: string | null,
    selectionSet: string | null,
    fragments: Array<readonly [string, string]> = [],
) => {
    console.log('toMutation start', endpointName, inputType)
    let fragmentStrings: string[] = [];
    for (let i = 0; i < fragments.length; i++) {
        fragmentStrings.push(`${toFragment(endpointName + i, fragments[i])}\n`);
    }
    const signature = inputType ? `(input: $input)` : '';
    console.log(`${fragmentStrings.join('\n')}
mutation ${endpointName}($input: ${inputType}!) {
    ${endpointName}${signature} ${selectionSet ?? ''}
}`)
    return [gql`${fragmentStrings.join('\n')}
mutation ${endpointName}($input: ${inputType}!) {
    ${endpointName}${signature} ${selectionSet ?? ''}
}`, endpointName] as const;
}