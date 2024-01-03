import { isObject } from "@local/shared";
import { LRUCache } from "../utils/lruCache";
import { removeTypenames } from "./removeTypenames";
import { selPad } from "./selPad";
import { toPartialPrismaSelect } from "./toPartialPrismaSelect";
import { PartialGraphQLInfo, PartialPrismaSelect, PrismaSelect } from "./types";

// Cache results of each `partial` conversion, so we only have to 
// do it once. This improves performance a bit
const cache = new LRUCache<string, PrismaSelect>(1000, 250_000);

/**
 * Converts shapes 2 and 3 of the GraphQL to Prisma conversion to shape 4.
 * @returns Object which can be passed into Prisma select directly
 */
export const selectHelper = (partial: PartialGraphQLInfo | PartialPrismaSelect): PrismaSelect | undefined => {
    // Check if cached
    const cacheKey = JSON.stringify(partial);
    const cachedResult = cache.get(cacheKey);
    if (cachedResult !== undefined) {
        return cachedResult;
    }
    // Convert partial's special cases (virtual/calculated fields, unions, etc.)
    let modified: { [x: string]: any } = toPartialPrismaSelect(partial);
    if (!isObject(modified)) return undefined;
    // Delete type fields
    modified = removeTypenames(modified);
    // Pad every relationship with "select"
    modified = selPad(modified)
    // Cache result
    cache.set(cacheKey, modified as PrismaSelect);
    return modified as PrismaSelect;
};
