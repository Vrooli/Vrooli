import { getUser } from "../auth";
import { addSupplementalFields, combineQueries, modelToGraphQL, onlyValidIds, selectHelper, timeFrameToPrisma, toPartialGraphQLInfo } from "../builders";
import { PaginatedSearchResult, PartialGraphQLInfo } from "../builders/types";
import { CustomError, logger } from "../events";
import { getSearchStringQuery } from "../getters";
import { ObjectMap } from "../models";
import { Searcher } from "../models/types";
import { SearchMap } from "../utils";
import { SortMap } from "../utils/sortMap";
import { ReadManyHelperProps } from "./types";

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
}: ReadManyHelperProps<Input>): Promise<PaginatedSearchResult> {
    const userData = getUser(req);
    const model = ObjectMap[objectType];
    if (!model) throw new CustomError('0349', 'InternalError', req.languages, { objectType });
    // Partially convert info type
    let partialInfo = toPartialGraphQLInfo(info, model.format.gqlRelMap, req.languages, true);
    // Make sure ID is in partialInfo, since this is required for cursor-based search
    partialInfo.id = true;
    const searcher: Searcher<any> | undefined = model.search;
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
    // Combine queries
    // TODO for monrning: compare to popular endpoint in feed.ts. AdditionQueries is missing isPrivate: false. customQueries is missing multiple things
    console.log('going to combine queries: ', JSON.stringify(additionalQueries), searchQuery, JSON.stringify(customQueries));
    const where = combineQueries([additionalQueries, searchQuery, ...customQueries]);
    // Determine sort order
    // Make sure sort field is valid
    const orderByField = searcher ? input.sortBy ?? searcher.defaultSort : undefined;
    const orderByIsValid = searcher ? searcher.sortBy[orderByField] === undefined : false;
    const orderBy = orderByIsValid ? SortMap[input.sortBy ?? searcher!.defaultSort] : undefined;
    // Find requested search array
    console.log('readmanyhelper before selectHelper', objectType, JSON.stringify(partialInfo), '\n\n');
    const select = selectHelper(partialInfo);
    console.log('readmanyhelper after selectHelper', objectType, JSON.stringify(select));
    let searchResults: any[];
    try {
        searchResults = await (model.delegate(prisma) as any).findMany({
            where,
            orderBy,
            take: input.take ?? 20,
            skip: input.after ? 1 : undefined, // First result on cursored requests is the cursor, so skip it
            cursor: input.after ? {
                id: input.after
            } : undefined,
            ...select
        });
    } catch (error) {
        logger.error('readManyHelper: Failed to find searchResults', { trace: '0383', error, objectType, ...select, where, orderBy });
        throw new CustomError('0383', 'InternalError', req.languages, { objectType });
    }
    console.log('readmanyhelper after searchResults', JSON.stringify(searchResults))
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
            __typename: `${model.__typename}SearchResult` as const,
            pageInfo: {
                __typename: 'PageInfo' as const,
                hasNextPage: hasNextPage.length > 0,
                endCursor: cursor,
            },
            edges: searchResults.map((result: any) => ({
                __typename: `${model.__typename}Edge` as const,
                cursor: result.id,
                node: result,
            }))
        }
    }
    // If there are no results
    else {
        paginatedResults = {
            __typename: `${model.__typename}SearchResult` as const,
            pageInfo: {
                __typename: 'PageInfo' as const,
                endCursor: null,
                hasNextPage: false,
            },
            edges: []
        }
    }
    console.log('readmanyhelper yeeeee 1', addSupplemental, JSON.stringify(paginatedResults), '\n');
    //TODO validate that the user has permission to read all of the results, including relationships
    // If not adding supplemental fields, return the paginated results
    if (!addSupplemental) return paginatedResults;
    // Return formatted for GraphQL
    let formattedNodes = paginatedResults.edges.map(({ node }) => node);
    console.log('readmanyhelper yeeeee 2', JSON.stringify(formattedNodes), '\n');
    formattedNodes = formattedNodes.map(n => modelToGraphQL(n, partialInfo as PartialGraphQLInfo));
    console.log('readmanyhelper yeeeee 3', JSON.stringify(formattedNodes), '\n');
    formattedNodes = await addSupplementalFields(prisma, userData, formattedNodes, partialInfo);
    console.log('readmanyhelper yeeeee 4', JSON.stringify(formattedNodes), '\n');
    return {
        ...paginatedResults,
        pageInfo: paginatedResults.pageInfo,
        edges: paginatedResults.edges.map(({ node, ...rest }) => ({ node: formattedNodes.shift(), ...rest }))
    };
}