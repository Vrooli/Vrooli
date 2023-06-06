import { VisibilityType } from "@local/shared";
import { getUser } from "../auth";
import { addSupplementalFields, combineQueries, modelToGql, selectHelper, toPartialGqlInfo, visibilityBuilder } from "../builders";
import { PaginatedSearchResult, PartialGraphQLInfo } from "../builders/types";
import { CustomError, logger } from "../events";
import { getSearchStringQuery } from "../getters";
import { ObjectMap } from "../models/base";
import { Searcher } from "../models/types";
import { SearchMap } from "../utils";
import { SortMap } from "../utils/sortMap";
import { ReadManyHelperProps } from "./types";

const DEFAULT_TAKE = 20;
const MAX_TAKE = 100;

/**
 * Helper function for reading many objects in a single line.
 * Cursor-based search. Supports pagination, sorting, and filtering by string.
 * NOTE: Permissions queries should be passed into additionalQueries
 * @returns Paginated search result
 */
export async function readManyHelper<Input extends { [x: string]: any }>({
    additionalQueries,
    addSupplemental = true,
    info,
    input,
    objectType,
    prisma,
    req,
    visibility = VisibilityType.Public,
}: ReadManyHelperProps<Input>): Promise<PaginatedSearchResult> {
    const userData = getUser(req);
    const model = ObjectMap[objectType];
    if (!model) throw new CustomError("0349", "InternalError", req.languages, { objectType });
    // Partially convert info type
    const partialInfo = toPartialGqlInfo(info, model.format.gqlRelMap, req.languages, true);
    // Make sure ID is in partialInfo, since this is required for cursor-based search
    partialInfo.id = true;
    const searcher: Searcher<any, any> | undefined = model.search;
    // Check take limit
    if (Number.isInteger(input.take) && input.take > MAX_TAKE) {
        throw new CustomError("0389", "InternalError", req.languages, { objectType, take: input.take });
    }
    // Determine text search query
    const searchQuery = (input.searchString && searcher?.searchStringQuery) ? getSearchStringQuery({ objectType: model.__typename, searchString: input.searchString }) : undefined;
    // Loop through search fields and add each to the search query, 
    // if the field is specified in the input
    const customQueries: { [x: string]: any }[] = [];
    if (searcher) {
        for (const field of Object.keys(searcher.searchFields)) {
            if (input[field as string] !== undefined) {
                customQueries.push(SearchMap[field as string](input[field], userData, model.__typename));
            }
        }
    }
    if (searcher?.customQueryData) {
        customQueries.push(searcher.customQueryData(input, userData));
    }
    // Create query for visibility, if supported
    const visibilityQuery = visibilityBuilder({ objectType, userData, visibility: input.visibility ?? visibility });
    // Combine queries
    const where = combineQueries([additionalQueries, searchQuery, visibilityQuery, ...customQueries]);
    // Determine sort order
    // Make sure sort field is valid
    const orderBy = SortMap[input.sortBy ?? searcher!.defaultSort] ?? undefined;
    // Find requested search array
    const select = selectHelper(partialInfo);
    let searchResults: any[];
    try {
        searchResults = await (model.delegate(prisma) as any).findMany({
            where,
            orderBy,
            take: Number.isInteger(input.take) ? input.take : DEFAULT_TAKE,
            skip: input.after ? 1 : undefined, // First result on cursored requests is the cursor, so skip it
            cursor: input.after ? {
                id: input.after,
            } : undefined,
            ...select,
        });
    } catch (error) {
        logger.error("readManyHelper: Failed to find searchResults", { trace: "0383", error, objectType, ...select, where, orderBy });
        throw new CustomError("0383", "InternalError", req.languages, { objectType });
    }
    let hasNextPage = false;
    let endCursor: string | null = null;
    // If there are results, find the end cursor and hasNextPage flag
    if (searchResults.length > 0) {
        // Find cursor
        endCursor = searchResults[searchResults.length - 1].id;
        // Query after the cursor to check if there are more results
        const nextPage = await (model.delegate(prisma) as any).findMany({
            take: 1,
            cursor: {
                id: endCursor,
            },
        });
        hasNextPage = nextPage.length > 0;
    }
    //TODO validate that the user has permission to read all of the results, including relationships
    // Add supplemental fields, if requested
    if (addSupplemental) {
        searchResults = searchResults.map(n => modelToGql(n, partialInfo as PartialGraphQLInfo));
        searchResults = await addSupplementalFields(prisma, userData, searchResults, partialInfo);
    }
    // Return formatted for GraphQL
    return {
        __typename: `${model.__typename}SearchResult` as const,
        pageInfo: {
            __typename: "PageInfo" as const,
            hasNextPage,
            endCursor,
        },
        edges: searchResults.map((result: any) => ({
            __typename: `${model.__typename}Edge` as const,
            cursor: result.id,
            node: result,
        })),
    };
}
