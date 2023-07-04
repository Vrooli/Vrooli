/**
 * Searches for the first non-null field in an object that matches one of the specified fields.
 *
 * @param obj - The object to search.
 * @param fieldsToCheck - The list of field names to check.
 * @returns An array with the field name and its value if found, or `undefined` if none of the fields exist in the object.
 */
export const findFirstRel = (obj: Record<string, any>, fieldsToCheck: string[]): [string | undefined, any | undefined] => {
    // Loop through each field in the list of fields to check
    for (const fieldName of fieldsToCheck) {
        // Get the value of the current field from the object
        const value = obj[fieldName];
        // If the value is not null or undefined, return the field name and value as an array
        if (value !== null && value !== undefined) {
            return [fieldName, value];
        }
    }
    // If no non-null field is found, return undefined
    return [undefined, undefined];
};
