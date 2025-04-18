import { ModelType, PageInfo, TimeFrame, ViewFor, camelCase } from "@local/shared";
import { SessionService } from "../auth/session.js";
import { combineQueries } from "../builders/combineQueries.js";
import { InfoConverter, addSupplementalFields } from "../builders/infoConverter.js";
import { timeFrameToSql } from "../builders/timeFrame.js";
import { PaginatedSearchResult, PartialApiInfo, PrismaDelegate } from "../builders/types.js";
import { visibilityBuilderPrisma } from "../builders/visibilityBuilder.js";
import { DbProvider } from "../db/provider.js";
import { SqlBuilder } from "../db/sqlBuilder.js";
import { CustomError } from "../events/error.js";
import { logger } from "../events/logger.js";
import { getIdFromHandle } from "../getters/getIdFromHandle.js";
import { getLatestVersion } from "../getters/getLatestVersion.js";
import { getSearchStringQuery } from "../getters/getSearchStringQuery.js";
import { ModelMap } from "../models/base/index.js";
import { ViewModelLogic } from "../models/base/types.js";
import { Searcher } from "../models/types.js";
import { RecursivePartial } from "../types.js";
import { SearchEmbeddingsCache } from "../utils/embeddings/cache.js";
import { getEmbeddings } from "../utils/embeddings/getEmbeddings.js";
import { getAuthenticatedData } from "../utils/getAuthenticatedData.js";
import { SearchMap } from "../utils/searchMap.js";
import { SortMap } from "../utils/sortMap.js";
import { permissionsCheck } from "../validators/permissions.js";
import { ReadManyHelperProps, ReadManyWithEmbeddingsHelperProps, ReadOneHelperProps } from "./types.js";

const DEFAULT_TAKE = 20;
const MAX_TAKE = 100;

/**
 * Finds the take to use for a readMany query
 */
export function getDesiredTake(take: unknown, trace?: string): number {
    const desiredTake = Number.isInteger(take) ? take as number : DEFAULT_TAKE;
    if (desiredTake < 1) throw new CustomError("0389", "InternalError", { take, trace });
    if (desiredTake > MAX_TAKE) throw new CustomError("0391", "InternalError", { take, trace });
    return desiredTake;
}

/**
 * Helper function for reading one object in a single line
 * @returns GraphQL response object
 */
