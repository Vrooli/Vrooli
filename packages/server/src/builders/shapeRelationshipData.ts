import { isObject } from "@vrooli/shared";

/**
 * Filters excluded fields from an object
 * @param data The object to filter
 * @param excludes The fields to exclude
 */
export function filterFields(data: object, excludes: string[]): object {
    if (!isObject(data)) return {};
    // Create result object
    const converted: object = {};
    // Loop through object's keys
    Object.keys(data).forEach((key) => {
        // If key is not in excludes, add to result
        if (!excludes.some(e => e === key)) {
            converted[key] = data[key];
        }
    });
    return converted;
}

/**
 * Helper method to shape Prisma connect, disconnect, create, update, and delete data
 * 
 * Examples when "isOneToOne" is false (the default):
 *  - '123' => [{ id: '123' }]
 *  - { id: '123' } => [{ id: '123' }]
 *  - { name: 'John' } => [{ name: 'John' }]
 *  - ['123', '456'] => [{ id: '123' }, { id: '456' }]
 * 
 * Examples when "isOneToOne" is true:
 * - '123' => { id: '123' }
 * - { id: '123' } => { id: '123' }
 * - { name: 'John' } => { name: 'John' }
 * - ['123', '456'] => { id: '123' }
 * 
 * @param data The data to shape
 * @param excludes The fields to exclude from the shape
 * @param isOneToOne Whether the data is one-to-one (i.e. a single object)
 */
export function shapeRelationshipData<IsOneToOne extends boolean>(
    data: unknown,
    excludes: string[] = [],
    isOneToOne: IsOneToOne = false as IsOneToOne,
): IsOneToOne extends true ? object : object[] {
    function shapeAsMany(data: unknown): object[] {
        if (Array.isArray(data)) {
            return data.map(e => {
                if (isObject(e)) {
                    return filterFields(e, excludes);
                } else {
                    return { id: e };
                }
            });
        } else if (isObject(data)) {
            return [filterFields(data, excludes)];
        } else {
            return [{ id: data }];
        }
    }
    // Shape as if "isOneToOne" is fasel
    const result = shapeAsMany(data);
    // Then if "isOneToOne" is true, return the first element
    if (isOneToOne) {
        return (result.length > 0 ? result[0] : {}) as IsOneToOne extends true ? object : object[];
    }
    return result;
}
