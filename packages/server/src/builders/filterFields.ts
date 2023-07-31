/**
 * Filters excluded fields from an object
 * @param data The object to filter
 * @param excludes The fields to exclude
 */
export const filterFields = (data: { [x: string]: any }, excludes: string[]): { [x: string]: any } => {
    // Create result object
    const converted: { [x: string]: any } = {};
    // Loop through object's keys
    Object.keys(data).forEach((key) => {
        // If key is not in excludes, add to result
        if (!excludes.some(e => e === key)) {
            converted[key] = data[key];
        }
    });
    return converted;
};
