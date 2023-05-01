import { CustomError } from "../events";
import { GqlRelMap } from "../models/types";
import { resolveGraphQLInfo } from "../utils";
import { injectTypenames } from "./injectTypenames";
import { GraphQLInfo, PartialGraphQLInfo } from "./types";

/**
 * Converts shapes 1 and 2 in a GraphQL to Prisma conversion to shape 2
 * @param info - GraphQL info object, or result of this function
 * @param gqlRelMap - Map of relationship names to typenames
 * @param languages - Preferred languages for error messages
 * @param throwIfNotPartial - Throw error if info is not partial
 * @returns Partial Prisma select. This can be passed into the function again without changing the result.
 */
export const toPartialGqlInfo = <
    GQLObject extends { [x: string]: any },
    PrismaObject extends { [x: string]: any },
    ThrowErrorIfNotPartial extends boolean
>(
    info: GraphQLInfo | PartialGraphQLInfo,
    gqlRelMap: GqlRelMap<GQLObject, PrismaObject>,
    languages: string[],
    throwIfNotPartial: ThrowErrorIfNotPartial = false as ThrowErrorIfNotPartial,
): ThrowErrorIfNotPartial extends true ? PartialGraphQLInfo : (PartialGraphQLInfo | undefined) => {
    // Return undefined if info not set
    if (!info) {
        if (throwIfNotPartial)
            throw new CustomError("0345", "InternalError", languages);
        return undefined as any;
    }
    // Find select fields in info object
    let select;
    const isGraphQLResolveInfo = Object.prototype.hasOwnProperty.call(info, "fieldNodes") && Object.prototype.hasOwnProperty.call(info, "returnType");
    if (isGraphQLResolveInfo) {
        select = resolveGraphQLInfo(JSON.parse(JSON.stringify(info)));
    } else {
        select = info;
    }
    // If fields are in the shape of a paginated search query, convert to a Prisma select object
    if (Object.prototype.hasOwnProperty.call(select, "pageInfo") && Object.prototype.hasOwnProperty.call(select, "edges")) {
        select = select.edges.node;
    }
    // If fields are in the shape of a comment thread search query, convert to a Prisma select object
    else if (Object.prototype.hasOwnProperty.call(select, "endCursor") && Object.prototype.hasOwnProperty.call(select, "totalThreads") && Object.prototype.hasOwnProperty.call(select, "threads")) {
        select = select.threads.comment;
    }
    // Inject type fields
    select = injectTypenames(select, gqlRelMap);
    if (!select)
        throw new CustomError("0346", "InternalError", languages);
    return select;
};
