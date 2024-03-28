/**
 * Helper function for combining Prisma queries. This is basically a spread, 
 * but it also combines AND, OR, and NOT queries
 * @param queries Array of query objects to combine
 * @returns Combined query object, with all fields combined
 */
export const combineQueries = (queries: ({ [x: string]: any } | null | undefined)[]): { [x: string]: any } => {
    const combined: { [x: string]: any } = {};
    for (const query of queries) {
        if (!query) continue;
        for (const [key, value] of Object.entries(query)) {
            let currValue = value;
            // If key is AND, OR, or NOT, combine
            if (["AND", "OR", "NOT"].includes(key)) {
                // Value should be an array
                if (!Array.isArray(value)) {
                    currValue = [value];
                }
                // For AND, combine arrays
                if (key === "AND") {
                    combined[key] = key in combined ? [...combined[key], ...currValue] : currValue;
                }
                // For OR and NOT, set as value if none exists
                else if (!(key in combined)) {
                    combined[key] = currValue;
                }
                // Otherwise, combine values using AND. This is because we can't have duplicate keys
                else {
                    // Store temp value 
                    const temp = combined[key];
                    // Delete key
                    delete combined[key];
                    // Add old and new value to AND array
                    combined.AND = [
                        ...(combined.AND || []),
                        { [key]: temp },
                        { [key]: currValue },
                    ];
                }
            }
            // Otherwise, add it or cancel it out if it has an opposite boolean value
            else {
                if (key in combined && typeof combined[key] === "boolean" && typeof currValue === "boolean" && combined[key] !== currValue) {
                    // Cancel out true/false values
                    delete combined[key];
                } else {
                    combined[key] = value;
                }
            }
        }
    }
    return combined;
};
