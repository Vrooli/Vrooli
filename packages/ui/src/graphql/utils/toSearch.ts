import { GqlPartial } from "types";



/**
 * Helper function for generating a GraphQL selection of a 
 * paginated search result using a template literal.
 * 
 * @param partial A GqlPartial object for the object type being searched
 * @returns A new GqlPartial object like the one passed in, but wrapped 
 * in a paginated search result.
 */
export const toSearch = <
    GqlObject extends { __typename: string }
>(
    partial: GqlPartial<GqlObject>,
): [GqlPartial<any>, 'list'] => {
    return [{
        __typename: `${partial.__typename}SearchResult`,
        list: {
            edges: {
                cursor: true,
                node: asdfasfd, //TODO combine list with common here
            },
            pageInfo: {
                endCursor: true,
                hasNextPage: true,
            }
        }
    }, 'list']
}