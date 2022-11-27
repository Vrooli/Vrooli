import { isObject } from "@shared/utils";
import { filterFields } from "./filterFields";

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
export const shapeRelationshipData = (data: any, excludes: string[] = [], isOneToOne: boolean = false): any => {
    const shapeAsMany = (data: any): any => {
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
    let result = shapeAsMany(data);
    // Then if "isOneToOne" is true, return the first element
    if (isOneToOne) {
        if (result.length > 0) {
            result = result[0];
        } else {
            result = {};
        }
    }
    return result;
}