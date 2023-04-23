import { getUser } from "../auth";
import { addSupplementalFields, combineQueries, modelToGql, selectHelper, toPartialGqlInfo } from "../builders";
import { CustomError, logger } from "../events";
import { getSearchStringQuery } from "../getters";
import { ObjectMap } from "../models";
import { SearchMap } from "../utils";
import { SortMap } from "../utils/sortMap";
const DEFAULT_TAKE = 20;
const MAX_TAKE = 100;
export async function readManyHelper({ additionalQueries, addSupplemental = true, info, input, objectType, prisma, req, }) {
    const userData = getUser(req);
    const model = ObjectMap[objectType];
    if (!model)
        throw new CustomError("0349", "InternalError", req.languages, { objectType });
    const partialInfo = toPartialGqlInfo(info, model.format.gqlRelMap, req.languages, true);
    partialInfo.id = true;
    const searcher = model.search;
    if (Number.isInteger(input.take) && input.take > MAX_TAKE) {
        throw new CustomError("0389", "InternalError", req.languages, { objectType, take: input.take });
    }
    const searchQuery = (input.searchString && searcher?.searchStringQuery) ? getSearchStringQuery({ objectType: model.__typename, searchString: input.searchString }) : undefined;
    const customQueries = [];
    if (searcher) {
        for (const field of Object.keys(searcher.searchFields)) {
            if (input[field] !== undefined) {
                customQueries.push(SearchMap[field](input[field], userData, model.__typename));
            }
        }
    }
    if (searcher?.customQueryData) {
        customQueries.push(searcher.customQueryData(input, userData));
    }
    const where = combineQueries([additionalQueries, searchQuery, ...customQueries]);
    const orderBy = SortMap[input.sortBy ?? searcher.defaultSort] ?? undefined;
    const select = selectHelper(partialInfo);
    let searchResults;
    try {
        searchResults = await model.delegate(prisma).findMany({
            where,
            orderBy,
            take: Number.isInteger(input.take) ? input.take : 25,
            skip: input.after ? 1 : undefined,
            cursor: input.after ? {
                id: input.after,
            } : undefined,
            ...select,
        });
    }
    catch (error) {
        logger.error("readManyHelper: Failed to find searchResults", { trace: "0383", error, objectType, ...select, where, orderBy });
        throw new CustomError("0383", "InternalError", req.languages, { objectType });
    }
    let paginatedResults;
    if (searchResults.length > 0) {
        const cursor = searchResults[searchResults.length - 1].id;
        const hasNextPage = await model.delegate(prisma).findMany({
            take: 1,
            cursor: {
                id: cursor,
            },
        });
        paginatedResults = {
            __typename: `${model.__typename}SearchResult`,
            pageInfo: {
                __typename: "PageInfo",
                hasNextPage: hasNextPage.length > 0,
                endCursor: cursor,
            },
            edges: searchResults.map((result) => ({
                __typename: `${model.__typename}Edge`,
                cursor: result.id,
                node: result,
            })),
        };
    }
    else {
        paginatedResults = {
            __typename: `${model.__typename}SearchResult`,
            pageInfo: {
                __typename: "PageInfo",
                endCursor: null,
                hasNextPage: false,
            },
            edges: [],
        };
    }
    if (!addSupplemental)
        return paginatedResults;
    let formattedNodes = paginatedResults.edges.map(({ node }) => node);
    formattedNodes = formattedNodes.map(n => modelToGql(n, partialInfo));
    formattedNodes = await addSupplementalFields(prisma, userData, formattedNodes, partialInfo);
    return {
        ...paginatedResults,
        pageInfo: paginatedResults.pageInfo,
        edges: paginatedResults.edges.map(({ node, ...rest }) => ({ node: formattedNodes.shift(), ...rest })),
    };
}
//# sourceMappingURL=readManyHelper.js.map