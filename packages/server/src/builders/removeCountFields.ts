/**
 * Helper function for converting Prisma relationship counts to GraphQL count fields
 * @param obj - Prisma-shaped object
 * @param countFields - List of GraphQL field names (e.g. ['commentsCount', reportsCount']) 
 * that correspond to Prisma relationship counts (e.g. { _count: { comments: true, reports: true } })
 */
export const removeCountFields = (obj: any, countFields: readonly string[] | undefined): any => {
    if (!obj || !countFields) return obj;
    // Create result object
    let result: any = {};
    // If no counts, no reason to continue
    if (!obj._count) return obj;
    // Iterate over count map
    for (const key of countFields) {
        // Relationship name is the count field without the 'Count' suffix
        const value = key.slice(0, -5);
        if (obj._count[value] !== undefined && obj._count[value] !== null) {
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