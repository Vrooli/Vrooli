import { VisibilityType } from "@local/shared";
import { getUser } from "../auth";
import { addSupplementalFields, modelToGql, selectHelper, toPartialGqlInfo } from "../builders";
import { PaginatedSearchResult, PartialGraphQLInfo } from "../builders/types";
import { CustomError } from "../events";
import { ObjectMap } from "../models/base";
import { findTags } from "../utils";
import { ReadManyHelperProps } from "./types";

const DEFAULT_TAKE = 20;
const MAX_TAKE = 100;

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
    const userData = getUser(req);
    const model = ObjectMap[objectType];
    if (!model) throw new CustomError("0487", "InternalError", req.languages, { objectType });
    // Check take limit
    if (Number.isInteger(input.take) && input.take > MAX_TAKE) {
        throw new CustomError("0488", "InternalError", req.languages, { objectType, take: input.take });
    }
    const embedResults = await findTags({ //TODO support more than just tags
        limit: Number.isInteger(input.take) ? input.take : DEFAULT_TAKE,
        offset: 0, //TODO support offset
        prisma,
        searchString: input.searchString ?? "",
        sortOption: input.sortBy ?? model.search?.defaultSort,
        thresholdBookmarks: 0,
        thresholdDistance: 2,
    });
    const partialInfo = toPartialGqlInfo(info, model.format.gqlRelMap, req.languages, true);
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

