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
    if (!updated || !original) return fields.reduce((acc, field) => ({ ...acc, [field]: undefined }), {}) as any;
    const changedFields = fields.reduce((acc, field) => ({ ...acc, [field]: updated[field] !== original[field] ? updated[field] : undefined }), {});
    if (!primary) return changedFields as any;
    return { ...changedFields, [primary]: original[primary] } as any;
}