export async function readOneHelper<ObjectModel extends { [x: string]: any }>({
    info,
    input,
    objectType,
    req,
}: ReadOneHelperProps): Promise<RecursivePartial<ObjectModel>> {
    const userData = SessionService.getUser(req);
    const model = ModelMap.get(objectType);
    // Validate input. This can be of the form FindByIdInput, FindByIdOrHandleInput, or FindVersionInput
    // Between these, the possible fields are id, idRoot, handle, and handleRoot
    if (!input.id && !input.idRoot && !input.handle && !input.handleRoot)
        throw new CustomError("0019", "IdOrHandleRequired");
    // Partially convert info
    const partialInfo = InfoConverter.get().fromApiToPartialApi(info, model.format.apiRelMap, true);
    // If using idRoot or handleRoot, this means we are requesting a versioned object using data from the root object.
    // To query the version, we must find the latest completed version associated with the root object.
    let id: string | null | undefined;
    if (input.idRoot || input.handleRoot) {
        id = await getLatestVersion({ objectType: objectType as any, idRoot: input.idRoot, handleRoot: input.handleRoot });
    }
    // If using handle, find the id of the object with that handle
    else if (input.handle) {
        id = await getIdFromHandle({ handle: input.handle, objectType });
    }
    // Otherwise, use the id provided
    else {
        id = input.id;
    }
    if (!id)
        throw new CustomError("0434", "NotFound", { objectType });
    // Query for all authentication data
    const authDataById = await getAuthenticatedData({ [model.__typename]: [id] }, userData ?? null);
    if (Object.keys(authDataById).length === 0) {
        throw new CustomError("0021", "NotFound", { id, objectType });
    }
    // Check permissions
    await permissionsCheck(authDataById, { ["Read"]: [id as string] }, {}, userData);
    // Get the Prisma object
    let object: any;
    try {
        object = await (DbProvider.get()[model.dbTable] as PrismaDelegate).findUnique({ where: { id }, ...InfoConverter.get().fromPartialApiToPrismaSelect(partialInfo) });
        if (!object)
            throw new CustomError("0022", "NotFound", { objectType });
    } catch (error) {
        throw new CustomError("0435", "NotFound", { objectType, error });
    }
    // Return formatted for GraphQL
    const formatted = InfoConverter.get().fromDbToApi(object, partialInfo) as RecursivePartial<ObjectModel>;
    // If logged in and object tracks view counts, add a view
    if (userData?.id && objectType in ViewFor) {
        ModelMap.get<ViewModelLogic>("View").view(userData, { forId: object.id, viewFor: objectType as ViewFor });
    }
    const result = (await addSupplementalFields(userData, [formatted], partialInfo))[0] as RecursivePartial<ObjectModel>;
    return result;
}


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
    req,
    visibility,
}: ReadManyHelperProps<Input>): Promise<PaginatedSearchResult> {
    const userData = SessionService.getUser(req);
    const model = ModelMap.get(objectType);
    // Partially convert info type
    const partialInfo = InfoConverter.get().fromApiToPartialApi(info, model.format.apiRelMap, true);
    // Make sure ID is in partialInfo, since this is required for cursor-based search
    partialInfo.id = true;
    const searcher: Searcher<any, any> | undefined = model.search;
    const searchData = {
        objectType: model.__typename,
        req,
        searchInput: input,
        userData,
        visibility: input.visibility ?? visibility,
    };
    // Determine text search query
    const searchQuery = (input.searchString && searcher?.searchStringQuery) ? getSearchStringQuery({ objectType: model.__typename, searchString: input.searchString }) : undefined;
    // Loop through search fields and add each to the search query, 
    // if the field is specified in the input
    const customQueries: object[] = [];
    if (searcher) {
        for (const field of Object.keys(searcher.searchFields)) {
            const fieldInput = input[field];
            const searchMapper = SearchMap[field];
            if (fieldInput !== undefined && searchMapper !== undefined) {
                customQueries.push(searchMapper(fieldInput, searchData));
            }
        }
    }
    // Create query for visibility, if supported
    const { query: visibilityQuery } = visibilityBuilderPrisma(searchData);
    // Combine queries
    const where = combineQueries([additionalQueries, searchQuery, visibilityQuery, ...customQueries]);
    // Determine sortBy, orderBy, and take
    const sortBy = input.sortBy ?? searcher?.defaultSort;
    const orderBy = sortBy in SortMap ? SortMap[sortBy] : undefined;
    const desiredTake = getDesiredTake(input.take, objectType);
    // Find requested search array
    const select = InfoConverter.get().fromPartialApiToPrismaSelect(partialInfo);
    // Search results have at least an id
    let searchResults: Record<string, any>[] = [];
    try {
        searchResults = await (DbProvider.get()[model.dbTable] as PrismaDelegate).findMany({
            where,
            orderBy,
            take: desiredTake + 1, // Take one extra so we can determine if there is a next page
            skip: input.after ? 1 : undefined, // First result on cursored requests is the cursor, so skip it
            cursor: input.after ? {
                id: input.after,
            } : undefined,
            ...select,
        });
    } catch (error) {
        logger.error("readManyHelper: Failed to find searchResults", { trace: "0383", error, objectType, select, where, orderBy });
        throw new CustomError("0383", "InternalError", { objectType });
    }
    let hasNextPage = false;
    let endCursor: string | null = null;
    // Check if there's an extra item (indicating more results)
    if (searchResults.length > desiredTake) {
        hasNextPage = true;
        searchResults.pop(); // remove the extra item
    }
    if (searchResults.length > 0) {
        endCursor = searchResults[searchResults.length - 1].id;
    }
    //TODO validate that the user has permission to read all of the results, including relationships
    // Add supplemental fields, if requested
    if (addSupplemental) {
        searchResults = searchResults.map(n => InfoConverter.get().fromDbToApi(n, partialInfo as PartialApiInfo));
        searchResults = await addSupplementalFields(userData, searchResults, partialInfo);
    }
    // Return formatted for GraphQL
    return {
        __typename: `${model.__typename}SearchResult` as const,
        pageInfo: {
            __typename: "PageInfo" as const,
            hasNextPage,
            endCursor,
        },
        edges: searchResults.map((result) => ({
            __typename: `${model.__typename}Edge` as const,
            cursor: result.id,
            node: result,
        })),
    };
}

