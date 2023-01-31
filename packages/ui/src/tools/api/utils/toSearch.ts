import { GqlPartial } from "../types";
import { relPartial } from "./relPartial";



/**
 * Helper function for generating a GraphQL selection of a 
 * paginated search result using a template literal.
 * 
 * @param partial A GqlPartial object for the object type being searched
 * @returns A new GqlPartial object like the one passed in, but wrapped 
 * in a paginated search result.
 */
export const toSearch = async <
    GqlObject extends { __typename: string }
>(
    partial: GqlPartial<GqlObject>,
): Promise<[GqlPartial<any>, 'list']> => {
    // Combine and remove fragments, so we can put them in the top level
    const { __define, ...node } = await relPartial(partial, 'list');
    return [{
        __typename: `${partial.__typename}SearchResult`,
        list: {
            __define,
            edges: {
                cursor: true,
                node,
            },
            pageInfo: {
                endCursor: true,
                hasNextPage: true,
            }
        }
    }, 'list']
}