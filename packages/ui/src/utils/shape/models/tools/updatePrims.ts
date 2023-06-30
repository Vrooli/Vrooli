import { DUMMY_ID, uuid } from "@local/shared";

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
    ...fields: (K | [K, (val: any) => any])[]
): ({ [F in K]: Exclude<T[F], null | undefined> } & { [F in PK]: T[F] }) => {
    // If no original or updated, return undefined for all fields
    if (!updated || !original) {
        return fields.reduce((acc, field) => {
            const key = Array.isArray(field) ? field[0] : field;
            return { ...acc, [key]: undefined };
        }, {}) as any;
    }

    // Create prims
    const changedFields = fields.reduce((acc, field) => {
        const key = Array.isArray(field) ? field[0] : field;
        const value = Array.isArray(field) ? field[1](updated[key]) : updated[key];
        return { ...acc, [key]: value !== original[key] ? value : undefined };
    }, {});

    // If no primary key, return changed fields
    if (!primary) return changedFields as any;

    // If primary key is not an ID, return changed fields with primary key
    if (primary !== "id") return { ...changedFields, [primary]: original[primary] } as any;

    // If primary key is an ID, return changed fields with primary key, and make sure it's not DUMMY_ID
    return { ...changedFields, [primary]: original[primary] === DUMMY_ID ? uuid() : original[primary] } as any;
};
