/**
 * Helper function for generating a GraphQL selection of a 
 * paginated search result using a template literal.
 * 
 * @param selectionSet A tuple of the object type and selection set (i.e. what fields to include) for the fragment.
 * @returns a graphql-tag string for the fragment.
 */
export const toSearch = (
    [_, selectionSet]: readonly [string, string]
) => {
    return `{
        pageInfo {
            endCursor
            hasNextPage
        }
        edges {
            cursor
            node ${selectionSet}
        }
    }`;
}