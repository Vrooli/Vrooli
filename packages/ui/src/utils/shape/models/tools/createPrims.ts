/**
 * Helper function for setting a list of primitive fields of a create 
 * shape. Essentially, adds every field that's defined, and converts nulls to 
 * undefined
 * 
 * NOTE: Due to TypeScript limitations, return type assumes that every field 
 * will be defined, even if it's undefined
 */
export const createPrims = <T, K extends keyof T>(
    object: T,
    ...fields: K[]
): { [F in K]: Exclude<T[F], null | undefined> } => {
    return fields.reduce((acc, field) => ({ ...acc, [field]: object[field] !== null ? object[field] : undefined }), {}) as any;
}