export type ReadManyAsFeedResult = {
    pageInfo: PageInfo;
    nodes: object[];
}

/**
 * Helper function for reading many objects and converting them to a GraphQL response
 * (except for supplemental fields). This is useful when querying feeds
 */
export async function readManyAsFeedHelper<Input extends { [x: string]: any }>({
    additionalQueries,
    info,
    input,
    objectType,
    req,
    visibility,
}: Omit<ReadManyHelperProps<Input>, "addSupplemental">): Promise<ReadManyAsFeedResult> {
    const readManyResult = await readManyHelper({
        additionalQueries,
        addSupplemental: false, // Skips conversion to API format
        info,
        input,
        objectType,
        req,
        visibility,
    });
    const format = ModelMap.get(objectType).format;
    const nodes = readManyResult.edges.map(({ node }: any) =>
        InfoConverter.get().fromDbToApi(node, InfoConverter.get().fromApiToPartialApi(info, format.apiRelMap, true))) as any[];
    return {
        pageInfo: readManyResult.pageInfo,
        nodes,
    };
}

/**
 * Helper function for reading many embeddable objects in a single line.
 * Cursor-based search. Supports pagination, sorting, and filtering by string.
 * 
 * NOTE: This mode uses raw SQL to search for results, which comes with more limitations than the normal search mode.
 * @returns Paginated search result
 */
