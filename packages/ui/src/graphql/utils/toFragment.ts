/**
 * Helper function for generating a GraphQL fragment using a template literal.
 * 
 * Example: 'boop', 'Beep', {
 *   id
 *   translations {
 *       id
 *       language
 *       description
 *   }
 * } => fragment boop on Beep {
 *   id
 *   translations {
 *       id
 *       language
 *       description
 *   }
 * }
 * 
 * @param fragmentName The name of the fragment. Must be unique across the entire graphql schema.
 * @param selectionSet A tuple of the object type and selection set (i.e. what fields to include) for the fragment.
 * @returns a graphql-tag string for the fragment.
 */
export const toFragment = (
    fragmentName: string,
    [type, selectionSet]: readonly [string, string]
) => {
    return `fragment ${fragmentName} on ${type} ${selectionSet}`;
}