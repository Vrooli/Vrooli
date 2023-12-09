import { PageInfo, ViewFor, VisibilityType } from "@local/shared";
import { getUser } from "../auth/request";
import { addSupplementalFields } from "../builders/addSupplementalFields";
import { combineQueries } from "../builders/combineQueries";
import { modelToGql } from "../builders/modelToGql";
import { selectHelper } from "../builders/selectHelper";
import { toPartialGqlInfo } from "../builders/toPartialGqlInfo";
import { PaginatedSearchResult, PartialGraphQLInfo } from "../builders/types";
import { visibilityBuilder } from "../builders/visibilityBuilder";
import { CustomError } from "../events/error";
import { logger } from "../events/logger";
import { getIdFromHandle } from "../getters/getIdFromHandle";
import { getLatestVersion } from "../getters/getLatestVersion";
import { getSearchStringQuery } from "../getters/getSearchStringQuery";
import { ModelMap } from "../models/base";
import { ViewModelLogic } from "../models/base/types";
import { Searcher } from "../models/types";
import { RecursivePartial } from "../types";
import { findTags } from "../utils/embeddings/search/tags";
import { getAuthenticatedData } from "../utils/getAuthenticatedData";
import { SearchMap } from "../utils/searchMap";
import { SortMap } from "../utils/sortMap";
import { permissionsCheck } from "../validators/permissions";
import { ReadManyHelperProps, ReadOneHelperProps } from "./types";

const DEFAULT_TAKE = 20;
const MAX_TAKE = 100;

/**
 * Finds the take to use for a readMany query
 */
export const getDesiredTake = (take: unknown, userLanguages?: string[], trace?: string): number => {
    const desiredTake = Number.isInteger(take) ? take as number : DEFAULT_TAKE;
    if (desiredTake < 1) throw new CustomError("0389", "InternalError", userLanguages ?? ["en"], { take, trace });
    if (desiredTake > MAX_TAKE) throw new CustomError("0391", "InternalError", userLanguages ?? ["en"], { take, trace });
    return desiredTake;
};

/**
 * Helper function for reading one object in a single line
 * @returns GraphQL response object
 */
