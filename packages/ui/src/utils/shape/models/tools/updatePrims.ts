import { DUMMY_ID, uuid } from "@shared/uuid";

/**
 * Helper function for setting a list of primitive fields of an update 
 * shape. If updated is different from original, return updated,
 * otherwise return undefined
 * 
 * NOTE: Due to TypeScript limitations, return type assumes that every field 
 * will be defined, even if it's undefined
 * @param original The original object
 * @param updated The updated object
 * @param primary The primary key of the object, which is always returned
 * @param fields The fields to check, which are returned if they are different
 */
export const updatePrims = <T, K extends keyof T, PK extends keyof T>(
    original: T | null | undefined,
    updated: T | null | undefined,
    primary: PK | null,
    ...fields: K[]
): ({ [F in K]: Exclude<T[F], null | undefined> } & { [F in PK]: T[F] }) => {
    // If no original or updated, return undefined for all fields
    if (!updated || !original) return fields.reduce((acc, field) => ({ ...acc, [field]: undefined }), {}) as any;
    // Create prims
    const changedFields = fields.reduce((acc, field) => ({ ...acc, [field]: updated[field] !== original[field] ? updated[field] : undefined }), {});
    // If no primary key, return changed fields
    if (!primary) return changedFields as any;
    // If primary key is not an ID, return changed fields with primary key
    if (primary !== 'id') return { ...changedFields, [primary]: original[primary] } as any;
    // If primary key is an ID, return changed fields with primary key, and make sure it's not DUMMY_ID
    return { ...changedFields, [primary]: original[primary] === DUMMY_ID ? uuid() : original[primary] } as any;
}
