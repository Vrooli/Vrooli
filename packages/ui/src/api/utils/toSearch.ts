import { GqlPartial } from "types";
import { partialCombine } from "./partialCombine";



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
    // Combine the common and list selections from the partial
    const combinedSelection = partialCombine(partial.list ?? {}, partial.common ?? {});
    // Remove __define (i.e. fragments) from the combined selection, so we can put it in the top level
    const { __define, ...node } = combinedSelection;
    return [{
        __typename: `${partial.__typename}SearchResult`,
        list: {
            __define: __define as any,
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