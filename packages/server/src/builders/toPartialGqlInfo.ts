import { GqlModelType } from "@local/shared";
import { CustomError } from "../events/error";
import { GqlRelMap, ModelLogicType } from "../models/types";
import { LRUCache } from "../utils/lruCache";
import { injectTypenames } from "./injectTypenames";
import { ApiEndpointInfo } from "./types";

// Cache results of each `info` conversion, so we only have to 
// do it once. This improves performance a bit
const cache = new LRUCache<string, ApiEndpointInfo>(1000, 250_000);

/**
 * Converts API endpoint info object into a shape that's more easily usable by Prisma. 
 * This typically means adding typenames and removing any endpoint-specific shapes 
 * like pageInfo and edges.
 * @param info API endpoint info object, or result of this function
 * @param gqlRelMap Map of relationship names to typenames
 * @param throwIfNotPartial Throw error if info is not partial
 * @returns Partial Prisma select. This can be passed into the function again without changing the result.
 */
export function toPartialGqlInfo<
    Typename extends `${GqlModelType}`,
    GqlModel extends ModelLogicType["GqlModel"],
    PrismaModel extends ModelLogicType["PrismaModel"],
    ThrowErrorIfNotPartial extends boolean
>(
    info: ApiEndpointInfo,
    gqlRelMap: GqlRelMap<Typename, GqlModel, PrismaModel>,
    throwIfNotPartial: ThrowErrorIfNotPartial = false as ThrowErrorIfNotPartial,
): ThrowErrorIfNotPartial extends true ? ApiEndpointInfo : (ApiEndpointInfo | undefined) {
    // Return undefined if info not set
    if (!info) {
        if (throwIfNotPartial)
            throw new CustomError("0345", "InternalError");
        return undefined as any;
    }
    // Check if cached
    const cacheKey = JSON.stringify(info);
    const cachedResult = cache.get(cacheKey);
    if (cachedResult !== undefined) {
        return cachedResult;
    }
    // Find select fields in info object
    let select = info;
    // If fields are in the shape of a paginated search query, convert to a Prisma select object
    if (Object.prototype.hasOwnProperty.call(select, "pageInfo") && Object.prototype.hasOwnProperty.call(select, "edges")) {
        select = (select as { edges: { node: any } }).edges.node;
    }
    // If fields are in the shape of a comment thread search query, convert to a Prisma select object
    else if (Object.prototype.hasOwnProperty.call(select, "endCursor") && Object.prototype.hasOwnProperty.call(select, "totalThreads") && Object.prototype.hasOwnProperty.call(select, "threads")) {
        select = (select as { threads: { comment: any } }).threads.comment;
    }
    // Inject type fields
    select = injectTypenames(select, gqlRelMap);
    if (!select)
        throw new CustomError("0346", "InternalError");
    // Cache result
    cache.set(cacheKey, select);
    return select;
}
