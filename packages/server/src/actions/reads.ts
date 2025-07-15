import { type FindByIdInput, type FindByPublicIdInput, type FindVersionInput, ModelType, type PageInfo, type TimeFrame, ViewFor, VisibilityType, camelCase, validatePK, validatePublicId } from "@vrooli/shared";
import { SessionService } from "../auth/session.js";
import { combineQueries } from "../builders/combineQueries.js";
// AI_CHECK: TYPE_SAFETY=phase2-actions | LAST: 2025-07-04 - Improved type safety in read operations
import { InfoConverter, addSupplementalFields } from "../builders/infoConverter.js";
import { timeFrameToSql } from "../builders/timeFrame.js";
import { type PaginatedSearchResult, type PartialApiInfo, type PrismaDelegate } from "../builders/types.js";
import { visibilityBuilderPrisma } from "../builders/visibilityBuilder.js";
import { DbProvider } from "../db/provider.js";
import { SqlBuilder } from "../db/sqlBuilder.js";
import { CustomError } from "../events/error.js";
import { logger } from "../events/logger.js";
import { getSearchStringQuery } from "../getters/getSearchStringQuery.js";
import { ModelMap } from "../models/base/index.js";
import { type ViewModelLogic } from "../models/base/types.js";
import { type Searcher } from "../models/types.js";
import { EmbeddingService } from "../services/embedding.js";
import { type RecursivePartial } from "../types.js";
import { getAuthenticatedData } from "../utils/getAuthenticatedData.js";
import { SearchMap } from "../utils/searchMap.js";
import { SortMap } from "../utils/sortMap.js";
import { permissionsCheck } from "../validators/permissions.js";
import { type ReadManyHelperProps, type ReadManyWithEmbeddingsHelperProps, type ReadOneHelperProps } from "./types.js";

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

const supportHandles: ModelType[] = [ModelType.Team, ModelType.User];
const supportPublicIds: ModelType[] = [ModelType.Chat, ModelType.Issue, ModelType.Meeting, ModelType.Team, ModelType.Member, ModelType.PullRequest, ModelType.Report, ModelType.Resource, ModelType.ResourceVersion, ModelType.Schedule, ModelType.User];
const supportVersionLabels: ModelType[] = [ModelType.ResourceVersion];

/**
 * Helper function for reading one object in a single line
 * @returns GraphQL response object
 */
// AI_CHECK: TYPE_SAFETY=tier1-1 | LAST: 2025-07-03 - Fixed visibility type casting, searchString casting, cursor type assertions, and offset type assertions
export async function readOneHelper<ObjectModel extends Record<string, unknown>>({
    info,
    input,
    objectType,
    req,
}: ReadOneHelperProps): Promise<RecursivePartial<ObjectModel>> {
    const userData = SessionService.getUser(req);
    const model = ModelMap.get(objectType);
    // Build where clause
    const { id, publicId, versionLabel } = (input as FindByIdInput & FindByPublicIdInput & FindVersionInput);
    let where: Record<string, unknown> = {};
    // Only allow one publicId or id
    // Check for publicId (sometimes it's a handle)
    if (publicId && validatePublicId(publicId) && !publicId.startsWith("@") && supportPublicIds.includes(objectType as ModelType)) {
        const selector = { publicId };
        if (supportVersionLabels.includes(objectType as ModelType)) {
            where.root = selector;
        } else {
            where = selector;
        }
    }
    // Check for handle
    else if (publicId && publicId.startsWith("@") && supportHandles.includes(objectType as ModelType)) {
        const selector = { handle: publicId.slice(1) };
        if (supportVersionLabels.includes(objectType as ModelType)) {
            where.root = selector;
        } else {
            where = selector;
        }
    }
    // Check for id
    else if (validatePK(id)) {
        const selector = { id: BigInt(id) };
        if (supportVersionLabels.includes(objectType as ModelType)) {
            where.root = selector;
        } else {
            where = selector;
        }
    }
    // Make sure we have enough information to find the object
    if (Object.keys(where).length === 0) {
        throw new CustomError("0019", "IdOrHandleRequired");
    }
    // For versioned objects, we find the latest public version if no version label is provided
    if (supportVersionLabels.includes(objectType as ModelType)) {
        if (typeof versionLabel === "string" && versionLabel.length > 0) {
            where.versionLabel = versionLabel;
        } else {
            where.isLatestPublic = true;
        }
    }

    // Partially convert info
    const partialInfo = InfoConverter.get().fromApiToPartialApi(info, model.format.apiRelMap, true);

    // Get the Prisma object
    let object: any;
    try {
        const prismaSelect = InfoConverter.get().fromPartialApiToPrismaSelect(partialInfo);
        if (!prismaSelect) {
            throw new CustomError("0348", "InternalError", { objectType });
        }
        object = await (DbProvider.get()[model.dbTable] as PrismaDelegate).findUnique({ where, ...prismaSelect });
        if (!object)
            throw new CustomError("0022", "NotFound", { objectType });
    } catch (error) {
        throw new CustomError("0435", "NotFound", { objectType, error });
    }

    // Check permissions after fetching the object since we have the ID
    const objectId = object?.id?.toString();
    if (!objectId) {
        throw new CustomError("0022", "NotFound", { objectType });
    }
    const authDataById = await getAuthenticatedData({ [model.__typename]: [objectId] }, userData ?? null);
    await permissionsCheck(authDataById, { ["Read"]: [objectId] }, {}, userData);

    // Return formatted for GraphQL
    const formatted = InfoConverter.get().fromDbToApi<ObjectModel>(object, partialInfo);
    // If logged in and object tracks view counts, add a view
    if (userData?.id && objectType in ViewFor) {
        ModelMap.get<ViewModelLogic>("View").view(userData, { forId: object.id, viewFor: objectType as ViewFor });
    }
    const supplemented = await addSupplementalFields(userData, [formatted], partialInfo);
    const result = supplemented[0] as RecursivePartial<ObjectModel>;
    return result;
}


