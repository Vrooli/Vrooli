/**
 * Helper function for converting GraphQL count fields to Prisma relationship counts
 * @param obj - GraphQL-shaped object
 * @param countFields - List of GraphQL field names (e.g. ['commentsCount', reportsCount']) 
 * that correspond to Prisma relationship counts (e.g. { _count: { comments: true, reports: true } })
 */
export const addCountFields = (obj: any, countFields: { [x: string]: true } | undefined): any => {
    if (!countFields) return obj;
    // Create result object
    const result: any = {};
    // Iterate over count map
    for (const key of Object.keys(countFields)) {
        if (obj[key]) {
            if (!obj._count) obj._count = {};
            // Relationship name is the count field without the 'Count' suffix
            const value = key.slice(0, -5);
            obj._count[value] = true;
            delete obj[key];
        }
    }
    return {
        ...obj,
        ...result,
    };
};
