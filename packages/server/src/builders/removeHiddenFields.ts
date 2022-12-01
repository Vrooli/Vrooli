/**
 * Removes a list of fields from a GraphQL object. Useful when additional fields 
 * are added to the Prisma select to calculate supplemental fields, but should never 
 * be returned to the client.
 * 
 * NOTE: Supports dot notation through recursion
 */
export const removeHiddenFields = <T extends { [x: string]: any }>(
    obj: T,
    fields: string[] | undefined
): T => {
    if (!fields) return obj;
    // Initialize result
    let result: any = {};
    // Iterate over object
    for (const [key, value] of Object.entries(obj)) {
        // If key is in fields, skip
        if (fields.includes(key)) continue;
        // If value is an object
        if (typeof value === 'object') {
            // Find nested fields
            const nestedFields = fields.filter((field) => field.startsWith(`${key}.`)).map((field) => field.replace(`${key}.`, ''));
            // If there are nested fields, recurse
            if (nestedFields.length > 0) {
                // Recurse
                result[key] = removeHiddenFields(value, nestedFields);
            }
            // Otherwise, just add the value
            else {
                result[key] = value;
            }
        }
        // Otherwise, add to result
        else result[key] = value;
    }
    return result;
}