/**
 * Helper function for reading many objects in a single line.
 * Cursor-based search. Supports pagination, sorting, and filtering by string.
 * NOTE: Permissions queries should be passed into additionalQueries
 * @returns Paginated search result
 */
export async function readManyHelper<Input extends Record<string, unknown>>({
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
        visibility: (input.visibility ?? visibility ?? VisibilityType.Public) as VisibilityType,
    };
    // Determine text search query
    const searchQuery = (input.searchString && searcher?.searchStringQuery) ? getSearchStringQuery({ objectType: model.__typename, searchString: input.searchString as string }) : undefined;
    // Loop through search fields and add each to the search query, 
    // if the field is specified in the input
    const customQueries: Record<string, unknown>[] = [];
    if (searcher) {
        for (const field of Object.keys(searcher.searchFields)) {
            const fieldInput = input[field];
            const searchMapper = SearchMap[field];
            if (fieldInput !== undefined && searchMapper !== undefined) {
                customQueries.push(searchMapper(fieldInput, searchData) as Record<string, unknown>);
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
    let searchResults: Record<string, unknown>[] = [];
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
        endCursor = searchResults[searchResults.length - 1].id as string;
    }
    //TODO validate that the user has permission to read all of the results, including relationships
    // Add supplemental fields, if requested
    if (addSupplemental) {
        searchResults = searchResults.map(n => InfoConverter.get().fromDbToApi(n, partialInfo));
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
            cursor: result.id as string,
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
export async function readManyAsFeedHelper<Input extends Record<string, unknown>>({
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
export async function readManyWithEmbeddingsHelper<Input extends Record<string, unknown>>({
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
    const searchStringTrimmed = ((input.searchString as string | undefined) ?? "").trim();
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
    let finalResults: Record<string, unknown>[] = [];

    // Get visibility query
    const searchData = {
        objectType,
        req,
        searchInput: input,
        userData,
        visibility: (input.visibility ?? visibility) as VisibilityType | null | undefined,
    };
    const { query: visibilityQuery, visibilityUsed } = visibilityBuilderPrisma(searchData);

    // Check cache for previously fetched IDs for this specific situation
    const cacheKey = EmbeddingService.createSearchResultCacheKey({ objectType, searchString: searchStringTrimmed, sortOption, userId: userData?.id, visibility: visibilityUsed });
    const cachedResults = await EmbeddingService.checkSearchResultCacheRange({ cacheKey, offset: offset as number, take });

    // Add all cached results to the final results
    if (cachedResults && cachedResults.length > 0) {
        finalResults = cachedResults;
        idsNeeded = take - cachedResults.length;
    }

    // If we still need more results (i.e. no cached results or only some cached), fetch them
    if (idsNeeded > 0) {
        const newOffset = (offset as number) + (cachedResults ? cachedResults.length : 0);
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
        const embeddings = await EmbeddingService.get().getEmbeddings(objectType, [searchStringTrimmed]);
        // Handle potential null embedding if API failed
        if (!embeddings[0]) {
            throw new Error(`Failed to get embedding for search string: ${searchStringTrimmed}`);
        }
        // Convert it to an equation based on the sort option
        // TODO embed field will change if versioned. Should update embedDistance, addSelect, and addWhere accordingly
        builder.embedPoints(translationObjectType, objectType, embeddings[0] as number[], sortOption);
        // Get date queries for restricting search results by time
        const createdAtDateLimit = timeFrameToSql("createdAt", input.createdTimeFrame as TimeFrame | undefined);
        const updatedAtDateLimit = timeFrameToSql("updatedAt", input.updatedTimeFrame as TimeFrame | undefined);
        // Add dates to SELECT and WHERE clauses
        if (createdAtDateLimit) {
            builder.addSelect(objectType, "createdAt");
            builder.addWhere(createdAtDateLimit);
        }
        if (updatedAtDateLimit) {
            builder.addSelect(objectType, "updatedAt");
            builder.addWhere(updatedAtDateLimit);
        }
        // TODO: visibilityBuilderPrisma returns a Prisma query object. SqlBuilder needs raw SQL segments.
        // This needs refactoring. For now, skipping visibility in raw SQL for embeddings search.
        // builder.buildQueryFromPrisma(visibilityQuery);

        // Set order by, limit, and offset
        builder.addOrderByRaw("points " + (sortOption.endsWith("Desc") ? "DESC" : "ASC"));
        builder.setLimit(idsNeeded); // Fetch only the remaining needed IDs
        builder.setOffset(newOffset); // Offset by the number already found in cache
        const rawQuery = builder.serialize();

        try {
            // Should be safe to use $queryRawUnsafe in this context, as the only user input is 
            // the search string, and that has been converted to embeddings
            const additionalResults = await DbProvider.get().$queryRawUnsafe(rawQuery) as Record<string, any>[];

            // Combine cached and newly fetched results
            const combinedResults = [
                ...(cachedResults || []),
                ...additionalResults.map(res => ({ id: res.id.toString() })), // Ensure IDs are strings like cache
            ];

            // Update the cache with the combined results for the *fetched range*
            // Use EmbeddingService.setSearchResultCacheRange
            await EmbeddingService.setSearchResultCacheRange({
                cacheKey,
                offset: newOffset, // Cache starting from where we fetched
                take: additionalResults.length,
                results: additionalResults.map((res: any) => ({ id: res.id.toString() })),
            });

            // Update finalResults (only up to the originally needed 'take' amount)
            finalResults = combinedResults.slice(0, take);

        } catch (error) {
            logger.error("readManyWithEmbeddingsHelper: Failed to execute raw query", { trace: "0384", error, objectType, rawQuery });
            throw new CustomError("0384", "InternalError", { objectType });
        }
    }

    // Determine hasNextPage based on whether we retrieved more than desiredTake
    const hasNextPage = finalResults.length > desiredTake;
    if (hasNextPage) {
        finalResults.pop(); // Remove the extra item used for hasNextPage check
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
                cursor: t.id as string,
                node: { id: t.id }, // Returning only IDs
            })),
        };
    }

    // Fetch additional data for the search results
    const partialInfo = InfoConverter.get().fromApiToPartialApi(info, model.format.apiRelMap, true);
    const select = InfoConverter.get().fromPartialApiToPrismaSelect(partialInfo);
    // Need to handle potential empty finalResults (e.g., cache miss and DB returns nothing)
    if (finalResults.length === 0) {
        return {
            __typename: `${model.__typename}SearchResult` as const,
            pageInfo: { __typename: "PageInfo", hasNextPage: false, endCursor: null },
            edges: [],
        };
    }
    const fetchedNodes = await (DbProvider.get()[model.dbTable] as PrismaDelegate).findMany({
        where: { id: { in: finalResults.map(t => t.id) } },
        ...select,
    });
    //TODO validate that the user has permission to read all of the results, including relationships
    // Return formatted for GraphQL
    let formattedNodes = fetchedNodes.map(n => InfoConverter.get().fromDbToApi(n, partialInfo as PartialApiInfo));
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
            cursor: t.id as string,
            node: formattedNodesMap.get(t.id as string) ?? {} as Record<string, unknown>,
        })),
    };
}
