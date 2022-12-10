import { getUser } from "../auth";
import { addSupplementalFields, combineQueries, modelToGraphQL, onlyValidIds, selectHelper, timeFrameToPrisma, toPartialGraphQLInfo } from "../builders";
import { PaginatedSearchResult, PartialGraphQLInfo } from "../builders/types";
import { CustomError } from "../events";
import { getSearchString } from "../getters";
import { ObjectMap } from "../models";
import { Searcher } from "../models/types";
import { ReadManyHelperProps } from "./types";

/**
 * Helper function for reading many objects in a single line.
 * Cursor-based search. Supports pagination, sorting, and filtering by string.
 * NOTE: Permissions queries should be passed into additionalQueries
 * @returns Paginated search result
 */
export async function readManyHelper<GraphQLModel extends { [x: string]: any }>({
    additionalQueries,
    addSupplemental = true,
    info,
    input,
    objectType,
    prisma,
    req,
}: ReadManyHelperProps<GraphQLModel>): Promise<PaginatedSearchResult> {
    const userData = getUser(req);
    const model = ObjectMap[objectType];
    if (!model) throw new CustomError('0349', 'InternalError', req.languages, { objectType });
    // Partially convert info type
    let partialInfo = toPartialGraphQLInfo(info, model.format.relationshipMap, req.languages, true);
    // Make sure ID is in partialInfo, since this is required for cursor-based search
    partialInfo.id = true;
    // Create query for specified ids
    const idQuery = (Array.isArray(input.ids)) ? ({ id: { in: onlyValidIds(input.ids) } }) : undefined;
    const searcher: Searcher<any, any, any, any> | undefined = model.search;
    // Determine text search query
    const searchQuery = (input.searchString && searcher?.searchStringQuery) ? getSearchString({ objectType: model.__typename, searchString: input.searchString }) : undefined;
    // Determine createdTimeFrame query
    const createdQuery = timeFrameToPrisma('created_at', input.createdTimeFrame);
    // Determine updatedTimeFrame query
    const updatedQuery = timeFrameToPrisma('updated_at', input.updatedTimeFrame);
    // Create type-specific queries
    let typeQuery = searcher?.customQueries ? searcher.customQueries(input, userData) : undefined;
    // Combine queries
    const where = combineQueries([additionalQueries, idQuery, searchQuery, createdQuery, updatedQuery, typeQuery]);
    // Determine sort order
    const orderBy = searcher?.sortMap ? searcher.sortMap[input.sortBy ?? searcher.defaultSort] : undefined;
    // Find requested search array
    const searchResults = await (model.delegate(prisma) as any).findMany({
        where,
        orderBy,
        take: input.take ?? 20,
        skip: input.after ? 1 : undefined, // First result on cursored requests is the cursor, so skip it
        cursor: input.after ? {
            id: input.after
        } : undefined,
        ...selectHelper(partialInfo)
    });
    // If there are results
    let paginatedResults: PaginatedSearchResult;
    if (searchResults.length > 0) {
        // Find cursor
        const cursor = searchResults[searchResults.length - 1].id;
        // Query after the cursor to check if there are more results
        const hasNextPage = await (model.delegate(prisma) as any).findMany({
            take: 1,
            cursor: {
                id: cursor
            }
        });
        paginatedResults = {
            pageInfo: {
                hasNextPage: hasNextPage.length > 0,
                endCursor: cursor,
            },
            edges: searchResults.map((result: any) => ({
                cursor: result.id,
                node: result,
            }))
        }
    }
    // If there are no results
    else {
        paginatedResults = {
            pageInfo: {
                endCursor: null,
                hasNextPage: false,
            },
            edges: []
        }
    }
    // If not adding supplemental fields, return the paginated results
    if (!addSupplemental) return paginatedResults;
    // Return formatted for GraphQL
    let formattedNodes = paginatedResults.edges.map(({ node }) => node);
    formattedNodes = formattedNodes.map(n => modelToGraphQL(n, partialInfo as PartialGraphQLInfo));
    formattedNodes = await addSupplementalFields(prisma, userData, formattedNodes, partialInfo);
    return { pageInfo: paginatedResults.pageInfo, edges: paginatedResults.edges.map(({ node, ...rest }) => ({ node: formattedNodes.shift(), ...rest })) };
}