export async function readManyWithEmbeddingsHelper<Input extends { [x: string]: any }>({
    fetchMode = "full",
    info,
    input,
    objectType,
    req,
    visibility,
}: ReadManyWithEmbeddingsHelperProps<Input>): Promise<PaginatedSearchResult> {
    const userData = SessionService.getUser(req);
    const model = ModelMap.get(objectType);
    const desiredTake = getDesiredTake(input.take, objectType);
    const searchStringTrimmed = (input.searchString ?? "").trim();
    // If there isn't a search string, we can perform non-embedding search
    if (searchStringTrimmed.length === 0) {
        // Call normal search function
        const results = await readManyHelper({
            addSupplemental: fetchMode === "full",
            info,
            input,
            objectType,
            req,
            visibility,
        });
        return results;
    }

    const offset = input.offset ?? 0;
    const sortOption = input.sortBy ?? model.search?.defaultSort;
    const take = desiredTake + 1; // Add 1 for hasNextPage check

    let idsNeeded = take;
    let finalResults: Record<string, any>[] = [];

    // Get visibility query
    const searchData = {
        objectType,
        req,
        searchInput: input,
        userData,
        visibility: input.visibility ?? visibility,
    };
    const { query: visibilityQuery, visibilityUsed } = visibilityBuilderPrisma(searchData);

    // Check cache for previously fetched IDs for this specific situation
    const cacheKey = SearchEmbeddingsCache.createKey({ objectType, searchString: searchStringTrimmed, sortOption, userId: userData?.id, visibility: visibilityUsed });
    const cachedResults = await SearchEmbeddingsCache.check({ cacheKey, offset, take });

    // Add all cached results to the final results
    if (cachedResults && cachedResults.length > 0) {
        finalResults = cachedResults;
        idsNeeded = take - cachedResults.length;
    }

    // If we still need more results (i.e. no cached results or only some cached), fetch them
    if (idsNeeded > 0) {
        const newOffset = offset + (cachedResults ? cachedResults.length : 0);
        const translationObjectType = (objectType + "Translation") as ModelType;
        // Create builder to construct the SQL query to fetch missing data
        const builder = new SqlBuilder(objectType);
        // Make sure we select the ID
        builder.addSelect(objectType, "id");
        // Join with translation table, as that's where the embeddings are stored
        builder.addJoin(
            translationObjectType,
            "INNER",
            SqlBuilder.equals(builder.field(objectType, "id"), builder.field(translationObjectType, camelCase(objectType) + "Id")),
        );
        // Get the embedding for the search string
        const embeddings = await getEmbeddings(objectType, [searchStringTrimmed]);
        // Convert it to an equation based on the sort option
        // TODO embed field will change if versioned. Should update embedDistance, addSelect, and addWhere accordingly
        builder.embedPoints(translationObjectType, objectType, embeddings[0] as number[], sortOption);
        // Get date queries for restricting search results by time
        const createdAtDateLimit = timeFrameToSql("created_at", input.createdTimeFrame as TimeFrame | undefined);
        const updatedAtDateLimit = timeFrameToSql("updated_at", input.updatedTimeFrame as TimeFrame | undefined);
        // Add dates to SELECT and WHERE clauses
        if (createdAtDateLimit) {
            builder.addSelect(objectType, "created_at");
            builder.addWhere(createdAtDateLimit);
        }
        if (updatedAtDateLimit) {
            builder.addSelect(objectType, "updated_at");
            builder.addWhere(updatedAtDateLimit);
        }
        builder.buildQueryFromPrisma(visibilityQuery); //TODO
        // Set order by, limit, and offset
        builder.addOrderByRaw("points " + (sortOption.endsWith("Desc") ? "DESC" : "ASC"));
        builder.setLimit(take);
        builder.setOffset(offset);
        const rawQuery = builder.serialize();

        try {
            // Should be safe to use $queryRawUnsafe in this context, as the only user input is 
            // the search string, and that has been converted to embeddings
            const additionalResults = await DbProvider.get().$queryRawUnsafe(rawQuery) as Record<string, any>[];
            finalResults = [...finalResults, ...additionalResults];
            // Cache the newly fetched results
            await SearchEmbeddingsCache.set({
                cacheKey,
                offset: newOffset,
                take: additionalResults.length,
                results: additionalResults.map((res: any) => ({ id: res.id })),
            });
        } catch (error) {
            logger.error("readManyWithEmbeddingsHelper: Failed to execute raw query", { trace: "0384", error, objectType, rawQuery });
            throw new CustomError("0384", "InternalError", { objectType });
        }
    }

    // Remove last result if we fetched more than we needed (i.e. hasNextPage is true)
    const hasNextPage = finalResults.length > desiredTake;
    if (hasNextPage) {
        finalResults.pop();
    }

    // If fetch mode is ids, return the IDs
    if (fetchMode === "ids") {
        // Return only the IDs
        return {
            __typename: `${model.__typename}SearchResult` as const,
            pageInfo: {
                __typename: "PageInfo" as const,
                hasNextPage,
                endCursor: null, // Only used for cursor-based search
            },
            edges: finalResults.map(t => ({
                __typename: `${model.__typename}Edge` as const,
                cursor: t.id,
                node: { id: t.id }, // Returning only IDs
            })),
        };
    }

    // Fetch additional data for the search results
    const partialInfo = InfoConverter.get().fromApiToPartialApi(info, model.format.apiRelMap, true);
    const select = InfoConverter.get().fromPartialApiToPrismaSelect(partialInfo);
    finalResults = await (DbProvider.get()[model.dbTable] as PrismaDelegate).findMany({
        where: { id: { in: finalResults.map(t => t.id) } },
        ...select,
    });
    //TODO validate that the user has permission to read all of the results, including relationships
    // Return formatted for GraphQL
    let formattedNodes = finalResults.map(n => InfoConverter.get().fromDbToApi(n, partialInfo as PartialApiInfo));
    // If fetch mode is "full", add supplemental fields
    if (fetchMode === "full") {
        formattedNodes = await addSupplementalFields(userData, formattedNodes, partialInfo);
    }
    // Construct and return the final response
    const formattedNodesMap = new Map(formattedNodes.map(n => [n.id, n]));
    return {
        __typename: `${model.__typename}SearchResult` as const,
        pageInfo: {
            __typename: "PageInfo" as const,
            hasNextPage,
            endCursor: null, // Only used for cursor-based search
        },
        edges: finalResults.map(t => ({
            __typename: `${model.__typename}Edge` as const,
            cursor: t.id,
            node: formattedNodesMap.get(t.id),
        })),
    };
}
