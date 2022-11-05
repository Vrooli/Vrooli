import { CODE } from "@shared/consts";
import { CustomError, genErrorCode } from "../../events";
import { getUserId, toPartialGraphQLInfo, onlyValidIds, timeFrameToPrisma, combineQueries, selectHelper, modelToGraphQL, addSupplementalFields } from "../builder";
import { PaginatedSearchResult, PartialGraphQLInfo, SearchInputBase } from "../types";
import { ReadManyHelperProps } from "./types";

/**
 * Helper function for reading many objects in a single line.
 * Cursor-based search. Supports pagination, sorting, and filtering by string.
 * NOTE: Permissions queries should be passed into additionalQueries
 * @returns Paginated search result
 */
export async function readManyHelper<GraphQLModel, SearchInput extends SearchInputBase<any>>({
    additionalQueries,
    addSupplemental = true,
    info,
    input,
    model,
    prisma,
    req,
}: ReadManyHelperProps<GraphQLModel, SearchInput>): Promise<PaginatedSearchResult> {
    const userId = getUserId(req);
    // Partially convert info type
    let partialInfo = toPartialGraphQLInfo(info, model.format.relationshipMap);
    if (!partialInfo)
        throw new CustomError(CODE.InternalError, 'Could not convert info to partial select', { code: genErrorCode('0023') });
    // Make sure ID is in partialInfo, since this is required for cursor-based search
    partialInfo.id = true;
    // Create query for specified ids
    const idQuery = (Array.isArray(input.ids)) ? ({ id: { in: onlyValidIds(input.ids) } }) : undefined;
    // Determine text search query
    const searchQuery = (input.searchString && model.search?.getSearchStringQuery) ? model.search.getSearchStringQuery(input.searchString) : undefined;
    // Determine createdTimeFrame query
    const createdQuery = timeFrameToPrisma('created_at', input.createdTimeFrame);
    // Determine updatedTimeFrame query
    const updatedQuery = timeFrameToPrisma('updated_at', input.updatedTimeFrame);
    // Create type-specific queries
    let typeQuery = model.search?.customQueries ? model.search.customQueries(input, userId) : undefined;
    // Combine queries
    const where = combineQueries([additionalQueries, idQuery, searchQuery, createdQuery, updatedQuery, typeQuery]);
    // Determine sort order
    const orderBy = model.search?.getSortQuery ? model.search.getSortQuery(input.sortBy ?? model.search.defaultSort) : undefined;
    // Find requested search array
    const searchResults = await (model.prismaObject(prisma) as any).findMany({
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
        const hasNextPage = await (model.prismaObject(prisma) as any).findMany({
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
    formattedNodes = await addSupplementalFields(prisma, userId, formattedNodes, partialInfo);
    return { pageInfo: paginatedResults.pageInfo, edges: paginatedResults.edges.map(({ node, ...rest }) => ({ node: formattedNodes.shift(), ...rest })) };
}