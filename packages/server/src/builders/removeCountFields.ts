import { exists } from "@shared/utils";

/**
 * Helper function for converting Prisma relationship counts to GraphQL count fields
 * @param obj - Prisma-shaped object
 * @param countFields - List of GraphQL field names (e.g. ['commentsCount', reportsCount']) 
 * that correspond to Prisma relationship counts (e.g. { _count: { comments: true, reports: true } })
 */
export const removeCountFields = (obj: any, countFields: { [x: string]: true } | undefined): any => {
    if (!obj || !countFields) return obj;
    // Create result object
    let result: any = {};
    // If no counts, no reason to continue
    if (!obj._count) return obj;
    // Iterate over count map
    for (const key of Object.keys(countFields)) {
        // Relationship name is the count field without the 'Count' suffix
        const value = key.slice(0, -5);
        if (exists(obj._count[value])) {
            obj[key] = obj._count[value];
        }
    }
    // Make sure to delete _count field
    delete obj._count;
    return {
        ...obj,
        ...result
    }
}