export async function readOneHelper<GraphQLModel extends { [x: string]: any }>({
    info,
    input,
    objectType,
    prisma,
    req,
}: ReadOneHelperProps): Promise<RecursivePartial<GraphQLModel>> {
    const userData = getUser(req.session);
    const model = ModelMap.get(objectType);
    // Validate input. This can be of the form FindByIdInput, FindByIdOrHandleInput, or FindVersionInput
    // Between these, the possible fields are id, idRoot, handle, and handleRoot
    if (!input.id && !input.idRoot && !input.handle && !input.handleRoot)
        throw new CustomError("0019", "IdOrHandleRequired", userData?.languages ?? req.session.languages);
    // Partially convert info
    const partialInfo = toPartialGqlInfo(info, model.format.gqlRelMap, req.session.languages, true);
    // If using idRoot or handleRoot, this means we are requesting a versioned object using data from the root object.
    // To query the version, we must find the latest completed version associated with the root object.
    let id: string | null | undefined;
    if (input.idRoot || input.handleRoot) {
        id = await getLatestVersion({ objectType: objectType as any, prisma, idRoot: input.idRoot, handleRoot: input.handleRoot });
    }
    // If using handle, find the id of the object with that handle
    else if (input.handle) {
        id = await getIdFromHandle({ handle: input.handle, objectType, prisma });
    }
    // Otherwise, use the id provided
    else {
        id = input.id;
    }
    if (!id)
        throw new CustomError("0434", "NotFound", userData?.languages ?? req.session.languages, { objectType });
    // Query for all authentication data
    const authDataById = await getAuthenticatedData({ [model.__typename]: [id] }, prisma, userData ?? null);
    if (Object.keys(authDataById).length === 0) {
        throw new CustomError("0021", "NotFound", userData?.languages ?? req.session.languages, { objectType });
    }
    // Check permissions
    await permissionsCheck(authDataById, { ["Read"]: [id as string] }, {}, userData);
    // Get the Prisma object
    let object: any;
    try {
        object = await model.delegate(prisma).findUnique({ where: { id }, ...selectHelper(partialInfo) });
        if (!object)
            throw new CustomError("0022", "NotFound", userData?.languages ?? req.session.languages, { objectType });
    } catch (error) {
        throw new CustomError("0435", "NotFound", userData?.languages ?? req.session.languages, { objectType, error });
    }
    // Return formatted for GraphQL
    const formatted = modelToGql(object, partialInfo) as RecursivePartial<GraphQLModel>;
    // If logged in and object tracks view counts, add a view
    if (userData?.id && objectType in ViewFor) {
        ModelMap.get<ViewModelLogic>("View").view(prisma, userData, { forId: object.id, viewFor: objectType as ViewFor });
    }
    const result = (await addSupplementalFields(prisma, userData, [formatted], partialInfo))[0] as RecursivePartial<GraphQLModel>;
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
    prisma,
    req,
    visibility = VisibilityType.Public,
}: ReadManyHelperProps<Input>): Promise<PaginatedSearchResult> {
    const userData = getUser(req.session);
    const model = ModelMap.get(objectType);
    // Partially convert info type
    const partialInfo = toPartialGqlInfo(info, model.format.gqlRelMap, req.session.languages, true);
    // Make sure ID is in partialInfo, since this is required for cursor-based search
    partialInfo.id = true;
    const searcher: Searcher<any, any> | undefined = model.search;
    // Determine text search query
    const searchQuery = (input.searchString && searcher?.searchStringQuery) ? getSearchStringQuery({ objectType: model.__typename, searchString: input.searchString }) : undefined;
    // Loop through search fields and add each to the search query, 
    // if the field is specified in the input
    const customQueries: { [x: string]: unknown }[] = [];
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
    // Determine sortBy, orderBy, and take
    const sortBy = input.sortBy ?? searcher?.defaultSort;
    const orderBy = sortBy in SortMap ? SortMap[sortBy] : undefined;
    const desiredTake = getDesiredTake(input.take, req.session.languages, objectType);
    // Find requested search array
    const select = selectHelper(partialInfo);
    // Search results have at least an id
    let searchResults: Record<string, any>[] = [];
    try {
        searchResults = await (model.delegate(prisma)).findMany({
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
        throw new CustomError("0383", "InternalError", req.session.languages, { objectType });
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
    prisma,
    req,
}: Omit<ReadManyHelperProps<Input>, "addSupplemental">): Promise<ReadManyAsFeedResult> {
    const readManyResult = await readManyHelper({
        additionalQueries,
        addSupplemental: false,
        info,
        input,
        objectType,
        prisma,
        req,
    });
    const format = ModelMap.get(objectType).format;
    const nodes = readManyResult.edges.map(({ node }: any) =>
        modelToGql(node, toPartialGqlInfo(info, format.gqlRelMap, req.session.languages, true))) as any[];
    return {
        pageInfo: readManyResult.pageInfo,
        nodes,
    };
}

/**
 * Helper function for reading many embeddable objects in a single line.
 * Cursor-based search. Supports pagination, sorting, and filtering by string.
 * NOTE: Permissions queries should be passed into additionalQueries
 * @returns Paginated search result
 */
export async function readManyWithEmbeddingsHelper<Input extends { [x: string]: any }>({
    additionalQueries,
    addSupplemental = true,
    info,
    input,
    objectType,
    prisma,
    req,
    visibility = VisibilityType.Public,
}: ReadManyHelperProps<Input>): Promise<PaginatedSearchResult> {
    const userData = getUser(req.session);
    const model = ModelMap.get(objectType);
    const desiredTake = getDesiredTake(input.take, req.session.languages, objectType);
    const embedResults = await findTags({ //TODO support more than just tags
        limit: desiredTake,
        offset: 0, //TODO support offset
        prisma,
        searchString: input.searchString ?? "",
        sortOption: input.sortBy ?? model.search?.defaultSort,
        thresholdBookmarks: 0,
        thresholdDistance: 2,
    });
    const partialInfo = toPartialGqlInfo(info, model.format.gqlRelMap, req.session.languages, true);
    const select = selectHelper(partialInfo);
    const searchResults = await prisma.tag.findMany({
        where: { id: { in: embedResults.map(t => t.id) } },
        ...select,
    });
    //TODO validate that the user has permission to read all of the results, including relationships
    // Return formatted for GraphQL
    let formattedNodes = searchResults.map(n => modelToGql(n, partialInfo as PartialGraphQLInfo));
    formattedNodes = await addSupplementalFields(prisma, userData, formattedNodes, partialInfo);
    // Reorder nodes to match the order of the embedResults
    const formattedNodesMap = new Map(formattedNodes.map(n => [n.id, n]));
    return {
        __typename: `${model.__typename}SearchResult` as const,
        pageInfo: {
            __typename: "PageInfo" as const,
            hasNextPage: false,
            endCursor: null,
        },
        edges: embedResults.map(t => ({
            __typename: `${model.__typename}Edge` as const,
            cursor: t.id,
            node: formattedNodesMap.get(t.id),
        })),
    };
}
