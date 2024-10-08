import { isObject } from "@local/shared";
import { LRUCache } from "../utils/lruCache";
import { isRelationshipObject } from "./isOfType";
import { removeTypenames } from "./removeTypenames";
import { toPartialPrismaSelect } from "./toPartialPrismaSelect";
import { PartialGraphQLInfo, PartialPrismaSelect, PrismaSelect } from "./types";

// Cache results of each `partial` conversion, so we only have to 
// do it once. This improves performance a bit
const SELECT_CACHE_LIMIT_COUNT = 1_000;
const SELECT_CACHE_LIMIT_SIZE_BYTES = 250_000;
const cache = new LRUCache<string, PrismaSelect>(SELECT_CACHE_LIMIT_COUNT, SELECT_CACHE_LIMIT_SIZE_BYTES);

/**
 * Adds "select" to the correct parts of an object to make it a Prisma select. 
 * This function is recursive and idempotent.
 */
export function selPad(fields: object): PrismaSelect {
    // Only pad if fields is an object
    if (!isRelationshipObject(fields)) {
        // Return empty object for invalid/empty fields
        if (fields === null || fields === undefined || Array.isArray(fields)) return {} as PrismaSelect;
        // Anything else (strings, Dates, numbers, etc.) should be returned as true
        return true as unknown as PrismaSelect;
    }
    // Recursively pad
    const converted: object = {};
    Object.keys(fields).forEach((key) => {
        // Check if already padded
        const isAlreadyPadded = key === "select";
        // Recursively pad, even if already padded. 
        // This is because nested relationships may not be padded
        let padded = selPad(fields[key]);
        // If it was padded before, remove top-level "select"
        if (isAlreadyPadded) padded = (padded as unknown as { select: PrismaSelect })["select"];
        // If padded is empty object, skip
        if (isRelationshipObject(padded) && Object.keys(padded).length === 0) return;
        // Add padded to converted
        converted[key] = padded;
    });
    // If converted is empty, return
    if (Object.keys(converted).length === 0) return converted as PrismaSelect;
    // If already padded, return
    if ("select" in converted) return converted as PrismaSelect;
    // Return object with "select" padded
    return { select: converted } as PrismaSelect;
}

/**
 * Converts shapes 2 and 3 of the GraphQL to Prisma conversion to shape 4.
 * @returns Object which can be passed into Prisma select directly
 */
export function selectHelper(partial: PartialGraphQLInfo | PartialPrismaSelect): PrismaSelect | undefined {
    // Check if cached
    const cacheKey = JSON.stringify(partial);
    const cachedResult = cache.get(cacheKey);
    if (cachedResult !== undefined) {
        return cachedResult;
    }
    // Convert partial's special cases (virtual/calculated fields, unions, etc.)
    let modified = toPartialPrismaSelect(partial);
    if (!isObject(modified)) return undefined;
    // Delete type fields
    modified = removeTypenames(modified);
    // Pad every relationship with "select"
    modified = selPad(modified);
    // Cache result
    cache.set(cacheKey, modified as PrismaSelect);
    return modified as PrismaSelect;
}
